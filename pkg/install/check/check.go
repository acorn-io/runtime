package check

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/acorn-io/acorn/pkg/client/term"
	"github.com/acorn-io/acorn/pkg/k8schannel"
	"github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/acorn/pkg/streams"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/restconfig"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	klogv2 "k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CheckResult describes the results of a check, making it human-readable
type CheckResult struct {
	Message string `json:"message"`
	Passed  bool   `json:"passed"`
	Name    string `json:"name"`
}

// PreflightChecks is a list of all checks that are run before the installation.
// They are crictial and will make the installation fail.
func PreflightChecks(ctx context.Context) []CheckResult {
	return RunChecks(ctx,
		CheckRBAC,
		CheckNodesReady,
	)
}

// InFlightChecks is a list of all checks that are run after the installation.
// They are not critical and should not affect the installation process.
func InFlightChecks(ctx context.Context) []CheckResult {

	return RunChecks(ctx,
		CheckDefaultStorageClass,
		CheckIngressCapability,
		CheckExec,
	)
}

// IsFailed is a simple helper function marking a list of check results
// as failed if one or more results show failed status.
func IsFailed(results []CheckResult) bool {
	for _, r := range results {
		if !r.Passed {
			return true
		}
	}
	return false
}

// RunChecks runs a list of checks and returns their results as a list.
func RunChecks(ctx context.Context, checks ...func(ctx context.Context) CheckResult) []CheckResult {
	var results []CheckResult
	for _, check := range checks {
		results = append(results, check(ctx))
	}
	return results
}

// silenceKlog is a helper function to silence klog output which could disturb
// the clean installation logs.
func silenceKlog() {
	klog.SetOutput(io.Discard)
	klogv2.SetOutput(io.Discard)
	utilruntime.ErrorHandlers = nil
}

func CheckExec(ctx context.Context) CheckResult {
	result := CheckResult{
		Name: "Exec",
	}

	silenceKlog()
	cli, err := k8sclient.Default()
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Error creating client: %v", err)
		return result
	}

	// Create new pod
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "acorn-install-check-exec",
			Namespace: system.Namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            "exec",
					Image:           "busybox",
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command: []string{
						"tail",
						"-f",
						"/dev/null",
					},
				},
			},
		},
	}

	if err := cli.Create(ctx, pod); err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Error creating pod: %v", err)
		return result
	}

	defer func() {
		if err := cli.Delete(ctx, pod); err != nil {
			fmt.Printf("Error deleting pod: %v\n", err)
		}
	}()

	// Wait for pod to be ready
	var podList corev1.PodList
	watcher, err := cli.Watch(ctx, &podList, client.InNamespace(system.Namespace), client.MatchingFields{"metadata.name": pod.GetName()})
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Error creating watcher: %v", err)
		return result
	}
	defer watcher.Stop()
	for {
		event := <-watcher.ResultChan()
		if event.Type == watch.Error {
			result.Passed = false
			result.Message = fmt.Sprintf("Error watching pod: %v", event.Object)
			return result
		}
		if event.Type == watch.Added || event.Type == watch.Modified {
			pod := event.Object.(*corev1.Pod)
			if pod.Status.Phase == corev1.PodRunning {
				break
			}
		}
	}

	// Execute command in container
	// had to reimplement this from the client pkg to avoid an import cycle
	cfg, err := restconfig.Default()
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Error creating client: %v", err)
		return result
	}

	dialer, err := k8schannel.NewDialer(cfg, false)
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Error creating dialer: %v", err)
		return result
	}

	cfg.APIPath = "api"
	cfg.GroupVersion = &corev1.SchemeGroupVersion
	cfg.NegotiatedSerializer = scheme.Codecs

	c, err := rest.RESTClientFor(cfg)
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Error creating client: %v", err)
		return result
	}

	req := c.Get().
		Namespace(pod.GetNamespace()).
		Resource("pods").
		Name(pod.GetName()).
		SubResource("exec").
		VersionedParams(
			&corev1.PodExecOptions{Container: pod.Spec.Containers[0].Name, Command: []string{"/bin/sh", "-c", "echo Hello"}, Stderr: true},
			scheme.ParameterCodec,
		)

	conn, err := dialer.DialContext(ctx, req.URL().String(), nil)
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Error dialing for container exec: %v", err)
		return result
	}

	cIO := conn.ToExecIO(false)

	exitCode, err := term.Pipe(cIO, &streams.Streams{Output: streams.Output{Out: io.Discard, Err: io.Discard}, In: os.Stdin})
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Error piping execIO: %v", err)
		return result
	}

	if exitCode != 0 {
		result.Passed = false
		result.Message = fmt.Sprintf("Container replica exec exited with code %d", exitCode)
		return result
	}

	result.Passed = true
	result.Message = "Successfully executed command in container replica"
	return result

}

func CheckIngressCapability(ctx context.Context) CheckResult {
	result := CheckResult{
		Name: "IngressCapability",
	}

	silenceKlog()
	cli, err := k8sclient.Default()
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Error creating client: %v", err)
		return result
	}

	// Create a new Endpoint object
	ep := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "acorn-ingress-test",
			Namespace: system.Namespace,
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: "1.1.1.1",
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Name:     "http",
						Port:     80,
						Protocol: corev1.ProtocolTCP,
					},
				},
			},
		},
	}

	// Create a new Service object
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "acorn-ingress-test",
			Namespace: system.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Port: 80,
			}},
			Selector: map[string]string{
				"app": "acorn-ingress-test",
			},
		},
	}

	// Create a new ingress object
	pt := networkingv1.PathTypeImplementationSpecific
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "acorn-ingress-test",
			Namespace: system.Namespace,
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: "inflight-check.acorn.io",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pt,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "acorn-ingress-test",
											Port: networkingv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Create objects
	if err := cli.Create(ctx, ep); err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Error creating endpoint: %v", err)
		return result
	}
	defer func() {
		if err := cli.Delete(ctx, ep); err != nil && !errors.IsNotFound(err) {
			klog.Errorf("Error deleting endpoint: %v", err)
		}
	}()

	if err := cli.Create(ctx, svc); err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Error creating service: %v", err)
		return result
	}
	defer func() {
		if err := cli.Delete(ctx, svc); err != nil {
			klog.Errorf("Error deleting service: %v", err)
		}
	}()

	if err := cli.Create(ctx, ing); err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Error creating ingress: %v", err)
		return result
	}
	defer func() {
		if err := cli.Delete(ctx, ing); err != nil {
			klog.Errorf("Error deleting ingress: %v", err)
		}
	}()

	// Wait for ingress to be ready, 10s timeout
	ingw := &networkingv1.IngressList{Items: []networkingv1.Ingress{*ing}}
	w, err := cli.Watch(ctx, ingw, client.InNamespace(system.Namespace))
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Error watching ingress: %v", err)
		return result
	}

	nctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	for {
		select {
		case <-nctx.Done():
			result.Passed = false
			result.Message = "Ingress not ready (test timed out)"
			return result
		case f := <-w.ResultChan():
			if f.Type == watch.Modified {
				ing := f.Object.(*networkingv1.Ingress)
				if ing.Status.LoadBalancer.Ingress != nil {
					result.Passed = true
					result.Message = "Ingress is ready"
					return result
				}
			}
		}
	}

}

/*
 * CheckNodesReady checks if all nodes are ready.
 * -> This is a critical check, which "could" affect the installation.
 * TODO: We only need to check if the cluster is operational.
 * -> A single malfunctiuning node should not prevent the installation.
 */
func CheckNodesReady(ctx context.Context) CheckResult {
	result := CheckResult{
		Name: "NodesReady",
	}

	silenceKlog()
	cli, err := k8sclient.Default()
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Error creating client: %v", err)
		return result
	}

	// Try to list cluster nodes
	var nds corev1.NodeList

	if err := cli.List(ctx, &nds); err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Error listing nodes: %v", err)
		return result
	}

	nrdy := 0
	for _, nd := range nds.Items {
		for _, c := range nd.Status.Conditions {
			if c.Type == corev1.NodeReady && c.Status != corev1.ConditionTrue {
				nrdy++
				break
			}
		}
	}

	if nrdy > 0 {
		result.Passed = false
		result.Message = fmt.Sprintf("%d nodes are not ready", nrdy)
	} else {
		result.Passed = true
		result.Message = "All nodes are ready"
	}

	return result
}

/*
 * CheckRBAC checks if the user has the necessary privileges allow
 * Acorn to run. In this case, we check if the user has the rights
 * to create a namespace, which is required for the installation.
 * -> This is a critical check that must be passed for Acorn to be installed.
 */
func CheckRBAC(ctx context.Context) CheckResult {

	result := CheckResult{
		Name: "RBAC",
	}

	silenceKlog()
	cli, err := k8sclient.Default()
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Error creating client: %v", err)
		return result
	}

	// Check if the cluster is authorized to create a namespace
	av := &authorizationv1.SelfSubjectAccessReview{
		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Verb:     "create",
				Group:    "",
				Resource: "namespace",
			},
		},
	}
	if err := cli.Create(ctx, av); err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Error creating SelfSubjectAccessReview to verify AuthZ: %v", err)
	} else {
		result.Passed = av.Status.Allowed
		if av.Status.Allowed {
			result.Message = "User can create namespaces"
		} else {
			result.Message = "User cannot create namespaces"
		}
	}

	return result
}

/*
 * CheckDefaultStorageClass checks if a default storage class is defined.
 * -> This is a non-critical check that "only" affects some features of Acorn.
 */
func CheckDefaultStorageClass(ctx context.Context) CheckResult {
	result := CheckResult{
		Name: "DefaultStorageClass",
	}

	silenceKlog()
	cli, err := k8sclient.Default()
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Error creating client: %v", err)
		return result
	}

	// List registered storageClasses
	var scs storagev1.StorageClassList

	if err := cli.List(ctx, &scs); err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Error listing storage classes: %v", err)
		return result
	}

	// Check if there is a default storageClass
	defaultSc := ""
	for _, sc := range scs.Items {
		for k, v := range sc.Annotations {
			if k == "storageclass.kubernetes.io/is-default-class" && v == "true" {
				defaultSc = sc.Name
				break
			}
		}
		if defaultSc != "" {
			break
		}
	}

	if len(scs.Items) == 0 {
		result.Passed = false
		result.Message = "No storage classes found"
	} else if defaultSc == "" {
		result.Passed = false
		result.Message = fmt.Sprintf("Found %d storage classes, but none are marked as default", len(scs.Items))
	} else {
		result.Passed = true
		result.Message = fmt.Sprintf("Found default storage class %s", defaultSc)
	}

	return result
}
