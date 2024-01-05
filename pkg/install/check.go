package install

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/acorn-io/baaah/pkg/name"
	"github.com/acorn-io/baaah/pkg/randomtoken"
	"github.com/acorn-io/baaah/pkg/restconfig"
	"github.com/acorn-io/baaah/pkg/watcher"
	"github.com/acorn-io/runtime/pkg/client/term"
	"github.com/acorn-io/runtime/pkg/k8schannel"
	"github.com/acorn-io/runtime/pkg/k8sclient"
	"github.com/acorn-io/runtime/pkg/publish"
	"github.com/acorn-io/runtime/pkg/scheme"
	"github.com/acorn-io/runtime/pkg/streams"
	"github.com/acorn-io/runtime/pkg/system"
	"github.com/acorn-io/runtime/pkg/tolerations"
	"github.com/pkg/errors"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	klogv2 "k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/util/storage"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// CheckResult describes the results of a check, making it human-readable
type CheckResult struct {
	Message string `json:"message"`
	Passed  bool   `json:"passed"`
	Name    string `json:"name"`
}

// CheckOptions defines some extra settings for the tests
type CheckOptions struct {
	Client kclient.WithWatch

	// RuntimeImage is required for tests that spin up a pod and that we want to work in airgap environments
	RuntimeImage string `json:"runtimeImage"`

	// IngressClassName is required for tests that check for ingress capabilities
	IngressClassName *string `json:"ingressClassName"`

	// Namespace to override the namespace in which tests are executed (default: acorn-system)
	Namespace *string `json:"namespace"`
}

func (copts *CheckOptions) setDefaults(ctx context.Context) error {
	if copts.Client == nil {
		silenceKlog()
		cli, err := k8sclient.Default()
		if err != nil {
			return err
		}
		copts.Client = cli
	}

	if copts.Namespace == nil {
		ns := system.Namespace
		copts.Namespace = &ns
	}

	if copts.RuntimeImage == "" {
		copts.RuntimeImage = system.DefaultImage()
	}

	if copts.IngressClassName == nil {
		icn, err := publish.IngressClassNameIfNoDefault(ctx, copts.Client)
		if err != nil {
			return err
		}
		copts.IngressClassName = icn
	}

	return nil
}

// PreInstallChecks is a list of all checks that are run before the installation.
// They are critical and will make the installation fail.
func PreInstallChecks(ctx context.Context, opts CheckOptions) []CheckResult {
	return RunChecks(ctx, opts,
		CheckRBAC,
		CheckNodesReady,
	)
}

// PostInstallChecks is a list of all checks that are run after the installation.
// They are not critical and should not affect the installation process.
func PostInstallChecks(ctx context.Context, opts CheckOptions) []CheckResult {
	return RunChecks(ctx, opts,
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
func RunChecks(ctx context.Context, opts CheckOptions, checks ...func(ctx context.Context, opts CheckOptions) CheckResult) []CheckResult {
	var results []CheckResult

	if err := opts.setDefaults(ctx); err != nil {
		return append(results, CheckResult{
			Passed:  false,
			Message: fmt.Sprintf("Error setting default check options: %v", err),
		})
	}
	for _, check := range checks {
		results = append(results, check(ctx, opts))
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

func CheckExec(ctx context.Context, opts CheckOptions) CheckResult {
	result := CheckResult{
		Name: "Exec",
	}

	silenceKlog()

	image := system.DefaultImage()
	if opts.RuntimeImage != "" {
		image = opts.RuntimeImage
	}

	unique, err := randomtoken.Generate()
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Error generating random token: %v", err)
		return result
	}

	objectName := name.SafeConcatName("acorn-check-exec", unique[:8])

	// Create new pod
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objectName,
			Namespace: *opts.Namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            "exec",
					Image:           image,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command: []string{
						"tail",
						"-f",
						"/dev/null",
					},
				},
			},
			Tolerations: []corev1.Toleration{
				{
					Key:      tolerations.WorkloadTolerationKey,
					Operator: corev1.TolerationOpExists,
				},
			},
		},
	}

	if err := opts.Client.Create(ctx, pod); err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Error creating pod: %v", err)
		return result
	}

	defer func() {
		if err := opts.Client.Delete(ctx, pod); err != nil {
			fmt.Printf("Error deleting pod: %v\n", err)
		}
	}()

	_, err = watcher.New[*corev1.Pod](opts.Client).ByObject(ctx, pod, func(pod *corev1.Pod) (bool, error) {
		return pod.Status.Phase == corev1.PodRunning, nil
	})
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Error creating watcher: %v", err)
		return result
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

func CheckIngressCapability(ctx context.Context, opts CheckOptions) CheckResult {
	result := CheckResult{
		Name: "IngressCapability",
	}

	silenceKlog()

	unique, err := randomtoken.Generate()
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Error generating random token: %v", err)
		return result
	}

	objectName := name.SafeConcatName("acorn-check-ingress", unique[:8])

	// Create a new Endpoint object
	ep := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objectName,
			Namespace: *opts.Namespace,
			Labels: map[string]string{
				"app": objectName,
			},
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
			Name:      objectName,
			Namespace: *opts.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Port: 80,
			}},
			Selector: map[string]string{
				"app": objectName,
			},
		},
	}

	// Create a new ingress object
	pt := networkingv1.PathTypeImplementationSpecific
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objectName,
			Namespace: *opts.Namespace,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: opts.IngressClassName,
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
											Name: objectName,
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

	resources := []kclient.Object{ep, svc, ing}

	// Cleanup resources
	defer func() {
		for _, r := range resources {
			// context.Background() so deletion won't get cancelled by parent context
			if err := opts.Client.Delete(context.Background(), r); err != nil && !apierrors.IsNotFound(err) {
				klog.Errorf("error deleting check-ingress resource %s: %v", r.GetName(), err)
			}
		}
	}()

	// Create resources
	for _, r := range resources {
		if err := opts.Client.Create(ctx, r); err != nil {
			result.Passed = false
			result.Message = fmt.Sprintf("Error creating %s: %v", r.GetName(), err)
			return result
		}
	}

	// Wait for ingress to be ready, or timeout after 1 minute
	nctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
	_, err = watcher.New[*networkingv1.Ingress](opts.Client).ByObject(nctx, ing, func(ing *networkingv1.Ingress) (bool, error) {
		return ing.Status.LoadBalancer.Ingress != nil, nil
	})
	if errors.Is(err, context.DeadlineExceeded) {
		result.Passed = false
		result.Message = "Ingress not ready (test timed out after 1 minute)"
		return result
	} else if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Error watching ingress: %v", err)
		return result
	}

	result.Passed = true
	result.Message = "Ingress is ready"
	return result
}

/*
 * CheckNodesReady checks if all nodes are ready.
 * -> This is a critical check, which "could" affect the installation.
 * TODO: We only need to check if the cluster is operational.
 * -> A single malfunctioning node should not prevent the installation.
 */
func CheckNodesReady(ctx context.Context, opts CheckOptions) CheckResult {
	result := CheckResult{
		Name: "NodesReady",
	}

	silenceKlog()

	// Try to list cluster nodes
	var nds corev1.NodeList

	if err := opts.Client.List(ctx, &nds); err != nil {
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
func CheckRBAC(ctx context.Context, opts CheckOptions) CheckResult {
	result := CheckResult{
		Name: "RBAC",
	}

	silenceKlog()

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
	if err := opts.Client.Create(ctx, av); err != nil {
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
func CheckDefaultStorageClass(ctx context.Context, opts CheckOptions) CheckResult {
	result := CheckResult{
		Name: "DefaultStorageClass",
	}

	silenceKlog()

	// List registered storageClasses
	var scs storagev1.StorageClassList

	if err := opts.Client.List(ctx, &scs); err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Error listing storage classes: %v", err)
		return result
	}

	// Check if there is a default storageClass
	defaultSc := ""
	for _, sc := range scs.Items {
		for k, v := range sc.Annotations {
			if k == storage.IsDefaultStorageClassAnnotation && v == "true" {
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
