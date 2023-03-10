package appdefinition

import (
	"fmt"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/baaah/pkg/router"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"strconv"
	"strings"
)

func NetworkPolicy(req router.Request, resp router.Response) error {
	cfg, err := config.Get(req.Ctx, req.Client)
	if err != nil {
		return err
	} else if *cfg.DisableNetworkPolicies {
		return nil
	}

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
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From: []networkingv1.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							labels.AcornAppNamespace: appNamespace,
						},
					}},
				},
			}},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
		},
	})

	// get needed configuration information
	conf, err := config.Get(req.Ctx, req.Client)
	if err != nil {
		return err
	}
	var ingressNamespace, nodeCIDR string
	if conf.IngressControllerNamespace != nil {
		ingressNamespace = *conf.IngressControllerNamespace
	} else {
		ingressNamespace = ""
	}
	if conf.NodeCIDR != nil && *conf.NodeCIDR != "" {
		nodeCIDR = *conf.NodeCIDR
	} else {
		nodeCIDR = "0.0.0.0/0"
	}

	// next, create NetworkPolicies for each container in the app that has a published port
	// these policies allow ingress from anywhere
	for containerName, container := range app.Status.AppSpec.Containers {
		for _, port := range container.Ports {
			if port.Publish {
				if port.Protocol == v1.ProtocolHTTP {
					resp.Objects(buildNetPolForHTTPPublishedPort(
						fmt.Sprintf("%s-%s-%s", strings.ToLower(app.Name), strings.ToLower(containerName), strconv.Itoa(int(port.Port))),
						podNamespace, ingressNamespace, containerName, port.Port))
				} else {
					resp.Objects(buildNetPolForOtherPublishedPort(
						fmt.Sprintf("%s-%s-%s", strings.ToLower(app.Name), strings.ToLower(containerName), strconv.Itoa(int(port.Port))),
						podNamespace, nodeCIDR, containerName, port.Port))
				}
			}
		}
		// create policies for the sidecars as well
		for sidecarName, sidecar := range container.Sidecars {
			for _, port := range sidecar.Ports {
				if port.Publish {
					if port.Protocol == v1.ProtocolHTTP {
						resp.Objects(buildNetPolForHTTPPublishedPort(
							fmt.Sprintf("%s-%s-sidecar-%s-%s", strings.ToLower(app.Name), strings.ToLower(containerName), strings.ToLower(sidecarName), strconv.Itoa(int(port.Port))),
							podNamespace, ingressNamespace, containerName, port.Port))
					} else {
						resp.Objects(buildNetPolForOtherPublishedPort(
							fmt.Sprintf("%s-%s-sidecar-%s-%s", strings.ToLower(app.Name), strings.ToLower(containerName), strings.ToLower(sidecarName), strconv.Itoa(int(port.Port))),
							podNamespace, nodeCIDR, containerName, port.Port))
					}
				}
			}
		}
	}

	return nil
}

func buildNetPolForHTTPPublishedPort(name, namespace, ingressNamespace, containerName string, port int32) *networkingv1.NetworkPolicy {
	var namespaceSelector metav1.LabelSelector
	if ingressNamespace != "" {
		namespaceSelector = metav1.LabelSelector{
			MatchLabels: map[string]string{
				"kubernetes.io/metadata.name": ingressNamespace,
			},
		}
	}

	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					labels.AcornContainerName: containerName,
				},
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From: []networkingv1.NetworkPolicyPeer{{
					PodSelector:       &metav1.LabelSelector{},
					NamespaceSelector: &namespaceSelector,
				}},
				Ports: []networkingv1.NetworkPolicyPort{{
					Port: &intstr.IntOrString{
						IntVal: port,
					}},
				}},
			},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
		},
	}
}

func buildNetPolForOtherPublishedPort(name, namespace, nodeCIDR, containerName string, port int32) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					labels.AcornContainerName: containerName,
				},
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From: []networkingv1.NetworkPolicyPeer{{
					IPBlock: &networkingv1.IPBlock{
						CIDR: nodeCIDR,
					},
				}},
				Ports: []networkingv1.NetworkPolicyPort{{
					Port: &intstr.IntOrString{
						IntVal: port,
					}},
				}},
			},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
		},
	}
}
