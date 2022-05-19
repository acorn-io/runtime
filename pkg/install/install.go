package install

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	kclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/podstatus"
	"github.com/acorn-io/acorn/pkg/version"
	"github.com/acorn-io/acorn/pkg/watcher"
	"github.com/acorn-io/baaah/pkg/restconfig"
	"github.com/rancher/wrangler/pkg/apply"
	"github.com/rancher/wrangler/pkg/merr"
	"github.com/rancher/wrangler/pkg/yaml"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	v1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	InstallImage  = "ghcr.io/acorn-io/acorn"
	DefaultBranch = "main"

	//go:embed *.yaml
	files embed.FS
)

type Options struct {
	OutputFormat string
}

func DefaultImage() string {
	var image = fmt.Sprintf("%s:%s", InstallImage, version.Tag)
	if strings.Contains(image, "v0.0.0") {
		image = fmt.Sprintf("%s:%s", InstallImage, DefaultBranch)
	}
	return image
}

func Install(ctx context.Context, image string, opts *Options) error {
	if opts != nil && opts.OutputFormat != "" {
		return printObject(image, opts.OutputFormat)
	}

	apply, err := newApply(ctx)
	if err != nil {
		return err
	}

	logrus.Info("Installing ClusterRoles")
	if err := applyRoles(apply); err != nil {
		return err
	}

	logrus.Info("Installing APIServer and Controller")
	if err := applyDeployments(image, apply); err != nil {
		return err
	}

	kclient, err := kclient.Default()
	if err != nil {
		return err
	}

	if err := waitAPI(ctx, kclient); err != nil {
		return err
	}

	if err := waitController(ctx, kclient); err != nil {
		return err
	}

	logrus.Info("Installation done")
	return nil
}

func waitDeployment(ctx context.Context, client client.WithWatch, name string) error {
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		_, _ = watcher.New[*corev1.Pod](client).BySelector(childCtx, "acorn-system", labels.SelectorFromSet(map[string]string{
			"app": name,
		}), func(pod *corev1.Pod) (bool, error) {
			status := podstatus.GetStatus(pod)
			if status.Reason != "Running" {
				logrus.Infof("Pod %s/%s: %s", pod.Namespace, pod.Name, status)
			}
			return false, nil
		})
	}()

	_, err := watcher.New[*appsv1.Deployment](client).ByName(ctx, "acorn-system", name, func(dep *appsv1.Deployment) (bool, error) {
		for _, cond := range dep.Status.Conditions {
			if cond.Type == appsv1.DeploymentAvailable {
				logrus.Infof("Deployment acorn-system/%s: %s=%s (%s) %s", name, cond.Type, cond.Status, cond.Reason, cond.Message)
				if cond.Status == corev1.ConditionTrue {
					return true, nil
				}
			}
		}
		return false, nil
	})

	return err
}

func waitController(ctx context.Context, client client.WithWatch) error {
	logrus.Info("Waiting for controller deployment to be available")
	return waitDeployment(ctx, client, "acorn-controller")
}

func waitAPI(ctx context.Context, client client.WithWatch) error {
	logrus.Info("Waiting for API server deployment to be available")
	if err := waitDeployment(ctx, client, "acorn-api"); err != nil {
		return err
	}

	logrus.Info("Waiting for API service to be available")
	_, err := watcher.New[*v1.APIService](client).ByName(ctx, "", "v1.api.acorn.io", func(apiService *v1.APIService) (bool, error) {
		for _, cond := range apiService.Status.Conditions {
			if cond.Type == v1.Available {
				logrus.Infof("APIServer v1.api.acorn.io: %s=%s (%s) %s", cond.Type, cond.Status, cond.Reason, cond.Message)
				if cond.Status == v1.ConditionTrue {
					return true, nil
				}
			}
		}
		return false, nil
	})
	return err
}

func printObject(image, format string) error {
	roles, err := Roles()
	if err != nil {
		return err
	}

	deps, err := Deployments(image)
	if err != nil {
		return err
	}

	if format == "json" {
		m := map[string]interface{}{
			"items": deps,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(m)
	}

	data, err := yaml.Export(append(roles, deps...)...)
	if err != nil {
		return err
	}

	_, err = os.Stdout.Write(data)
	return err
}

func applyDeployments(imageName string, apply apply.Apply) error {
	objs, err := Deployments(imageName)
	if err != nil {
		return err
	}
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
					return fmt.Errorf("insufficient privileges to install into cluster: %w", err)
				}
			}
		}
		return err
	}
	return nil
}

func Deployments(runtimeImage string) ([]runtime.Object, error) {
	apiServerObjects, err := objectsFromFile("apiserver.yaml")
	if err != nil {
		return nil, err
	}

	controllerObjects, err := objectsFromFile("controller.yaml")
	if err != nil {
		return nil, err
	}

	return replaceImage(runtimeImage, append(apiServerObjects, controllerObjects...))
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
