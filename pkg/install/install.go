package install

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/mail"
	"os"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/autoupgrade/validate"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/install/progress"
	"github.com/acorn-io/acorn/pkg/k8sclient"
	labels2 "github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/podstatus"
	"github.com/acorn-io/acorn/pkg/prompt"
	"github.com/acorn-io/acorn/pkg/publish"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/acorn/pkg/term"
	"github.com/acorn-io/acorn/pkg/version"
	"github.com/acorn-io/baaah/pkg/apply"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/acorn-io/baaah/pkg/watcher"
	"github.com/pterm/pterm"
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
	SkipChecks         bool
	OutputFormat       string
	APIServerReplicas  *int
	ControllerReplicas *int
	Config             apiv1.Config
	Progress           progress.Builder
}

func (o *Options) complete() *Options {
	if o == nil {
		o := &Options{}
		return o.complete()
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

func validMailAddress(address string) bool {
	_, err := mail.ParseAddress(address)
	return err == nil
}

func Install(ctx context.Context, image string, opts *Options) error {
	// I don't want these errors on the screen. Probably a better way to do this.
	klog.SetOutput(io.Discard)
	klogv2.SetOutput(io.Discard)
	utilruntime.ErrorHandlers = nil

	if opts.Config.AutoUpgradeInterval != nil {
		if _, err := validate.AutoUpgradeInterval(*opts.Config.AutoUpgradeInterval); err != nil {
			return err
		}
	}

	c, err := k8sclient.Default()
	if err != nil {
		return err
	}

	serverConf, err := config.Incomplete(ctx, c)
	if err != nil {
		return err
	}

	// Require E-Mail address when using Let's Encrypt production
	if opts.Config.LetsEncrypt != nil && *opts.Config.LetsEncrypt == "enabled" {
		agreed := opts.Config.GetLetsEncryptTOSAgree()
		if !agreed && opts.Config.LetsEncryptTOSAgree == nil && serverConf.GetLetsEncryptTOSAgree() {
			agreed = true
		}
		if !agreed {
			ok, err := prompt.Bool("You are choosing to enable Let's Encrypt for TLS certificates. To do so, you must agree to their Terms of Service: https://letsencrypt.org/documents/LE-SA-v1.3-September-21-2022.pdf\nTip: use --lets-encrypt-tos-agree to skip this prompt\nDo you agree to Let's Encrypt TOS?", false)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("you must agree to Let's Encrypt TOS when enabling Let's Encrypt")
			}
			opts.Config.LetsEncryptTOSAgree = &ok
		}
		if opts.Config.LetsEncryptEmail == "" && serverConf.LetsEncryptEmail == "" {
			result, err := pterm.DefaultInteractiveTextInput.WithMultiLine(false).Show("Enter your email address for Let's Encrypt")
			if err != nil {
				return err
			}
			opts.Config.LetsEncryptEmail = result
		}
		pterm.Info.Println("You've enabled automatic TLS certificate provisioning with Let's Encrypt. This can take a few minutes to configure.")
	}

	// Validate E-Mail address provided for Let's Encrypt registration
	if opts.Config.LetsEncryptEmail != "" || (opts.Config.LetsEncrypt != nil && *opts.Config.LetsEncrypt == "enabled") {
		email := opts.Config.LetsEncryptEmail
		if email == "" {
			email = serverConf.LetsEncryptEmail
		}
		if !validMailAddress(email) {
			return fmt.Errorf("invalid email address '%s' provided for Let's Encrypt", opts.Config.LetsEncryptEmail)
		}
	}

	// Validate the non-default http-endpoint-pattern
	if opts.Config.HttpEndpointPattern != nil && *opts.Config.HttpEndpointPattern != "" {
		if err := publish.ValidateEndpointPattern(*opts.Config.HttpEndpointPattern); err != nil {
			return err
		}
	}

	opts = opts.complete()
	if opts.OutputFormat != "" {
		return printObject(image, opts)
	}

	checkOpts := CheckOptions{RuntimeImage: image}
	if !opts.SkipChecks {
		s := opts.Progress.New("Running Pre-install Checks")
		checkResults := PreInstallChecks(ctx, checkOpts)
		if IsFailed(checkResults) {
			msg := "Pre-install checks failed. Use `acorn check` to debug or `acorn install --checks=false` to skip"
			for _, result := range checkResults {
				if !result.Passed {
					msg += fmt.Sprintf("\n%s: %s", result.Name, result.Message)
				}
			}
			s.SuccessWithWarning(msg)
		} else {
			s.Success()
		}
	}

	var installIngressController bool
	if ok, err := config.IsDockerDesktop(ctx, c); err != nil {
		return err
	} else if ok {
		if opts.Config.IngressClassName == nil {
			installIngressController, err = missingIngressClass(ctx, c)
			if err != nil {
				return err
			}
			if installIngressController {
				opts.Config.IngressClassName = &[]string{"traefik"}[0]
			}
		}
	}

	apply, err := newApply(ctx)
	if err != nil {
		return err
	}

	if err := config.Set(ctx, c, &opts.Config); err != nil {
		return err
	}

	s := opts.Progress.New("Installing ClusterRoles")
	if err := applyRoles(ctx, apply); err != nil {
		return s.Fail(err)
	}
	s.Success()

	s = opts.Progress.New(fmt.Sprintf("Installing APIServer and Controller (image %s)", image))
	if err := applyDeployments(ctx, image, *opts.APIServerReplicas, *opts.ControllerReplicas, apply, c); err != nil {
		return s.Fail(err)
	}
	s.Success()

	if installIngressController {
		if err := installTraefik(ctx, opts.Progress, c, apply); err != nil {
			return err
		}
	}

	if err := waitController(ctx, opts.Progress, *opts.ControllerReplicas, image, c); err != nil {
		return err
	}

	if err := waitAPI(ctx, opts.Progress, *opts.APIServerReplicas, image, c); err != nil {
		return err
	}

	if !opts.SkipChecks {
		s = opts.Progress.New("Running Post-install Checks")
		checkResults := PostInstallChecks(ctx, checkOpts)
		if IsFailed(checkResults) {
			msg := "Post-install checks failed. Use `acorn check` to debug or `acorn install --checks=false` to skip"
			for _, result := range checkResults {
				if !result.Passed {
					msg += fmt.Sprintf("\n%s: %s", result.Name, result.Message)
				}
			}
			s.SuccessWithWarning(msg)
		} else {
			s.Success()
		}
	} else {
		s.Success()
	}

	pterm.Success.Println("Installation done")
	return nil
}

func TraefikResources() (result []kclient.Object, _ error) {
	objs, err := objectsFromFile("traefik.yaml")
	if err != nil {
		return nil, err
	}
	for _, obj := range objs {
		m, err := meta.Accessor(obj)
		if err != nil {
			return nil, err
		}

		labels := m.GetLabels()
		if labels == nil {
			labels = map[string]string{}
		}
		labels[labels2.AcornManaged] = "true"
		m.SetLabels(labels)

	}

	return objs, nil
}

func missingIngressClass(ctx context.Context, client kclient.Client) (bool, error) {
	ingressClassList := &networkingv1.IngressClassList{}
	err := client.List(ctx, ingressClassList)
	if err != nil {
		return false, err
	}
	return len(ingressClassList.Items) <= 0, nil
}

func installTraefik(ctx context.Context, p progress.Builder, client kclient.WithWatch, apply apply.Apply) (err error) {
	pb := p.New("Installing Traefik Ingress Controller")
	defer func() {
		_ = pb.Fail(err)
	}()

	objs, err := TraefikResources()
	if err != nil {
		return err
	}

	return apply.WithOwnerSubContext("acorn-install-traefik").WithNamespace(system.Namespace).Apply(ctx, nil, objs...)
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

func AllResources() ([]kclient.Object, error) {
	opts := &Options{}
	return resources(DefaultImage(), opts.complete())
}

func resources(image string, opts *Options) ([]kclient.Object, error) {
	var objs []kclient.Object

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
		m := map[string]any{
			"items": objs,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(m)
	}

	data, err := yaml.Export(typed.MapSlice(objs, func(t kclient.Object) runtime.Object {
		return t
	})...)
	if err != nil {
		return err
	}

	_, err = os.Stdout.Write(data)
	return err
}

func applyDeployments(ctx context.Context, imageName string, apiServerReplicas, controllerReplicas int, apply apply.Apply, c kclient.Client) error {
	// handle upgrade from <= v0.3.x
	if err := resetNamespace(ctx, c); err != nil {
		return err
	}

	objs, err := Namespace()
	if err != nil {
		return err
	}

	deps, err := Deployments(imageName, apiServerReplicas, controllerReplicas)
	if err != nil {
		return err
	}

	objs = append(objs, deps...)
	return apply.Apply(ctx, nil, objs...)
}

func applyRoles(ctx context.Context, apply apply.Apply) error {
	objs, err := Roles()
	if err != nil {
		return err
	}
	err = apply.Apply(ctx, nil, objs...)
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

func Config(cfg apiv1.Config) ([]kclient.Object, error) {
	cfgObj, err := config.AsConfigMap(&cfg)
	if err != nil {
		return nil, err
	}
	return []kclient.Object{cfgObj}, nil
}

func Namespace() ([]kclient.Object, error) {
	return objectsFromFile("namespace.yaml")
}

func Deployments(runtimeImage string, apiServerReplicas, controllerReplicas int) ([]kclient.Object, error) {
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

func replaceReplicas(replicas int, objs []kclient.Object) ([]kclient.Object, error) {
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

func replaceImage(image string, objs []kclient.Object) ([]kclient.Object, error) {
	for _, obj := range objs {
		ustr := obj.(*unstructured.Unstructured)
		if ustr.GetKind() == "Deployment" {
			containers, _, _ := unstructured.NestedSlice(ustr.Object, "spec", "template", "spec", "containers")
			for _, container := range containers {
				container.(map[string]any)["image"] = image
				container.(map[string]any)["env"] = []interface{}{
					map[string]any{
						"name":  "ACORN_IMAGE",
						"value": image,
					},
				}
				if !strings.Contains(image, ":v") {
					container.(map[string]any)["imagePullPolicy"] = "Always"
				}
			}
			if err := unstructured.SetNestedSlice(ustr.Object, containers, "spec", "template", "spec", "containers"); err != nil {
				return nil, err
			}
		}
	}
	return objs, nil
}

func Roles() ([]kclient.Object, error) {
	return objectsFromFile("role.yaml")
}

func newApply(ctx context.Context) (apply.Apply, error) {
	c, err := k8sclient.Default()
	if err != nil {
		return nil, err
	}

	apply := apply.New(c)
	if err != nil {
		return nil, err
	}
	return apply.WithOwnerSubContext("acorn-install"), nil
}

func objectsFromFile(name string) (result []kclient.Object, _ error) {
	f, err := files.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	objs, err := yaml.ToObjects(f)
	if err != nil {
		return nil, err
	}
	return typed.MapSlice(objs, func(obj runtime.Object) kclient.Object {
		return obj.(kclient.Object)
	}), nil
}

func resetNamespace(ctx context.Context, c kclient.Client) error {
	ns := &corev1.Namespace{}
	err := c.Get(ctx, router.Key("", system.Namespace), ns)
	if apierror.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}
	if ns.Labels[apply.LabelHash] == "9df1d588ddd6e2cf9585be17cd3442d14cfa76ca" {
		delete(ns.Labels, apply.LabelHash)
		return c.Update(ctx, ns)
	}
	return nil
}
