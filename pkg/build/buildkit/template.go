package buildkit

import (
	"context"
	"fmt"
	"strconv"

	"github.com/acorn-io/acorn/pkg/install"
	"github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/apply"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetRegistryPort(ctx context.Context, c client.Reader) (int, error) {
	return getRegistryPort(ctx, c)
}

func getRegistryPort(ctx context.Context, c client.Reader) (int, error) {
	var service corev1.Service
	err := c.Get(ctx, client.ObjectKey{Name: system.RegistryName, Namespace: system.Namespace}, &service)
	if err != nil {
		return 0, fmt.Errorf("getting %s/%s service: %w", system.Namespace, system.RegistryName, err)
	}
	for _, port := range service.Spec.Ports {
		if port.Name == system.RegistryName && port.NodePort > 0 {
			return int(port.NodePort), nil
		}
	}

	return 0, fmt.Errorf("failed to find node port for registry %s/%s", system.Namespace, system.RegistryName)
}

func deleteObjects(ctx context.Context) error {
	c, err := k8sclient.Default()
	if err != nil {
		return err
	}

	apply := apply.New(c)
	if err != nil {
		return err
	}
	return apply.
		WithOwnerSubContext("acorn-buildkitd").
		WithPruneGVKs(schema.GroupVersionKind{
			Version: "v1",
			Kind:    "Service",
		}, schema.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "Deployment",
		}, schema.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "Daemonset",
		}).
		Apply(ctx, nil)
}

func applyObjects(ctx context.Context) error {
	c, err := k8sclient.Default()
	if err != nil {
		return err
	}

	apply := apply.New(c)
	if err != nil {
		return err
	}

	err = apply.
		WithOwnerSubContext("acorn-buildkitd").
		Apply(ctx, nil, objects(system.Namespace, install.DefaultImage(), install.DefaultImage())...)
	if err != nil {
		return err
	}

	registryNodePort, err := GetRegistryPort(ctx, c)
	if err != nil {
		return err
	}

	return apply.
		WithOwnerSubContext("acorn-buildkitd").
		Apply(ctx, nil, containerdConfigPathDaemonSet(system.Namespace, install.DefaultImage(), strconv.Itoa(registryNodePort))...)
}

func objects(namespace, buildKitImage, registryImage string) []client.Object {
	return []client.Object{
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      system.RegistryName,
				Namespace: namespace,
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{
						Name:     system.RegistryName,
						Protocol: corev1.ProtocolTCP,
						Port:     int32(system.RegistryPort),
						TargetPort: intstr.IntOrString{
							IntVal: int32(system.RegistryPort),
						},
					},
				},
				Selector: map[string]string{
					"app": system.BuildKitName,
				},
				Type: corev1.ServiceTypeNodePort,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      system.BuildKitName,
				Namespace: namespace,
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": system.BuildKitName,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "buildkitd",
						},
					},
					Spec: corev1.PodSpec{
						EnableServiceLinks: new(bool),
						Containers: []corev1.Container{
							{
								Name: "registry",
								Env: []corev1.EnvVar{
									{
										Name:  "REGISTRY_STORAGE_DELETE_ENABLED",
										Value: "true",
									},
								},
								Image:   registryImage,
								Command: []string{"/usr/local/bin/registry", "serve", "/etc/docker/registry/config.yml"},
								LivenessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										TCPSocket: &corev1.TCPSocketAction{
											Port: intstr.IntOrString{
												IntVal: int32(system.RegistryPort),
											},
										},
									},
									InitialDelaySeconds: 15,
									TimeoutSeconds:      1,
									PeriodSeconds:       20,
									SuccessThreshold:    1,
									FailureThreshold:    3,
								},
								ReadinessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										TCPSocket: &corev1.TCPSocketAction{
											Port: intstr.IntOrString{
												IntVal: int32(system.RegistryPort),
											},
										},
									},
									InitialDelaySeconds: 2,
									TimeoutSeconds:      1,
									PeriodSeconds:       5,
									SuccessThreshold:    1,
									FailureThreshold:    3,
								},
								SecurityContext: &corev1.SecurityContext{
									RunAsUser:                &[]int64{1000}[0],
									RunAsNonRoot:             &[]bool{true}[0],
									ReadOnlyRootFilesystem:   &[]bool{true}[0],
									AllowPrivilegeEscalation: new(bool),
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "registry",
										MountPath: "/var/lib/registry",
									},
								},
							},
							{
								Name:    "buildkitd",
								Image:   buildKitImage,
								Command: []string{"/usr/local/bin/setup-binfmt"},
								Args: []string{
									"--debug",
									"--addr",
									"unix:///run/buildkit/buildkitd.sock",
									"--addr",
									fmt.Sprintf("tcp://0.0.0.0:%d", system.BuildkitPort),
								},
								LivenessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										Exec: &corev1.ExecAction{
											Command: []string{
												"buildctl",
												"debug",
												"workers",
											},
										},
									},
									InitialDelaySeconds: 5,
									PeriodSeconds:       30,
								},
								ReadinessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										Exec: &corev1.ExecAction{
											Command: []string{
												"buildctl",
												"debug",
												"workers",
											},
										},
									},
									InitialDelaySeconds: 2,
									PeriodSeconds:       30,
								},
								SecurityContext: &corev1.SecurityContext{
									Privileged: &[]bool{true}[0],
								},
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: int32(system.BuildkitPort),
									},
								},
							},
						},
						Volumes: []corev1.Volume{
							{
								VolumeSource: corev1.VolumeSource{
									EmptyDir: &corev1.EmptyDirVolumeSource{},
								},
								Name: "registry",
							},
						},
					},
				},
			},
		},
	}
}

func containerdConfigPathDaemonSet(namespace, image, registryServiceNodePort string) []client.Object {
	return []client.Object{
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      system.ContainerdConfigPathName,
				Namespace: namespace,
			},
			Spec: appsv1.DaemonSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"ds": system.ContainerdConfigPathName,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"ds": system.ContainerdConfigPathName,
						},
					},
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{
							{
								Name: "etc",
								VolumeSource: corev1.VolumeSource{
									HostPath: &corev1.HostPathVolumeSource{
										Path: "/etc",
									},
								},
							},
						},
						Containers: []corev1.Container{
							{
								Name: system.ContainerdConfigPathName,
								Command: []string{
									"/usr/local/bin/ds-containerd-config-path-entry",
								},
								Env: []corev1.EnvVar{
									{
										Name:  "REGISTRY_SERVICE_NODEPORT",
										Value: registryServiceNodePort,
									},
								},
								Image: image,
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "etc",
										MountPath: "/etc",
									},
								},
								SecurityContext: &corev1.SecurityContext{
									RunAsUser: &[]int64{0}[0],
								},
							},
						},
					},
				},
			},
		},
	}
}
