package install

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/mail"
	"os"
	"path/filepath"
	"strings"

	"github.com/acorn-io/baaah/pkg/apply"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/acorn-io/baaah/pkg/watcher"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/autoupgrade/validate"
	"github.com/acorn-io/runtime/pkg/buildserver"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/install/progress"
	"github.com/acorn-io/runtime/pkg/k8sclient"
	labels2 "github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/podstatus"
	"github.com/acorn-io/runtime/pkg/prompt"
	"github.com/acorn-io/runtime/pkg/publish"
	"github.com/acorn-io/runtime/pkg/roles"
	"github.com/acorn-io/runtime/pkg/system"
	"github.com/acorn-io/runtime/pkg/term"
	"github.com/acorn-io/z"
	"github.com/pterm/pterm"
	"github.com/rancher/wrangler/pkg/merr"
	"github.com/rancher/wrangler/pkg/yaml"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
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
	//go:embed *.yaml
	files embed.FS
)

type Mode string

type Options struct {
	SkipChecks                          bool
	OutputFormat                        string
	APIServerReplicas                   *int
	APIServerPodAnnotations             map[string]string
	ControllerReplicas                  *int
	ControllerServiceAccountAnnotations map[string]string
	Config                              apiv1.Config
	Progress                            progress.Builder
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
		o.APIServerReplicas = z.Pointer(1)
	}

	if o.ControllerReplicas == nil {
		o.ControllerReplicas = z.Pointer(1)
	}

	if o.Config.UseCustomCABundle == nil {
		o.Config.UseCustomCABundle = new(bool)
	}

	return o
}

func validMailAddress(address string) bool {
	_, err := mail.ParseAddress(address)
	return err == nil
}

func getDevImageOverride(ctx context.Context, c kclient.Client, image string) (string, error) {
	var cm corev1.ConfigMap
	err := c.Get(ctx, router.Key(system.Namespace, system.DevConfigName), &cm)
	if err == nil {
		devImage := cm.Annotations[labels2.DevImageName]
		if devImage != "" {
			logrus.Warnf("Overriding image %s with %s from dev config", image, devImage)
			return devImage, nil
		}
	} else if apierror.IsNotFound(err) {
		return image, nil
	}
	return image, err
}

func Install(ctx context.Context, image string, opts *Options) error {
	// I don't want these errors on the screen. Probably a better way to do this.
	klog.SetOutput(io.Discard)
	klogv2.SetOutput(io.Discard)
	utilruntime.ErrorHandlers = nil

	c, err := k8sclient.Default()
	if err != nil {
		return err
	}

	image, err = getDevImageOverride(ctx, c, image)
	if err != nil {
		return err
	}

	finalConfForValidation, err := config.TestSetGet(ctx, c, &opts.Config)
	if err != nil {
		return err
	}

	if _, err := validate.AutoUpgradeInterval(*finalConfForValidation.AutoUpgradeInterval); err != nil {
		return err
	}

	// Require E-Mail address when using Let's Encrypt production
	if *finalConfForValidation.LetsEncrypt == "enabled" {
		if !*finalConfForValidation.LetsEncryptTOSAgree {
			ok, err := prompt.Bool("You are choosing to enable Let's Encrypt for TLS certificates. To do so, you must agree to their Terms of Service: https://letsencrypt.org/documents/LE-SA-v1.3-September-21-2022.pdf\nTip: use --lets-encrypt-tos-agree to skip this prompt\nDo you agree to Let's Encrypt TOS?", false)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("you must agree to Let's Encrypt TOS when enabling Let's Encrypt")
			}
			opts.Config.LetsEncryptTOSAgree = &ok
		}
		if finalConfForValidation.LetsEncryptEmail == "" {
			result, err := pterm.DefaultInteractiveTextInput.WithMultiLine(false).Show("Enter your email address for Let's Encrypt")
			if err != nil {
				return err
			}
			opts.Config.LetsEncryptEmail = result
		}
		pterm.Info.Println("You've enabled automatic TLS certificate provisioning with Let's Encrypt. This can take a few minutes to configure.")
	}

	// Validate E-Mail address provided for Let's Encrypt registration
	email := opts.Config.LetsEncryptEmail
	if finalConfForValidation.LetsEncryptEmail != "" {
		email = finalConfForValidation.LetsEncryptEmail
	}
	if email != "" || *finalConfForValidation.LetsEncrypt == "enabled" {
		if !validMailAddress(email) {
			return fmt.Errorf("invalid email address '%s' provided for Let's Encrypt", email)
		}
	}

	// Validate the http-endpoint-pattern
	if err := publish.ValidateEndpointPattern(*finalConfForValidation.HttpEndpointPattern); err != nil {
		return err
	}

	if err = validateMemoryArgs(*finalConfForValidation.WorkloadMemoryDefault, *finalConfForValidation.WorkloadMemoryMaximum); err != nil {
		return err
	}

	if err = validateServiceLBAnnotations(finalConfForValidation.ServiceLBAnnotations); err != nil {
		return err
	}

	if err = system.ValidateResources(
		*finalConfForValidation.ControllerMemory, *finalConfForValidation.ControllerCPU,
		*finalConfForValidation.APIServerMemory, *finalConfForValidation.APIServerCPU,
		*finalConfForValidation.RegistryMemory, *finalConfForValidation.RegistryCPU,
		*finalConfForValidation.BuildkitdMemory, *finalConfForValidation.BuildkitdCPU,
		*finalConfForValidation.BuildkitdServiceMemory, *finalConfForValidation.BuildkitdServiceCPU,
	); err != nil {
		return err
	}

	opts = opts.complete()
	if opts.OutputFormat != "" {
		return printObject(image, opts)
	}

	if err := upgradeFromV03(ctx, c); err != nil {
		return err
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
		if finalConfForValidation.IngressClassName == nil {
			installIngressController, err = missingIngressClass(ctx, c)
			if err != nil {
				return err
			}
			if installIngressController {
				opts.Config.IngressClassName = z.Pointer("traefik")
			}
		}
	}

	apply, err := newApply()
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
	if err := applyDeployments(ctx, image, *opts.APIServerReplicas, *opts.ControllerReplicas, *opts.Config.UseCustomCABundle,
		opts.ControllerServiceAccountAnnotations, opts.APIServerPodAnnotations, apply, c); err != nil {
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

	if *finalConfForValidation.InternalRegistryPrefix == "" && *opts.ControllerReplicas > 0 {
		if err := waitRegistry(ctx, opts.Progress, image, c); err != nil {
			return err
		}
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
	}

	pterm.Success.Println("Installation done")
	return nil
}

func validateServiceLBAnnotations(annotations []string) error {
	for _, annotation := range annotations {
		_, _, found := strings.Cut(annotation, "=")
		if !found {
			return fmt.Errorf("invalid annotation %s, must be in the form of key=value", annotation)
		}
	}
	return nil
}

func validateMemoryArgs(defaultMemory int64, maximumMemory int64) error {
	// if default is set to unrestricted memory (0) and max memory is not default will be set to maximum
	if defaultMemory == 0 && maximumMemory != 0 {
		pterm.Info.Println("workload-memory-default is being set to workload-memory-maximum. If this is not intended please specify workload-memory-default to non-zero value")
		return nil
	}
	// if max memory is not set to unlimited default must be smaller than maximum
	if maximumMemory != 0 && defaultMemory > maximumMemory {
		defaultQuantity := resource.NewQuantity(defaultMemory, resource.BinarySI).String()
		maximumQuantity := resource.NewQuantity(maximumMemory, resource.BinarySI).String()

		return fmt.Errorf("invalid memory args: workload-memory-default set to %s which exceeds the workload-memory-maximum of %s",
			defaultQuantity, maximumQuantity)
	}
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

func waitDeployment(ctx context.Context, s progress.Progress, client kclient.WithWatch, imageName, name, namespace string, scale int32) error {
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	eg, _ := errgroup.WithContext(ctx)
	if scale > 0 {
		eg.Go(func() error {
			_, err := watcher.New[*corev1.Pod](client).BySelector(childCtx, namespace, labels.SelectorFromSet(map[string]string{
				"app": name,
			}), func(pod *corev1.Pod) (bool, error) {
				for _, container := range pod.Spec.Containers {
					if container.Image != imageName {
						continue
					}
					status := podstatus.GetStatus(pod)
					if status.Reason == "Running" {
						return true, nil
					}
					s.Infof("Pod %s/%s: %s", pod.Namespace, pod.Name, status)
				}
				return false, nil
			})
			return err
		})
	}

	eg.Go(func() error {
		_, err := watcher.New[*appsv1.Deployment](client).ByName(ctx, namespace, name, func(dep *appsv1.Deployment) (bool, error) {
			for _, cond := range dep.Status.Conditions {
				if cond.Type == appsv1.DeploymentAvailable {
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
	return s.Fail(waitDeployment(ctx, s, client, image, "acorn-controller", system.Namespace, int32(replicas)))
}

func waitRegistry(ctx context.Context, p progress.Builder, image string, client kclient.WithWatch) error {
	s := p.New("Waiting for registry server deployment to be available")
	return s.Fail(waitDeployment(ctx, s, client, image, system.RegistryName, system.ImagesNamespace, 1))
}

func waitAPI(ctx context.Context, p progress.Builder, replicas int, image string, client kclient.WithWatch) error {
	s := p.New("Waiting for API server deployment to be available")
	if err := waitDeployment(ctx, s, client, image, "acorn-api", system.Namespace, int32(replicas)); err != nil {
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
	return resources(system.DefaultImage(), opts.complete())
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

	deps, err := Deployments(image, *opts.APIServerReplicas, *opts.ControllerReplicas, *opts.Config.UseCustomCABundle,
		opts.ControllerServiceAccountAnnotations, opts.APIServerPodAnnotations)
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

func applyDeployments(ctx context.Context, imageName string, apiServerReplicas, controllerReplicas int, useCustomCABundle bool, controllerSAAnnotations, apiPodAnnotations map[string]string, apply apply.Apply, c kclient.Client) error {
	// handle upgrade from <= v0.3.x
	if err := resetNamespace(ctx, c); err != nil {
		return err
	}

	objs, err := Namespace()
	if err != nil {
		return err
	}

	deps, err := Deployments(imageName, apiServerReplicas, controllerReplicas, useCustomCABundle, controllerSAAnnotations, apiPodAnnotations)
	if err != nil {
		return err
	}

	objs = append(objs, deps...)
	return apply.WithNoPrune().Apply(ctx, nil, objs...)
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

func Deployments(runtimeImage string, apiServerReplicas, controllerReplicas int, useCustomCABundle bool, controllerSAAnnotations, apiPodAnnotations map[string]string) ([]kclient.Object, error) {
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

	apiServerObjects, err = replacePodAnnotations(apiPodAnnotations, apiServerObjects)
	if err != nil {
		return nil, err
	}

	controllerObjects, err = replaceReplicas(controllerReplicas, controllerObjects)
	if err != nil {
		return nil, err
	}

	controllerObjects, err = replaceSAAnnotations(controllerSAAnnotations, controllerObjects)
	if err != nil {
		return nil, err
	}

	var objects []kclient.Object
	objects = append(apiServerObjects, controllerObjects...)
	if useCustomCABundle {
		objects, err = replaceCABundleVolumes(objects)
		if err != nil {
			return nil, err
		}
	}

	return replaceImage(runtimeImage, objects)
}

func replacePodAnnotations(annotations map[string]string, objs []kclient.Object) ([]kclient.Object, error) {
	if len(annotations) == 0 {
		return objs, nil
	}

	val := make(map[string]any, len(annotations))
	for k, v := range annotations {
		val[k] = v
	}

	for _, obj := range objs {
		ustr := obj.(*unstructured.Unstructured)
		if ustr.GetKind() == "Deployment" {
			err := unstructured.SetNestedField(ustr.Object, val, "spec", "template", "metadata", "annotations")
			if err != nil {
				return nil, err
			}
		}
	}

	return objs, nil
}

func replaceSAAnnotations(annotations map[string]string, objs []kclient.Object) ([]kclient.Object, error) {
	if len(annotations) == 0 {
		return objs, nil
	}

	val := make(map[string]any, len(annotations))
	for k, v := range annotations {
		val[k] = v
	}

	for _, obj := range objs {
		ustr := obj.(*unstructured.Unstructured)
		if ustr.GetKind() == "ServiceAccount" {
			err := unstructured.SetNestedField(ustr.Object, val, "metadata", "annotations")
			if err != nil {
				return nil, err
			}
		}
	}

	return objs, nil
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
				acornImageEnv := map[string]any{
					"name":  "ACORN_IMAGE",
					"value": image,
				}
				envs := container.(map[string]any)["env"]
				if envs == nil {
					container.(map[string]any)["env"] = []interface{}{acornImageEnv}
				} else {
					container.(map[string]any)["env"] = append(envs.([]interface{}), acornImageEnv)
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

func replaceCABundleVolumes(objs []kclient.Object) ([]kclient.Object, error) {
	for _, obj := range objs {
		ustr := obj.(*unstructured.Unstructured)
		if ustr.GetKind() == "Deployment" {
			containers, _, _ := unstructured.NestedSlice(ustr.Object, "spec", "template", "spec", "containers")
			for _, container := range containers {
				container.(map[string]any)["volumeMounts"] = []interface{}{
					map[string]any{
						"name":      system.CustomCABundleSecretVolumeName,
						"mountPath": filepath.Join(system.CustomCABundleDir, system.CustomCABundleCertName),
						"subPath":   system.CustomCABundleCertName,
						"readOnly":  true,
					},
				}
			}
			if err := unstructured.SetNestedSlice(ustr.Object, containers, "spec", "template", "spec", "containers"); err != nil {
				return nil, err
			}

			volumes := []interface{}{
				map[string]any{
					"name": system.CustomCABundleSecretVolumeName,
					"secret": map[string]any{
						"secretName": system.CustomCABundleSecretName,
					},
				},
			}
			if err := unstructured.SetNestedSlice(ustr.Object, volumes, "spec", "template", "spec", "volumes"); err != nil {
				return nil, err
			}
		}
	}
	return objs, nil
}

func Roles() ([]kclient.Object, error) {
	objs, err := objectsFromFile("role.yaml")
	if err != nil {
		return nil, err
	}
	for _, role := range roles.ClusterRoles() {
		role := role
		objs = append(objs, &role)
	}
	return objs, nil
}

func upgradeFromV03(ctx context.Context, c kclient.Client) error {
	return buildserver.DeleteOld(ctx, c)
}

func newApply() (apply.Apply, error) {
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
