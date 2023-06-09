package networkpolicy

import (
	"fmt"
	"strconv"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/name"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/strings/slices"
)

// ForApp creates a single Kubernetes NetworkPolicy that restricts incoming network traffic
// to all pods in an app, so that they cannot be reached by pods from other projects.
func ForApp(req router.Request, resp router.Response) error {
	cfg, err := config.Get(req.Ctx, req.Client)
	if err != nil {
		return err
	} else if !*cfg.NetworkPolicies {
		return nil
	}

	app := req.Object.(*v1.AppInstance)
	appNamespace := app.Namespace        // this is where the AppInstance lives
	podNamespace := app.Status.Namespace // this is where the app is actually running

	allowedNamespaceSelectors := []networkingv1.NetworkPolicyPeer{{
		NamespaceSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				labels.AcornAppNamespace: appNamespace,
			},
		},
	}}
	for _, namespace := range cfg.AllowTrafficFromNamespace {
		allowedNamespaceSelectors = append(allowedNamespaceSelectors, networkingv1.NetworkPolicyPeer{
			NamespaceSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"kubernetes.io/metadata.name": namespace,
				},
			},
		})
	}

	// create the NetworkPolicy for the whole app
	// this allows traffic only from within the project
	resp.Objects(&networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: podNamespace,
			Labels: map[string]string{
				labels.AcornManaged: "true",
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: labels.Managed(app),
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From: allowedNamespaceSelectors,
			}},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
		},
	})
	return nil
}

// ForIngress creates Kubernetes NetworkPolicies to allow traffic to exposed HTTP ports on
// Acorn apps from the ingress controller. If the ingress controller namespace is not defined, traffic from
// all namespaces will be allowed instead.
func ForIngress(req router.Request, resp router.Response) error {
	cfg, err := config.Get(req.Ctx, req.Client)
	if err != nil {
		return err
	} else if !*cfg.NetworkPolicies {
		return nil
	}

	ingress := req.Object.(*networkingv1.Ingress)
	appName := ingress.Labels[labels.AcornAppName]
	projectName := ingress.Labels[labels.AcornAppNamespace]

	// look for case where the ingress is Acorn-managed but isn't for a specific app
	// this can happen when we install Traefik for Docker Desktop, for example
	if appName == "" || projectName == "" {
		return nil
	}

	// create a mapping of k8s Service names to published port names/numbers
	svcNameToPorts := make(map[string][]networkingv1.ServiceBackendPort)
	for _, rule := range ingress.Spec.Rules {
		for _, path := range rule.HTTP.Paths {
			svcName := path.Backend.Service.Name
			port := path.Backend.Service.Port
			svcNameToPorts[svcName] = append(svcNameToPorts[svcName], port)
		}
	}

	for svcName, ports := range svcNameToPorts {
		// get the Service from k8s
		svc := corev1.Service{}
		err = req.Get(&svc, ingress.Namespace, svcName)
		if err != nil {
			if apierror.IsNotFound(err) {
				// service doesn't exist yet, so return nil
				// this handler will get re-called later
				return nil
			}
			return err
		}

		// This service is either a normal ClusterIP service or an ExternalName service which
		// points to a service in a different namespace (if there are Acorn links involved).
		// If it's an ExternalName, we need to get the service to which it points.
		if svc.Spec.Type == corev1.ServiceTypeExternalName {
			externalName := svc.Spec.ExternalName

			// the ExternalName is in the format <service name>.<namespace>.svc.<cluster domain>
			svcName, rest, ok := strings.Cut(externalName, ".")
			if !ok {
				return fmt.Errorf("failed to parse ExternalName '%s' of svc '%s'", externalName, svc.Name)
			}
			svcNamespace, _, ok := strings.Cut(rest, ".")
			if !ok {
				return fmt.Errorf("failed to parse ExternalName '%s' of svc '%s'", externalName, svc.Name)
			}

			svc = corev1.Service{}
			if err = req.Get(&svc, svcNamespace, svcName); err != nil {
				if apierror.IsNotFound(err) {
					return fmt.Errorf("failed to find service '%s', targeted by ExternalName '%s'", svcName, externalName)
				}
				return err
			}
		}

		netPolName := name.SafeConcatName(projectName, appName, ingress.Name, svcName)

		// build the namespaceSelector for the NetPol
		var namespaceSelector metav1.LabelSelector
		if *cfg.IngressControllerNamespace != "" {
			namespaceSelector = metav1.LabelSelector{
				MatchLabels: map[string]string{
					"kubernetes.io/metadata.name": *cfg.IngressControllerNamespace,
				},
			}
		}

		// build the port slice for the NetPol
		var netPolPorts []networkingv1.NetworkPolicyPort
		for _, port := range ports {
			// try to map this ingress port to a port on the service
			for _, svcPort := range svc.Spec.Ports {
				if (svcPort.Name != "" && svcPort.Name == port.Name) || svcPort.Port == port.Number {
					targetPort := svcPort.TargetPort
					netPolPorts = append(netPolPorts, networkingv1.NetworkPolicyPort{
						Protocol: &[]corev1.Protocol{corev1.ProtocolTCP}[0],
						Port:     &targetPort,
					})
					netPolName = name.SafeConcatName(netPolName, strconv.Itoa(int(targetPort.IntVal)))
				}
			}
		}

		if len(netPolPorts) == 0 {
			logrus.Warnf("found no matching ports between Ingress %s and Service %s in Namespace %s", ingress.Name, svcName, ingress.Namespace)
			continue
		}

		// build the NetPol
		resp.Objects(&networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      netPolName,
				Namespace: svc.Namespace,
				Labels: map[string]string{
					labels.AcornManaged: "true",
				},
			},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{
					MatchLabels: svc.Spec.Selector, // the NetPol will target the same pods that the service targets
				},
				Ingress: []networkingv1.NetworkPolicyIngressRule{{
					From: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &namespaceSelector,
						},
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": "acorn-system",
								},
							},
						},
					},
					Ports: netPolPorts,
				}},
				PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			},
		})
	}

	return nil
}

// ForService creates a Kubernetes NetworkPolicy to allow traffic to published TCP/UDP ports
// on Acorn apps that are exposed with LoadBalancer Services.
func ForService(req router.Request, resp router.Response) error {
	cfg, err := config.Get(req.Ctx, req.Client)
	if err != nil {
		return err
	} else if !*cfg.NetworkPolicies {
		return nil
	}

	service := req.Object.(*corev1.Service)

	// we only care about LoadBalancer services that were created for published TCP/UDP ports
	if service.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return nil
	}

	appName := service.Labels[labels.AcornAppName]
	projectName := service.Labels[labels.AcornAppNamespace]
	containerName := service.Labels[labels.AcornContainerName]

	// look for case where the service is Acorn-managed but isn't for a specific app
	if appName == "" || projectName == "" || containerName == "" {
		return nil
	}

	// build the ipBlock for the NetPol
	ipBlock, err := buildExternalIPBlock(req)
	if err != nil {
		return err
	}

	// build the port slice for the NetPol
	var netPolPorts []networkingv1.NetworkPolicyPort
	for _, port := range service.Spec.Ports {
		proto := port.Protocol
		targetPort := port.TargetPort
		netPolPorts = append(netPolPorts, networkingv1.NetworkPolicyPort{
			Protocol: &proto,
			Port:     &targetPort,
		})
	}

	// build the NetPol
	resp.Objects(&networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.SafeConcatName(projectName, appName, service.Name, containerName),
			Namespace: service.Namespace,
			Labels: map[string]string{
				labels.AcornManaged: "true",
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: service.Spec.Selector, // the NetPol will target the same pods that the service targets
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From: []networkingv1.NetworkPolicyPeer{
					{
						IPBlock: ipBlock,
					},
					{
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"kubernetes.io/metadata.name": "kube-system",
							},
						},
					},
					{
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"kubernetes.io/metadata.name": "acorn-system",
							},
						},
					},
				},
				Ports: netPolPorts,
			}},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
		},
	})

	return nil
}

// ForBuilder creates a Kubernetes NetworkPolicy to allow traffic to the buildkitd pods from the acorn-api only.
// It also only allows outgoing traffic to CoreDNS, the Acorn registry, and the Internet.
func ForBuilder(req router.Request, resp router.Response) error {
	cfg, err := config.Get(req.Ctx, req.Client)
	if err != nil || !*cfg.NetworkPolicies {
		return err
	}

	deployment := req.Object.(*appsv1.Deployment)
	if deployment.Name != "buildkitd" && !strings.HasPrefix(deployment.Name, "bld-") {
		// this is not a builder deployment
		return nil
	}

	podLabels := deployment.Spec.Template.ObjectMeta.Labels
	externalIPBlock, err := buildExternalIPBlock(req)
	if err != nil {
		return err
	}

	// build the NetPol
	resp.Objects(&networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deployment.Name,
			Namespace: deployment.Namespace,
			Labels: map[string]string{
				labels.AcornManaged: "true",
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: podLabels,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From: []networkingv1.NetworkPolicyPeer{{
					// allow access from the acorn-apiserver
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"kubernetes.io/metadata.name": system.Namespace,
						},
					},
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": system.APIServerName,
						},
					},
				}},
			}},
			Egress: []networkingv1.NetworkPolicyEgressRule{{
				To: []networkingv1.NetworkPolicyPeer{
					{
						IPBlock: externalIPBlock, // allow traffic to the Internet
					},
					{
						NamespaceSelector: &metav1.LabelSelector{
							// allow traffic to kube-system for CoreDNS
							MatchLabels: map[string]string{
								"kubernetes.io/metadata.name": "kube-system",
							},
						},
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"k8s-app": "kube-dns",
							},
						},
					},
					{
						PodSelector: &metav1.LabelSelector{
							// allow traffic to the registry for pushing images
							MatchLabels: map[string]string{
								"app": system.RegistryName,
							},
						},
					},
				},
			}},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
		},
	})

	return nil
}

// buildExternalIPBlock creates a NetworkPolicy IPBlock with the CIDR set to 0.0.0.0/0
// and the Except set to the pod CIDRs of the nodes.
func buildExternalIPBlock(req router.Request) (*networkingv1.IPBlock, error) {
	ipBlock := networkingv1.IPBlock{
		CIDR: "0.0.0.0/0",
	}

	// get pod CIDRs from the nodes so that we can only allow IP addresses outside the cluster
	nodes := corev1.NodeList{}
	if err := req.Client.List(req.Ctx, &nodes); err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}
	for _, node := range nodes.Items {
		for _, cidr := range node.Spec.PodCIDRs {
			if !slices.Contains(ipBlock.Except, cidr) {
				ipBlock.Except = append(ipBlock.Except, cidr)
			}
		}
	}

	return &ipBlock, nil
}
