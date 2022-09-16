package helper

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TempNamespace(t *testing.T, client client.Client) *corev1.Namespace {
	t.Helper()

	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "acorn-test-",
			Labels: map[string]string{
				"test.acorn.io/namespace": "true",
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
		_ = client.Delete(ctx, &ns)
	})

	return &ns
}
