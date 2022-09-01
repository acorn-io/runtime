package install

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	uiv1 "github.com/acorn-io/acorn/pkg/apis/ui.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/install/progress"
	k8sclient "github.com/acorn-io/acorn/pkg/k8sclient"
	labels2 "github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/podstatus"
	"github.com/acorn-io/acorn/pkg/term"
	"github.com/acorn-io/acorn/pkg/version"
	"github.com/acorn-io/acorn/pkg/watcher"
	"github.com/acorn-io/baaah/pkg/restconfig"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/pterm/pterm"
	"github.com/rancher/wrangler/pkg/apply"
	"github.com/rancher/wrangler/pkg/merr"
	"github.com/rancher/wrangler/pkg/yaml"
	"golang.org/x/sync/errgroup"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog"
	klogv2 "k8s.io/klog/v2"
	v1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	InstallImage  = "ghcr.io/acorn-io/acorn"
	DefaultBranch = "main"
	devTag        = "v0.0.0-dev"

	//go:embed *.yaml
	files embed.FS
)

type Mode string

type Options struct {
	Checks             *bool
	OutputFormat       string
	APIServerReplicas  *int
	ControllerReplicas *int
	Config             apiv1.Config
	Mode               uiv1.InstallMode
	Progress           progress.Builder
}

func (o *Options) complete() *Options {
	if o == nil {
		o := &Options{}
		return o.complete()
	}

	if o.Mode == "" {
		o.Mode = uiv1.InstallModeBoth
	}

	if o.Progress == nil {
		o.Progress = &term.Builder{}
	}

	if o.APIServerReplicas == nil {
		o.APIServerReplicas = &[]int{1}[0]
	}

	if o.ControllerReplicas == nil {
		o.ControllerReplicas = &[]int{1}[0]
	}

	return o
}

func DefaultImage() string {
	img := os.Getenv("ACORN_IMAGE")
	if img != "" {
		return img
	}
	var image = fmt.Sprintf("%s:%s", InstallImage, version.Tag)
	if version.Tag == devTag {
		image = fmt.Sprintf("%s:%s", InstallImage, DefaultBranch)
	}
	return image
}

func Install(ctx context.Context, image string, opts *Options) error {
	// I don't want these errors on the screen. Probably a better way to do this.
	klog.SetOutput(io.Discard)
	klogv2.SetOutput(io.Discard)
	utilruntime.ErrorHandlers = nil

	opts = opts.complete()
	if opts.OutputFormat != "" {
		return printObject(image, opts)
	}

	checkOpts := CheckOptions{RuntimeImage: image}
	if opts.Checks == nil || *opts.Checks {
		s := opts.Progress.New("Running Preflight Checks")
		if IsFailed(PreflightChecks(ctx, checkOpts)) {
			_ = s.Fail(errors.New("preflight checks failed, use `acorn check` to debug or `acorn install --checks=false` to skip"))
		} else {
			s.Success()
		}
	}

	apply, err := newApply(ctx)
	if err != nil {
		return err
	}

	if opts.Mode.DoConfig() {
		c, err := k8sclient.Default()
		if err != nil {
			return err
		}
		if err := config.Set(ctx, c, &opts.Config); err != nil {
			return err
		}
	}

	if opts.Mode.DoResources() {
		s := opts.Progress.New("Installing ClusterRoles")
		if err := applyRoles(apply); err != nil {
			return s.Fail(err)
		}
		s.Success()

		s = opts.Progress.New(fmt.Sprintf("Installing APIServer and Controller (image %s)", image))
		if err := applyDeployments(image, *opts.APIServerReplicas, *opts.ControllerReplicas, apply); err != nil {
			return s.Fail(err)
		}
		s.Success()

		kclient, err := k8sclient.Default()
		if err != nil {
			return err
		}

		if ok, err := config.IsDockerDesktop(ctx, kclient); err != nil {
			return err
		} else if ok {
			if err := installNginx(ctx, opts.Progress, kclient, apply); err != nil {
				return err
			}
		}

		if err := waitController(ctx, opts.Progress, *opts.ControllerReplicas, image, kclient); err != nil {
			return err
		}

		if err := waitAPI(ctx, opts.Progress, *opts.APIServerReplicas, image, kclient); err != nil {
			return err
		}

	}

	s := opts.Progress.New("Running In-Flight Checks")
	checkresults := InFlightChecks(ctx, checkOpts)
	if IsFailed(checkresults) {
		_ = s.Fail(fmt.Errorf("failed in-flight check(s) may break some features of Acorn"))
	}

	for _, result := range checkresults {
		if !result.Passed {
			pterm.Warning.Printf("%s: %s\n", result.Name, result.Message)
		}
	}

	pterm.Success.Println("Installation done")
	return nil
}

func NGINXResources() (result []kclient.Object, _ error) {
	objs, err := nginxResources()
	if err != nil {
		return nil, err
	}
	return typed.MapSlice(objs, func(obj runtime.Object) kclient.Object {
		return obj.(kclient.Object)
	}), nil
}

func nginxResources() (result []runtime.Object, _ error) {
	objs, err := objectsFromFile("nginx.yaml")
	if err != nil {
		return nil, err
	}
	for _, obj := range objs {
		if obj.GetObjectKind().GroupVersionKind().Kind == "IngressClass" {
			m, err := meta.Accessor(obj)
			if err != nil {
				return nil, err
			}

			anno := m.GetAnnotations()
			if anno == nil {
				anno = map[string]string{}
			}
			anno["ingressclass.kubernetes.io/is-default-class"] = "true"
			m.SetAnnotations(anno)

			labels := m.GetLabels()
			if labels == nil {
				labels = map[string]string{}
			}
			labels[labels2.AcornManaged] = "true"
			m.SetLabels(labels)
		}
	}

	return objs, nil
}

func installNginx(ctx context.Context, p progress.Builder, client kclient.WithWatch, apply apply.Apply) (err error) {
	pb := p.New("Installing NGINX Ingress Controller")
	defer func() {
		_ = pb.Fail(err)
	}()

	ingressClassList := &networkingv1.IngressClassList{}
	err = client.List(ctx, ingressClassList)
	if err != nil {
		return err
	}
	if len(ingressClassList.Items) > 0 {
		return nil
	}

	objs, err := nginxResources()
	if err != nil {
		return err
	}

	return apply.WithSetID("acorn-install-nginx").ApplyObjects(objs...)
}

func waitDeployment(ctx context.Context, s progress.Progress, client kclient.WithWatch, imageName, name string, scale int32) error {
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	eg, _ := errgroup.WithContext(ctx)
	if scale > 0 {
		eg.Go(func() error {
			_, err := watcher.New[*corev1.Pod](client).BySelector(childCtx, "acorn-system", labels.SelectorFromSet(map[string]string{
				"app": name,
			}), func(pod *corev1.Pod) (bool, error) {
				if pod.Spec.Containers[0].Image != imageName {
					return false, nil
				}
				status := podstatus.GetStatus(pod)
				if status.Reason == "Running" {
					return true, nil
				}
				s.Infof("Pod %s/%s: %s", pod.Namespace, pod.Name, status)
				return false, nil
			})
			return err
		})
	}

	eg.Go(func() error {
		_, err := watcher.New[*appsv1.Deployment](client).ByName(ctx, "acorn-system", name, func(dep *appsv1.Deployment) (bool, error) {
			for _, cond := range dep.Status.Conditions {
				if cond.Type == appsv1.DeploymentAvailable {
					//s.Infof("Deployment acorn-system/%s: %s=%s (%s) %s", name, cond.Type, cond.Status, cond.Reason, cond.Message)
					if cond.Status == corev1.ConditionTrue && dep.Generation == dep.Status.ObservedGeneration && dep.Status.UpdatedReplicas == scale && dep.Status.ReadyReplicas == scale {
						return true, nil
					}
				}
			}
			return false, nil
		})
		return err
	})

	return eg.Wait()
}

func waitController(ctx context.Context, p progress.Builder, replicas int, image string, client kclient.WithWatch) error {
	s := p.New("Waiting for controller deployment to be available")
	return s.Fail(waitDeployment(ctx, s, client, image, "acorn-controller", int32(replicas)))
}

func waitAPI(ctx context.Context, p progress.Builder, replicas int, image string, client kclient.WithWatch) error {
	s := p.New("Waiting for API server deployment to be available")
	if err := waitDeployment(ctx, s, client, image, "acorn-api", int32(replicas)); err != nil {
		return s.Fail(err)
	}

	if replicas == 0 {
		return nil
	}

	s.Infof("Waiting for API service to be available")
	_, err := watcher.New[*v1.APIService](client).ByName(ctx, "", "v1.api.acorn.io", func(apiService *v1.APIService) (bool, error) {
		for _, cond := range apiService.Status.Conditions {
			if cond.Type == v1.Available {
				s.Infof("APIServer v1.api.acorn.io: %s=%s (%s) %s", cond.Type, cond.Status, cond.Reason, cond.Message)
				if cond.Status == v1.ConditionTrue {
					return true, nil
				}
			}
		}
		return false, nil
	})
	return s.Fail(err)
}

func AllResources() (result []kclient.Object, _ error) {
	opts := &Options{}
	resources, err := resources(DefaultImage(), opts.complete())
	if err != nil {
		return nil, err
	}
	for _, resource := range resources {
		result = append(result, resource.(kclient.Object))
	}
	return
}

func resources(image string, opts *Options) ([]runtime.Object, error) {
	var objs []runtime.Object

	roles, err := Roles()
	if err != nil {
		return nil, err
	}
	objs = append(objs, roles...)

	namespace, err := Namespace()
	if err != nil {
		return nil, err
	}
	objs = append(objs, namespace...)

	deps, err := Deployments(image, *opts.APIServerReplicas, *opts.ControllerReplicas)
	if err != nil {
		return nil, err
	}
	objs = append(objs, deps...)

	cfgs, err := Config(opts.Config)
	if err != nil {
		return nil, err
	}

	objs = append(objs, cfgs...)
	return objs, nil
}

func printObject(image string, opts *Options) error {
	objs, err := resources(image, opts)
	if err != nil {
		return err
	}

	if opts.OutputFormat == "json" {
		m := map[string]interface{}{
			"items": objs,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(m)
	}

	data, err := yaml.Export(objs...)
	if err != nil {
		return err
	}

	_, err = os.Stdout.Write(data)
	return err
}

func applyDeployments(imageName string, apiServerReplicas, controllerReplicas int, apply apply.Apply) error {
	objs, err := Namespace()
	if err != nil {
		return err
	}

	deps, err := Deployments(imageName, apiServerReplicas, controllerReplicas)
	if err != nil {
		return err
	}

	objs = append(objs, deps...)
	return apply.ApplyObjects(objs...)
}

func applyRoles(apply apply.Apply) error {
	objs, err := Roles()
	if err != nil {
		return err
	}
	err = apply.ApplyObjects(objs...)
	if err != nil {
		if merrs, ok := err.(merr.Errors); ok {
			for _, err := range merrs {
				if apierror.IsForbidden(err) {
					return fmt.Errorf("insufficient privileges to install into the cluster: %w", err)
				}
			}
		}
		return err
	}
	return nil
}

func Config(cfg apiv1.Config) ([]runtime.Object, error) {
	cfgObj, err := config.AsConfigMap(&cfg)
	if err != nil {
		return nil, err
	}
	return []runtime.Object{cfgObj}, nil
}

func Namespace() ([]runtime.Object, error) {
	return objectsFromFile("namespace.yaml")
}

func Deployments(runtimeImage string, apiServerReplicas, controllerReplicas int) ([]runtime.Object, error) {
	apiServerObjects, err := objectsFromFile("apiserver.yaml")
	if err != nil {
		return nil, err
	}

	controllerObjects, err := objectsFromFile("controller.yaml")
	if err != nil {
		return nil, err
	}

	apiServerObjects, err = replaceReplicas(apiServerReplicas, apiServerObjects)
	if err != nil {
		return nil, err
	}

	controllerObjects, err = replaceReplicas(controllerReplicas, controllerObjects)
	if err != nil {
		return nil, err
	}

	return replaceImage(runtimeImage, append(apiServerObjects, controllerObjects...))
}

func replaceReplicas(replicas int, objs []runtime.Object) ([]runtime.Object, error) {
	for _, obj := range objs {
		ustr := obj.(*unstructured.Unstructured)
		if ustr.GetKind() == "Deployment" {
			err := unstructured.SetNestedField(ustr.Object, int64(replicas), "spec", "replicas")
			if err != nil {
				return nil, err
			}
		}
	}
	return objs, nil
}

func replaceImage(image string, objs []runtime.Object) ([]runtime.Object, error) {
	for _, obj := range objs {
		ustr := obj.(*unstructured.Unstructured)
		if ustr.GetKind() == "Deployment" {
			containers, _, _ := unstructured.NestedSlice(ustr.Object, "spec", "template", "spec", "containers")
			for _, container := range containers {
				container.(map[string]interface{})["image"] = image
				if !strings.Contains(image, ":v") {
					container.(map[string]interface{})["imagePullPolicy"] = "Always"
				}
			}
			if err := unstructured.SetNestedSlice(ustr.Object, containers, "spec", "template", "spec", "containers"); err != nil {
				return nil, err
			}
		}
	}
	return objs, nil
}

func Roles() ([]runtime.Object, error) {
	return objectsFromFile("role.yaml")
}

func newApply(ctx context.Context) (apply.Apply, error) {
	cfg, err := restconfig.Default()
	if err != nil {
		return nil, err
	}

	apply, err := apply.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return apply.
		WithContext(ctx).
		WithDynamicLookup().
		WithSetID("acorn-install"), nil
}

func objectsFromFile(name string) ([]runtime.Object, error) {
	f, err := files.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return yaml.ToObjects(f)
}
