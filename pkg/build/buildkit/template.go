package buildkit

import (
	"context"
	"fmt"

	"github.com/ibuildthecloud/baaah/pkg/restconfig"
	"github.com/ibuildthecloud/baaah/pkg/router"
	"github.com/ibuildthecloud/herd/pkg/system"
	"github.com/rancher/wrangler/pkg/apply"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetRegistryPort(ctx context.Context, c router.Getter) (int, error) {
	return getRegistryPort(ctx, router.ToReader(c))
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

func applyObjects(ctx context.Context) error {
	cfg, err := restconfig.Default()
	if err != nil {
		return err
	}
	apply, err := apply.NewForConfig(cfg)
	if err != nil {
		return err
	}
	return apply.
		WithContext(ctx).
		WithDynamicLookup().
		WithSetID("herd-buildkitd").
		ApplyObjects(objects(system.Namespace, system.BuildkitImage, system.RegistryImage)...)
}

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
								Env: []corev1.EnvVar{
									{
										Name:  "REGISTRY_STORAGE_DELETE_ENABLED",
										Value: "true",
									},
								},
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
