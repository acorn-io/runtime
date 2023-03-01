package controller_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	kclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/acorn-io/acorn/integration/helper"
	_ "k8s.io/client-go/tools/remotecommand"
)

func TestReadinessProbeOnController(t *testing.T) {
	if os.Getenv("TEST_ACORN_CONTROLLER") != "external" {
		t.Skipf("TEST_ACORN_CONTROLLER != external, skipping")
	}

	client := helper.MustReturn(kclient.DefaultInterface)

	pods, err := client.CoreV1().Pods("acorn-system").List(context.Background(), metav1.ListOptions{})
	assert.NoError(t, err, "could not get pods, or no pods in namespace acorn-system")
	var foundPod *v1.Pod
	var names []string
	for _, pod := range pods.Items {
		if strings.HasPrefix(pod.Name, "acorn-controller") {
			foundPod = &pod
			break
		}
		names = append(names, pod.Name)
	}
	if foundPod == nil {
		t.Fatal(fmt.Errorf("could not find acorn controller, only found %s", names))
	}
	assert.NotNil(t, foundPod.Spec.Containers[0].ReadinessProbe, "missing readiness probe on controller")
	for _, condition := range foundPod.Status.Conditions {
		if string(condition.Type) == "Ready" {
			assert.Equal(t, "True", string(condition.Status), "controller pod is not ready")
		}
	}
}
