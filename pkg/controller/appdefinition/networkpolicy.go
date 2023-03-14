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
	"regexp"
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

	ingressNamespace := *cfg.IngressControllerNamespace
	podCIDR := *cfg.PodCIDR
	if podCIDR != "" {
		if err = validateCIDR(podCIDR); err != nil {
			return err
		}
	}

	// next, create NetworkPolicies for each container and job in the app that has a published port
	for containerName, container := range app.Status.AppSpec.Containers {
		buildNetPolsForContainer(app.Name, containerName, podNamespace, ingressNamespace, podCIDR, container, resp)
	}
	for jobName, job := range app.Status.AppSpec.Jobs {
		buildNetPolsForContainer(app.Name, jobName, podNamespace, ingressNamespace, podCIDR, job, resp)
	}

	return nil
}

func buildNetPolsForContainer(appName, containerName, podNamespace, ingressNamespace, podCIDR string, container v1.Container, resp router.Response) {
	for _, port := range container.Ports {
		if port.Publish {
			if port.Protocol == v1.ProtocolHTTP {
				resp.Objects(buildNetPolForHTTPPublishedPort(
					fmt.Sprintf("%s-%s-%s", strings.ToLower(appName), strings.ToLower(containerName), strconv.Itoa(int(port.Port))),
					podNamespace, ingressNamespace, containerName, port.Port))
			} else {
				resp.Objects(buildNetPolForOtherPublishedPort(
					fmt.Sprintf("%s-%s-%s", strings.ToLower(appName), strings.ToLower(containerName), strconv.Itoa(int(port.Port))),
					podNamespace, podCIDR, containerName, port.Port))
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
						podNamespace, podCIDR, containerName, port.Port))
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
func buildNetPolForOtherPublishedPort(name, namespace, podCIDR, containerName string, port int32) *networkingv1.NetworkPolicy {
	ipBlock := networkingv1.IPBlock{
		CIDR: "0.0.0.0/0",
	}
	if podCIDR != "" {
		ipBlock.Except = []string{podCIDR}
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

func validateCIDR(cidr string) error {
	fmtString := "ERROR: configured Pod CIDR '%s' is not valid"

	// check with regex to make sure it is in the right format:
	regex := regexp.MustCompile(`^([0-9]{1,3}\.){3}[0-9]{1,3}/[0-9]{1,2}$`)
	if !regex.MatchString(cidr) {
		return fmt.Errorf(fmtString, cidr)
	}

	// verify that each number is within the proper range (0-255 for the IP part, and 0-32 for the subnet part)
	pieces := strings.Split(cidr, "/")
	// check the length to be extra safe in case there is some edge case that is incorrect but matches the regex
	if len(pieces) != 2 {
		return fmt.Errorf(fmtString+" (SHOULD NOT HAPPEN)", cidr)
	}
	address := pieces[0]
	subnet := pieces[1]

	for _, segment := range strings.Split(address, ".") {
		segmentInt, err := strconv.Atoi(segment)
		if err != nil {
			return fmt.Errorf(fmtString+" (SHOULD NOT HAPPEN)", cidr)
		}
		if segmentInt < 0 || segmentInt > 255 {
			return fmt.Errorf(fmtString, cidr)
		}
	}

	subnetInt, err := strconv.Atoi(subnet)
	if err != nil {
		return fmt.Errorf(fmtString+" (SHOULD NOT HAPPEN)", cidr)
	}
	if subnetInt < 0 || subnetInt > 32 {
		return fmt.Errorf(fmtString, cidr)
	}

	return nil
}
