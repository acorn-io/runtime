package imagesystem

import (
	"fmt"
	"path/filepath"

	"github.com/acorn-io/acorn/pkg/system"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func BuilderObjects(name, namespace, forNamespace, buildKitImage, pub, privKey, builderUID, forwardAddress string, useCustomCabundle bool) []client.Object {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:     system.BuildKitName,
					Protocol: corev1.ProtocolTCP,
					Port:     int32(system.BuildkitPort),
					TargetPort: intstr.IntOrString{
						IntVal: int32(system.BuildkitPort),
					},
				},
			},
			Selector: map[string]string{
				"app": name,
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"pub":  []byte(pub),
			"priv": []byte(privKey),
		},
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": name,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "acorn-builder",
					EnableServiceLinks: new(bool),
					Containers: []corev1.Container{
						{
							Name:    "buildkitd",
							Image:   buildKitImage,
							Command: []string{"/usr/local/bin/setup-binfmt"},
							Args: []string{
								"--debug",
								"--addr",
								"unix:///run/buildkit/buildkitd.sock",
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
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "socket",
									MountPath: "/run/buildkit",
								},
							},
						},
						{
							Name:    "service",
							Image:   buildKitImage,
							Command: []string{"acorn", "--debug", "--debug-level=9"},
							Env: []corev1.EnvVar{
								{
									Name:  "ACORN_BUILD_SERVER_UUID",
									Value: builderUID,
								},
								{
									Name:  "ACORN_BUILD_SERVER_NAMESPACE",
									Value: forNamespace,
								},
								{
									Name:  "ACORN_BUILD_SERVER_FORWARD_SERVICE",
									Value: forwardAddress + fmt.Sprintf(":%d", system.RegistryPort),
								},
								{
									Name: "ACORN_BUILD_SERVER_PUBLIC_KEY",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: name,
											},
											Key: "pub",
										},
									},
								},
								{
									Name: "ACORN_BUILD_SERVER_PRIVATE_KEY",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: name,
											},
											Key: "priv",
										},
									},
								},
							},
							Args: []string{
								"build-server",
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.IntOrString{
											IntVal: int32(system.BuildkitPort),
										},
									},
								},
								InitialDelaySeconds: 2,
								PeriodSeconds:       5,
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: int32(system.BuildkitPort),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "socket",
									MountPath: "/run/buildkit",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "socket",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

	pdb := &policyv1.PodDisruptionBudget{
		ObjectMeta: deployment.ObjectMeta,
		Spec: policyv1.PodDisruptionBudgetSpec{
			Selector: deployment.Spec.Selector,
			MaxUnavailable: &intstr.IntOrString{
				Type:   intstr.String,
				StrVal: "25%",
			},
		},
	}

	if useCustomCabundle {
		for i := range deployment.Spec.Template.Spec.Containers {
			deployment.Spec.Template.Spec.Containers[i].VolumeMounts = append(deployment.Spec.Template.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
				Name:      system.CustomCABundleSecretVolumeName,
				MountPath: filepath.Join(system.CustomCABundleDir, system.CustomCABundleCertName),
				SubPath:   system.CustomCABundleCertName,
				ReadOnly:  true,
			})
		}
		deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: system.CustomCABundleSecretVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: system.CustomCABundleSecretName,
				},
			},
		})
	}
	return []client.Object{secret, service, deployment, pdb}
}
