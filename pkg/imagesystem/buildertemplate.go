package imagesystem

import (
	"fmt"
	"path/filepath"

	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/system"
	"github.com/acorn-io/runtime/pkg/tolerations"
	"github.com/acorn-io/z"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func BuilderObjects(name, namespace, forNamespace, buildKitImage, pub, privKey, builderUID, forwardAddress string, cfg *apiv1.Config) []client.Object {
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
				MatchLabels: labels.ManagedByApp(namespace, name, "app", name),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels.ManagedByApp(namespace, name, "app", name),
				},
				Spec: corev1.PodSpec{
					PriorityClassName:  system.AcornPriorityClass,
					ServiceAccountName: "acorn-builder",
					EnableServiceLinks: new(bool),
					Containers: []corev1.Container{
						{
							Name:    "buildkitd",
							Image:   buildKitImage,
							Command: []string{"/usr/local/bin/setup-binfmt"},
							Args: []string{
								"--addr",
								"unix:///run/buildkit/buildkitd.sock",
							},
							Resources: system.ResourceRequirementsFor(*cfg.BuildkitdMemory, *cfg.BuildkitdCPU),
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
								Privileged: z.P(true),
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: system.BuildkitPort,
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
							Command: []string{"acorn"},
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
							Resources: system.ResourceRequirementsFor(*cfg.BuildkitdServiceMemory, *cfg.BuildkitdServiceCPU),
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/ping",
										Port: intstr.IntOrString{IntVal: system.BuildkitPort},
									},
								},
								InitialDelaySeconds: 2,
								PeriodSeconds:       5,
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: system.BuildkitPort,
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
					Tolerations: []corev1.Toleration{
						{
							Key:      tolerations.WorkloadTolerationKey,
							Operator: corev1.TolerationOpExists,
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

	svc := &v1.ServiceInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: system.ImagesNamespace,
			Labels: map[string]string{
				labels.AcornAppName:      name,
				labels.AcornAppNamespace: system.ImagesNamespace,
			},
		},
		Spec: v1.ServiceInstanceSpec{
			AppName:      name,
			AppNamespace: system.ImagesNamespace,
			ContainerLabels: map[string]string{
				"app": name,
			},
			Labels: labels.ManagedByApp(system.ImagesNamespace, name),
			Ports: []v1.PortDef{
				{
					Port:     8080,
					Protocol: v1.ProtocolHTTP,
					Publish:  *cfg.PublishBuilders,
				},
			},
		},
	}

	if *cfg.UseCustomCABundle {
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
	return []client.Object{secret, deployment, pdb, svc}
}
