package appdefinition

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/install"
	"github.com/acorn-io/acorn/pkg/ports"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	name2 "github.com/rancher/wrangler/pkg/name"
	"golang.org/x/exp/maps"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func addRouters(appInstance *v1.AppInstance, resp router.Response) error {
	routers, err := toRouters(appInstance)
	if err != nil {
		return err
	}
	resp.Objects(routers...)
	return nil
}

func toRouters(appInstance *v1.AppInstance) (result []kclient.Object, _ error) {
	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Routers) {
		routerObjects, err := toRouter(appInstance, entry.Key, entry.Value)
		if err != nil {
			return nil, err
		}
		result = append(result, routerObjects...)
	}
	return result, nil
}

func toRouter(appInstance *v1.AppInstance, routerName string, router v1.Router) (result []kclient.Object, _ error) {
	if ports.IsLinked(appInstance, routerName) || len(router.Routes) == 0 {
		return nil, nil
	}

	conf, confName := toNginxConf(routerName, router)

	podLabels := routerLabels(appInstance, router, routerName)
	deploymentLabels := routerLabels(appInstance, router, routerName)
	matchLabels := routerSelectorMatchLabels(appInstance, routerName)
	maps.Copy(podLabels, ports.ToRouterLabels(appInstance, routerName))

	deploymentAnnotations := routerAnnotations(appInstance, router, routerName)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        routerName,
			Namespace:   appInstance.Status.Namespace,
			Labels:      deploymentLabels,
			Annotations: deploymentAnnotations,
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
					TerminationGracePeriodSeconds: &[]int64{5}[0],
					EnableServiceLinks:            new(bool),
					Containers: []corev1.Container{
						{
							Name:    "nginx",
							Image:   install.DefaultImage(),
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
					ServiceAccountName: routerName,
				},
			},
		},
	}

	if appInstance.Spec.Stop != nil && *appInstance.Spec.Stop {
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
	}, nil
}

func toNginxConf(routerName string, router v1.Router) (string, string) {
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
		buf.WriteString(" {\n  proxy_pass ")
		buf.WriteString("http://")
		buf.WriteString(route.TargetServiceName)
		buf.WriteString(":")
		buf.WriteString(strconv.Itoa(port))
		buf.WriteString(";\n}\n")
		if route.PathType == v1.PathTypePrefix && !strings.HasSuffix(route.Path, "/") {
			buf.WriteString("location ")
			buf.WriteString(route.Path)
			buf.WriteString("/")
			buf.WriteString(" {\n  proxy_pass ")
			buf.WriteString("http://")
			buf.WriteString(route.TargetServiceName)
			buf.WriteString(":")
			buf.WriteString(strconv.Itoa(port))
			buf.WriteString(";\n}\n")
		}
		if route.PathType == v1.PathTypePrefix && route.Path == "/" {
			buf.WriteString("location ")
			buf.WriteString("/")
			buf.WriteString(" {\n  proxy_pass ")
			buf.WriteString("http://")
			buf.WriteString(route.TargetServiceName)
			buf.WriteString(":")
			buf.WriteString(strconv.Itoa(port))
			buf.WriteString(";\n}\n")
		}
	}
	buf.WriteString("}\n")

	conf := buf.String()
	hash := sha256.Sum256([]byte(conf))
	return conf, name2.SafeConcatName(routerName, hex.EncodeToString(hash[:])[:8])
}
