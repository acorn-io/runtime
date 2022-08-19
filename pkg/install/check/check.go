package check

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/baaah/pkg/restconfig"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
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
func PreflightChecks() []CheckResult {
	return RunChecks(CheckNodesReady, CheckRBAC)
}

// InFlightChecks is a list of all checks that are run after the installation.
// They are not critical and should not affect the installation process.
func InFlightChecks() []CheckResult {
	checks := []func() CheckResult{
		CheckDefaultStorageClass,
	}

	// Some debugging test
	if os.Getenv("ACORN_INSTALL_FAIL_CHECKS") == "true" {
		checks = append(checks, func() CheckResult {
			return CheckResult{Name: "FailTest", Passed: false, Message: "This is a test failure"}
		})
	}

	return RunChecks(checks...)
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
func RunChecks(checks ...func() CheckResult) []CheckResult {
	var results []CheckResult
	for _, check := range checks {
		results = append(results, check())
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

// newClient is a helper function to quickly create a new k8s client instance
func newClient() (client.WithWatch, error) {
	cfg, err := restconfig.Default()
	if err != nil {
		return nil, err
	}

	return k8sclient.New(cfg)
}

/*
 * CheckNodesReady checks if all nodes are ready.
 * -> This is a critical check, which "could" affect the installation.
 * TODO: We only need to check if the cluster is operational.
 * -> A single malfunctiuning node should not prevent the installation.
 */
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

/*
 * CheckRBAC checks if the user has the necessary privileges allow
 * Acorn to run. In this case, we check if the user has the rights
 * to create a namespace, which is required for the installation.
 * -> This is a critical check that must be passed for Acorn to be installed.
 */
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

/*
 * CheckDefaultStorageClass checks if a default storage class is defined.
 * -> This is a non-critical check that "only" affects some features of Acorn.
 */
func CheckDefaultStorageClass() CheckResult {
	result := CheckResult{
		Name: "DefaultStorageClass",
	}

	silenceKlog()
	cli, err := newClient()
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Error creating client: %v", err)
		return result
	}

	// List registered storageClasses
	var scs storagev1.StorageClassList

	if err := cli.List(context.Background(), &scs); err != nil {
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
