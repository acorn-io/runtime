package registry

import (
	"fmt"

	"github.com/acorn-io/acorn/pkg/system"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func objects(namespace, buildKitImage, registryImage string) []runtime.Object {
	return []runtime.Object{
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		},
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
						Containers: []corev1.Container{
							{
								Name: "registry",
								//Env: []corev1.EnvVar{
								//	{
								//		Name:  "REGISTRY_AUTH",
								//		Value: "htpasswd",
								//	},
								//	{
								//		Name:  "REGISTRY_AUTH_HTPASSWD_REALM",
								//		Value: "Registry Realm",
								//	},
								//	{
								//		Name:  "REGISTRY_AUTH_HTPASSWD_PATH",
								//		Value: "/etc/registry/auth/htpasswd",
								//	},
								//},
								Image: registryImage,
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
								Name:  "buildkitd",
								Image: buildKitImage,
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
