package appdefinition

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/acorn-io/baaah/pkg/typed"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/pdb"
	"github.com/acorn-io/runtime/pkg/ports"
	"github.com/acorn-io/runtime/pkg/publicname"
	"github.com/acorn-io/runtime/pkg/system"
	"github.com/acorn-io/runtime/pkg/tolerations"
	"github.com/acorn-io/z"
	name2 "github.com/rancher/wrangler/pkg/name"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func toRouters(ctx context.Context, c kclient.Client, appInstance *v1.AppInstance) (result []kclient.Object, _ error) {
	cfg, err := config.Get(ctx, c)
	if err != nil {
		return nil, err
	}

	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Routers) {
		routerObjects, err := toRouter(appInstance, entry.Key, entry.Value, cfg.InternalClusterDomain)
		if err != nil {
			return nil, err
		}
		result = append(result, routerObjects...)
	}
	return result, nil
}

func toRouter(appInstance *v1.AppInstance, routerName string, router v1.Router, internalClusterDomain string) (result []kclient.Object, _ error) {
	if ports.IsLinked(appInstance, routerName) || len(router.Routes) == 0 {
		return nil, nil
	}

	conf, confName := toNginxConf(internalClusterDomain, appInstance.Status.Namespace, routerName, router)

	podLabels := routerLabels(appInstance, router, routerName, labels.AcornAppPublicName, publicname.Get(appInstance))
	deploymentLabels := routerLabels(appInstance, router, routerName)
	matchLabels := routerSelectorMatchLabels(appInstance, routerName)

	deploymentAnnotations := routerAnnotations(appInstance, router, routerName)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        routerName,
			Namespace:   appInstance.Status.Namespace,
			Labels:      deploymentLabels,
			Annotations: typed.Concat(deploymentAnnotations, map[string]string{labels.AcornConfigHashAnnotation: appInstance.Status.AppStatus.Routers[routerName].ConfigHash}),
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: matchLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podLabels,
					Annotations: deploymentAnnotations,
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: z.Pointer[int64](10),
					EnableServiceLinks:            new(bool),
					Containers: []corev1.Container{
						{
							Name:    "nginx",
							Image:   system.DefaultImage(),
							Command: []string{"/docker-entrypoint.sh"},
							Args: []string{
								"nginx",
								"-g",
								"daemon off;",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "conf",
									ReadOnly:  true,
									MountPath: "/etc/nginx/conf.d/nginx.conf",
									SubPath:   "config",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 8080,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.IntOrString{
											IntVal: 8080,
										},
									},
								},
							},
							Lifecycle: &corev1.Lifecycle{
								PreStop: &corev1.LifecycleHandler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"/bin/sh",
											"-c",
											"sleep 5 && /usr/sbin/nginx -s quit",
										},
									},
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "conf",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: confName,
									},
								},
							},
						},
					},
					Tolerations: []corev1.Toleration{
						{
							Key:      tolerations.WorkloadTolerationKey,
							Operator: corev1.TolerationOpExists,
						},
					},
					ServiceAccountName: routerName,
				},
			},
		},
	}

	if z.Dereference(appInstance.Spec.Stop) {
		dep.Spec.Replicas = new(int32)
	}

	return []kclient.Object{
		dep,
		&corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      confName,
				Namespace: dep.Namespace,
			},
			Data: map[string]string{
				"config": conf,
			},
		},
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:        routerName,
				Namespace:   appInstance.Status.Namespace,
				Labels:      deploymentLabels,
				Annotations: deploymentAnnotations,
			},
		},
		pdb.ToPodDisruptionBudget(dep),
	}, nil
}

func toNginxConf(internalClusterDomain, namespace, routerName string, router v1.Router) (string, string) {
	buf := &strings.Builder{}
	buf.WriteString("server {\nlisten 8080;\n")
	for _, route := range router.Routes {
		if route.TargetServiceName == "" || route.Path == "" {
			continue
		}
		port := 80
		if route.TargetPort != 0 {
			port = route.TargetPort
		}
		buf.WriteString("location ")
		buf.WriteString("= ")
		buf.WriteString(route.Path)
		buf.WriteString(" {\n  set $backend_servers ")
		buf.WriteString(route.TargetServiceName)
		buf.WriteString(".")
		buf.WriteString(namespace)
		buf.WriteString(".")
		buf.WriteString(internalClusterDomain)
		buf.WriteString(";\n  proxy_pass http://$backend_servers:")
		buf.WriteString(strconv.Itoa(port))
		buf.WriteString(";\n  proxy_set_header X-Forwarded-Host $http_host;")
		buf.WriteString("\n}\n")
		if route.PathType == v1.PathTypePrefix && !strings.HasSuffix(route.Path, "/") {
			buf.WriteString("location ")
			buf.WriteString(route.Path)
			buf.WriteString("/ {\n  set $backend_servers ")
			buf.WriteString(route.TargetServiceName)
			buf.WriteString(".")
			buf.WriteString(namespace)
			buf.WriteString(".")
			buf.WriteString(internalClusterDomain)
			buf.WriteString(";\n  proxy_pass http://$backend_servers:")
			buf.WriteString(strconv.Itoa(port))
			buf.WriteString(";\n  proxy_set_header X-Forwarded-Host $http_host;")
			buf.WriteString("\n}\n")
		}
		if route.PathType == v1.PathTypePrefix && route.Path == "/" {
			buf.WriteString("location / {\n  set $backend_servers ")
			buf.WriteString(route.TargetServiceName)
			buf.WriteString(".")
			buf.WriteString(namespace)
			buf.WriteString(".")
			buf.WriteString(internalClusterDomain)
			buf.WriteString(";\n  proxy_pass http://$backend_servers:")
			buf.WriteString(strconv.Itoa(port))
			buf.WriteString(";\n  proxy_set_header X-Forwarded-Host $http_host;")
			buf.WriteString("\n}\n")
		}
	}
	buf.WriteString("}\n")

	conf := buf.String()
	hash := sha256.Sum256([]byte(conf))
	return conf, name2.SafeConcatName(routerName, hex.EncodeToString(hash[:])[:8])
}
