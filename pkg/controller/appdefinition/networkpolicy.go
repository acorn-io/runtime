package appdefinition

import (
	"fmt"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/baaah/pkg/merr"
	"github.com/acorn-io/baaah/pkg/router"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"net"
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
	appNamespace := app.Namespace        // this is where the AppInstance lives
	podNamespace := app.Status.Namespace // this is where the app is actually running

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

	ingressNamespace := *cfg.IngressControllerNamespace
	podCIDRs := cfg.PodCIDRs

	// make sure the podCIDRs are valid
	var errs []error
	for _, cidr := range podCIDRs {
		if cidr != "" {
			_, _, err = net.ParseCIDR(cidr)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) > 0 {
		return merr.NewErrors(errs...)
	}

	// next, create NetworkPolicies for each container and job in the app that has a published port
	for containerName, container := range app.Status.AppSpec.Containers {
		buildNetPolsForContainer(app.Name, containerName, podNamespace, ingressNamespace, podCIDRs, container, resp)
	}
	for jobName, job := range app.Status.AppSpec.Jobs {
		buildNetPolsForContainer(app.Name, jobName, podNamespace, ingressNamespace, podCIDRs, job, resp)
	}

	return nil
}

func buildNetPolsForContainer(appName, containerName, podNamespace, ingressNamespace string, podCIDRs []string, container v1.Container, resp router.Response) {
	for _, port := range container.Ports {
		if port.Publish {
			if port.Protocol == v1.ProtocolHTTP {
				resp.Objects(buildNetPolForHTTPPublishedPort(
					fmt.Sprintf("%s-%s-%s", strings.ToLower(appName), strings.ToLower(containerName), strconv.Itoa(int(port.Port))),
					podNamespace, ingressNamespace, containerName, port.Port))
			} else {
				resp.Objects(buildNetPolForOtherPublishedPort(
					fmt.Sprintf("%s-%s-%s", strings.ToLower(appName), strings.ToLower(containerName), strconv.Itoa(int(port.Port))),
					podNamespace, containerName, podCIDRs, port.Port))
			}
		}
	}
	// create policies for the sidecars as well
	for sidecarName, sidecar := range container.Sidecars {
		for _, port := range sidecar.Ports {
			if port.Publish {
				if port.Protocol == v1.ProtocolHTTP {
					resp.Objects(buildNetPolForHTTPPublishedPort(
						fmt.Sprintf("%s-%s-sidecar-%s-%s", strings.ToLower(appName), strings.ToLower(containerName), strings.ToLower(sidecarName), strconv.Itoa(int(port.Port))),
						podNamespace, ingressNamespace, containerName, port.Port))
				} else {
					resp.Objects(buildNetPolForOtherPublishedPort(
						fmt.Sprintf("%s-%s-sidecar-%s-%s", strings.ToLower(appName), strings.ToLower(containerName), strings.ToLower(sidecarName), strconv.Itoa(int(port.Port))),
						podNamespace, containerName, podCIDRs, port.Port))
				}
			}
		}
	}
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

// For now, we lock down published TCP/UDP ports by allowing access from all pods in kube-system
// and all IP addresses that aren't part of the pod CIDR.
// This blocks traffic coming from pods from other projects (since their IPs are in the pod CIDR),
// but it allows traffic coming from klipper pods in kube-system (which might be doing the load balancing),
// and from nodes or load balancers that are from a cloud provider.
func buildNetPolForOtherPublishedPort(name, namespace, containerName string, podCIDRs []string, port int32) *networkingv1.NetworkPolicy {
	ipBlock := networkingv1.IPBlock{
		CIDR: "0.0.0.0/0",
	}
	for _, cidr := range podCIDRs {
		if cidr != "" {
			ipBlock.Except = append(ipBlock.Except, cidr)
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
				From: []networkingv1.NetworkPolicyPeer{
					{
						IPBlock: &ipBlock,
					},
					{
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"kubernetes.io/metadata.name": "kube-system",
							},
						},
					},
				},
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
