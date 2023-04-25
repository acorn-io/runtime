package helper

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TempProject creates a new Kubernetes Namespace object with a generated name starting with "acorn-project-"
// and sets some labels on it. This function then calls the tempCreateNamespaceHelper function to create the Namespace
// using the provided Kubernetes client. The created Namespace object is returned.
func TempProject(t *testing.T, client client.Client) *corev1.Namespace {
	t.Helper()
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			// namespace ends up as "acorn-test-{random chars}"
			GenerateName: "acorn-project-",
			Labels: map[string]string{
				"test.acorn.io/namespace": "true",
				labels.AcornProject:       "true",
			},
		},
	}
	return tempCreateNamespaceHelper(t, client, ns)
}

func NamedTempProject(t *testing.T, client client.Client, name string) *corev1.Namespace {
	t.Helper()
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"test.acorn.io/namespace": "true",
				labels.AcornProject:       "true",
			},
		},
	}
	return tempCreateNamespaceHelper(t, client, ns)
}
