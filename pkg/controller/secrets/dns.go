package secrets

import (
	"fmt"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/publish"
	"github.com/acorn-io/runtime/pkg/system"
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func HandleDNSSecret(req router.Request, resp router.Response) error {
	sec := req.Object.(*corev1.Secret)

	domain := string(sec.Data["domain"])
	if domain == "" {
		return fmt.Errorf("acorn dns secret is misconfigured, missing domain")
	}

	cfg, err := config.Get(req.Ctx, req.Client)
	if err != nil {
		return err
	}

	if !slices.Contains(cfg.ClusterDomains, domain) {
		return nil
	}

	ingressClassName := cfg.IngressClassName
	if ingressClassName == nil || *ingressClassName == "" {
		ingressClassName, err = publish.IngressClassNameIfNoDefault(req.Ctx, req.Client)
		if err != nil {
			return err
		}
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      system.DNSServiceName,
			Namespace: system.Namespace,
			Labels: map[string]string{
				labels.AcornManaged: "true",
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Port: 80,
			}},
			Selector: map[string]string{
				"app": system.Namespace,
			},
		},
	}

	pt := netv1.PathTypeImplementationSpecific
	ingress := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: system.DNSIngressName,
			Labels: map[string]string{
				labels.AcornManaged: "true",
			},
		},
		Spec: netv1.IngressSpec{
			IngressClassName: ingressClassName,
			Rules: []netv1.IngressRule{
				{
					Host: system.DNSIngressName + domain,
					IngressRuleValue: netv1.IngressRuleValue{
						HTTP: &netv1.HTTPIngressRuleValue{
							Paths: []netv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pt,
									Backend: netv1.IngressBackend{
										Service: &netv1.IngressServiceBackend{
											Name: system.DNSServiceName,
											Port: netv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	resp.Objects(svc, ingress)
	return nil
}
