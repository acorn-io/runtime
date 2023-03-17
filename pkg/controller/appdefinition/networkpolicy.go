package appdefinition

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/baaah/pkg/router"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NetworkPolicy(req router.Request, resp router.Response) error {
	app := req.Object.(*v1.AppInstance)

	resp.Objects(&networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Status.Namespace,
		},
		Spec: networkingv1.NetworkPolicySpec{
			Ingress:     []networkingv1.NetworkPolicyIngressRule{{}},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
		},
		Status: networkingv1.NetworkPolicyStatus{},
	})

	return nil
}
