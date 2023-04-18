package helper

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/labels"
	"k8s.io/apimachinery/pkg/api/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TempNamespace(t *testing.T, client client.Client) *corev1.Namespace {
	t.Helper()
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			// namespace ends up as "acorn-test-{random chars}"
			GenerateName: "acorn-test-",
			Labels: map[string]string{
				"test.acorn.io/namespace": "true",
				labels.AcornProject:       "true",
			},
		},
	}
	return tempCreateNamespaceHelper(t, client, ns)
}

func tempCreateNamespaceHelper(t *testing.T, client client.Client, namespaceObject corev1.Namespace) *corev1.Namespace {
	t.Helper()
	skipCleanup := false

	ctx := GetCTX(t)
	err := client.Create(ctx, &namespaceObject)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			t.Fatal(err)
		}
		// Namespace already exists.
		// Will want to get the existing namespace object to return it
		// and skip the cleanup function
		t.Logf("Namespace %s already exists, skipping creation.\n", namespaceObject.Name)
		skipCleanup = true

		if err = client.Get(ctx, k8sclient.ObjectKey{Name: namespaceObject.Name}, &namespaceObject); err != nil {
			t.Logf("Could not get object reprenting namespace %s.\n", namespaceObject.Name)
			t.Fatal(err)
		}
	}

	if !skipCleanup {
		t.Cleanup(func() {
			err = client.Delete(ctx, &namespaceObject)
			if err != nil {
				t.Logf("Could not delete namespace %s.\n", namespaceObject.Name)
				t.Fatal(err)
			}
		})
	}

	// Give the system:anonymous user access to get/list this project namespace.
	if err = NamespaceClusterRoleAndBinding(t, GetCTX(t), client, namespaceObject.Name); err != nil {
		t.Fatal(err)
	}

	return &namespaceObject
}
