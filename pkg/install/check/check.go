package check

import (
	"context"
	"fmt"
	"io"

	"github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/baaah/pkg/restconfig"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog"
	klogv2 "k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func PreflightChecks() []CheckResult {
	return Check(CheckNodesReady, CheckRBAC)
}

func IsFailed(results []CheckResult) bool {
	for _, r := range results {
		if !r.Passed {
			return true
		}
	}
	return false
}

type CheckResult struct {
	Message string `json:"message"`
	Passed  bool   `json:"passed"`
	Name    string `json:"name"`
}

func Check(checks ...func() CheckResult) []CheckResult {
	var results []CheckResult
	for _, check := range checks {
		results = append(results, check())
	}
	return results
}

func CheckNodesReady() CheckResult {
	result := CheckResult{
		Name: "NodesReady",
	}

	silenceKlog()
	cli, err := newClient()
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Error creating client: %v", err)
		return result
	}

	// Try to list cluster nodes
	var nds corev1.NodeList

	if err := cli.List(context.Background(), &nds); err != nil {
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

func silenceKlog() {
	klog.SetOutput(io.Discard)
	klogv2.SetOutput(io.Discard)
	utilruntime.ErrorHandlers = nil
}

func newClient() (client.WithWatch, error) {
	cfg, err := restconfig.Default()
	if err != nil {
		return nil, err
	}

	return k8sclient.New(cfg)
}

func CheckRBAC() CheckResult {

	result := CheckResult{
		Name: "RBAC",
	}

	silenceKlog()
	cli, err := newClient()
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
	if err := cli.Create(context.TODO(), av); err != nil {
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
