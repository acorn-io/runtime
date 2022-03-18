package helper

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CleanNamespaces(t *testing.T, c client.Client) error {
	nses := &corev1.NamespaceList{}
	err := c.List(GetCTX(t), nses, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set{
			"test.herd-project.io/namespace": "true",
		}),
	})
	if err != nil {
		return err
	}
	for _, ns := range nses.Items {
		if ns.Status.Phase == corev1.NamespaceActive {
			c.Delete(GetCTX(t), &ns)
		}
	}
	return nil
}

func TempNamespace(t *testing.T, client client.Client) *corev1.Namespace {
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "herd-test-",
			Labels: map[string]string{
				"test.herd-project.io/namespace": "true",
			},
		},
		Spec:   corev1.NamespaceSpec{},
		Status: corev1.NamespaceStatus{},
	}
	ctx := GetCTX(t)
	err := client.Create(ctx, &ns)
	if err != nil {
		t.Fatal(err)
		return nil
	}

	t.Cleanup(func() {
		client.Delete(ctx, &ns)
	})

	return &ns
}
