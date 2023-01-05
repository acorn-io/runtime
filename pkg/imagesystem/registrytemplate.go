package imagesystem

import (
	"github.com/acorn-io/acorn/pkg/system"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func registryService(namespace string) []client.Object {
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
					"app": system.RegistryName,
				},
				Type: corev1.ServiceTypeNodePort,
			},
		},
	}
}

func registryDeployment(namespace, registryImage string) []client.Object {
	return []client.Object{
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      system.RegistryName,
				Namespace: namespace,
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": system.RegistryName,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": system.RegistryName,
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
