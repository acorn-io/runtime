package appdefinition

import (
	"fmt"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/baaah/pkg/router"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"strconv"
)

func NetworkPolicy(req router.Request, resp router.Response) error {
	app := req.Object.(*v1.AppInstance)
	appNamespace := app.ObjectMeta.Namespace // this is where the AppInstance lives
	podNamespace := app.Status.Namespace     // this is where the app is actually running

	// first, create the NetworkPolicy for the whole app
	resp.Objects(&networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: podNamespace,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: labels.Managed(app),
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									labels.AcornAppNamespace: appNamespace,
								},
							},
						},
					},
				},
			},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
		},
	})

	// next, create NetworkPolicies for each container in the app that has a published port (ingress)
	// these policies allow ingress from anywhere, so that all services within and without the cluster can connect
	for containerName, container := range app.Status.AppSpec.Containers {
		for _, port := range container.Ports {
			if port.Publish {
				resp.Objects(&networkingv1.NetworkPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-%s-%s", app.Name, containerName, strconv.Itoa(int(port.Port))),
						Namespace: podNamespace,
					},
					Spec: networkingv1.NetworkPolicySpec{
						PodSelector: metav1.LabelSelector{
							MatchLabels: map[string]string{
								labels.AcornContainerName: containerName,
							},
						},
						Ingress: []networkingv1.NetworkPolicyIngressRule{
							{
								From: []networkingv1.NetworkPolicyPeer{
									{
										PodSelector:       &metav1.LabelSelector{},
										NamespaceSelector: &metav1.LabelSelector{},
									},
								},
								Ports: []networkingv1.NetworkPolicyPort{
									{
										Port: &intstr.IntOrString{
											IntVal: port.Port,
										},
									},
								},
							},
						},
						PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
					},
				})
			}
		}
	}

	return nil
}
