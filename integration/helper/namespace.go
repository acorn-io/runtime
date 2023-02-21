package helper

import (
	"github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/labels"
	"k8s.io/apimachinery/pkg/api/errors"
	"testing"

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
			},
		},
		Spec:   corev1.NamespaceSpec{},
		Status: corev1.NamespaceStatus{},
	}
	return tempCreateHelper(t, client, ns)
}

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
		Spec:   corev1.NamespaceSpec{},
		Status: corev1.NamespaceStatus{},
	}
	return tempCreateHelper(t, client, ns)
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
		Spec:   corev1.NamespaceSpec{},
		Status: corev1.NamespaceStatus{},
	}
	return tempCreateHelper(t, client, ns)
}

func tempCreateHelper(t *testing.T, client client.Client, namespaceObject corev1.Namespace) *corev1.Namespace {
	t.Helper()
	namespaceName := namespaceObject.GetName()
	ctx := GetCTX(t)
	err := client.Create(ctx, &namespaceObject)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			// Namespace already exists.
			// Will want to get the existing namespace object to return it
			// and set a cleaning function to remove it after testing.
			t.Logf("Namespace %s already exists, skipping creation.\n", namespaceName)

			// construct blank object pointer
			var objPointer = corev1.Namespace{}
			// populate objectPointer from call to Get
			err = client.Get(ctx, k8sclient.ObjectKey{Namespace: namespaceName, Name: namespaceName}, &objPointer)
			if err != nil {
				t.Logf("Could not get object reprenting namespace %s.\n", namespaceName)
				t.Fatal(err)
			}
			t.Cleanup(func() {
				namespaceDeleting := objPointer.Name
				err = client.Delete(ctx, &objPointer)
				if err != nil {
					t.Logf("Could not delete namespace %s.\n", namespaceDeleting)
				}
			})
			return &objPointer
		}
		t.Fatal(err)
		return nil
	}

	t.Cleanup(func() {
		namespaceDeleting := namespaceObject.Name
		err = client.Delete(ctx, &namespaceObject)
		if err != nil {
			t.Logf("Could not delete namespace %s.\n", namespaceDeleting)
		}
	})

	return &namespaceObject
}
