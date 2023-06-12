package appdefinition

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"path"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/appdefinition"
	"github.com/acorn-io/acorn/pkg/condition"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/images"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/pdb"
	"github.com/acorn-io/acorn/pkg/ports"
	"github.com/acorn-io/acorn/pkg/publicname"
	"github.com/acorn-io/acorn/pkg/secrets"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/acorn/pkg/volume"
	"github.com/acorn-io/baaah/pkg/apply"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/rancher/wrangler/pkg/data/convert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func FilterLabelsAndAnnotationsConfig(h router.Handler) router.Handler {
	return router.HandlerFunc(func(req router.Request, resp router.Response) error {
		appInstance := req.Object.(*v1.AppInstance)
		cfg, err := config.Get(req.Ctx, req.Client)
		if err != nil {
			return err
		}

		// Note that IgnoreUserLabelsAndAnnotations will not be nil here because
		// config.Get "completes" the config object to fill in default values.
		if *cfg.IgnoreUserLabelsAndAnnotations {
			req.Object = labels.FilterUserDefined(appInstance, cfg.AllowUserLabels, cfg.AllowUserAnnotations)
		}

		return h.Handle(req, resp)
	})
}

func DeploySpec(req router.Request, resp router.Response) (err error) {
	appInstance := req.Object.(*v1.AppInstance)
	status := condition.Setter(appInstance, resp, v1.AppInstanceConditionDefined)
	interpolator := secrets.NewInterpolator(req.Ctx, req.Client, appInstance)

	defer func() {
		if err == nil {
			// if there is an issue with interpolation, we just record it on the condition
			// but still allow the handler to reconcile resp objects. This is to avoid a
			// possible catch 22 where interpolation fails but we can't update the acorn
			// because it's failing. This happens when we refer to an object that has
			// yet to be created
			status.Error(interpolator.Err())
		} else if errors.Is(err, appdefinition.ErrInvalidInput) {
			status.Error(err)
			err = nil
		} else {
			status.Error(err)
		}
	}()

	tag, err := images.GetRuntimePullableImageReference(req.Ctx, req.Client, appInstance.Namespace, appInstance.Status.AppImage.ID)
	if err != nil {
		return err
	}

	pullSecrets, err := NewPullSecrets(req, appInstance)
	if err != nil {
		return err
	}

	if err := addDeployments(req, appInstance, tag, pullSecrets, interpolator, resp); err != nil {
		return err
	}
	if err := addRouters(appInstance, resp); err != nil {
		return err
	}
	if err := addJobs(req, appInstance, tag, pullSecrets, interpolator, resp); err != nil {
		return err
	}
	if err := addServices(req, appInstance, resp); err != nil {
		return err
	}
	if err := addPVCs(req, appInstance, resp); err != nil {
		return err
	}
	addAcorns(req, appInstance, tag, pullSecrets, resp)

	resp.Objects(pullSecrets.Objects()...)
	resp.Objects(interpolator.Objects()...)
	return pullSecrets.Err()
}

func addDeployments(req router.Request, appInstance *v1.AppInstance, tag name.Reference, pullSecrets *PullSecrets, secrets *secrets.Interpolator, resp router.Response) error {
	deps, err := ToDeployments(req, appInstance, tag, pullSecrets, secrets)
	if err != nil {
		return err
	}
	resp.Objects(deps...)
	return nil
}

func toEnvFrom(envs []v1.EnvVar) (result []corev1.EnvFromSource) {
	for _, env := range envs {
		if env.Secret.Name != "" && env.Secret.Key == "" {
			result = append(result, corev1.EnvFromSource{
				Prefix: env.Value,
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: env.Secret.Name,
					},
				},
			})
		}
	}
	return
}

func toEnv(envs []v1.EnvVar, appEnv []v1.NameValue, interpolator *secrets.Interpolator) (result []corev1.EnvVar) {
	for _, env := range envs {
		if env.Secret.Name == "" {
			result = append(result, interpolator.ToEnv(env.Name, env.Value))
		} else {
			if env.Secret.Key == "" {
				continue
			}
			result = append(result, corev1.EnvVar{
				Name: env.Name,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: env.Secret.Name,
						},
						Key: env.Secret.Key,
					},
				},
			})
		}
	}
	for _, appEnv := range appEnv {
		result = append(result, corev1.EnvVar{
			Name:  appEnv.Name,
			Value: appEnv.Value,
		})
	}
	return
}

func hasContextDir(container v1.Container) bool {
	for _, dir := range container.Dirs {
		if dir.ContextDir != "" {
			return true
		}
		for _, sidecar := range container.Sidecars {
			if hasContextDir(sidecar) {
				return true
			}
		}
	}
	return false
}

func toContainers(app *v1.AppInstance, tag name.Reference, name string, container v1.Container, interpolator *secrets.Interpolator) ([]corev1.Container, []corev1.Container) {
	var (
		containers     []corev1.Container
		initContainers []corev1.Container
	)

	if app.Status.GetDevMode() && hasContextDir(container) {
		initContainers = append(initContainers, corev1.Container{
			Name:            "acorn-helper",
			Image:           system.DefaultImage(),
			Command:         []string{"acorn-helper-init"},
			ImagePullPolicy: corev1.PullIfNotPresent,
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      sanitizeVolumeName(AcornHelper),
					MountPath: AcornHelperPath,
				},
			},
		})
	}

	newContainer := toContainer(app, tag, name, container, interpolator)
	containers = append(containers, newContainer)
	for _, entry := range typed.Sorted(container.Sidecars) {
		newContainer = toContainer(app, tag, entry.Key, entry.Value, interpolator)

		if entry.Value.Init {
			initContainers = append(initContainers, newContainer)
		} else {
			containers = append(containers, newContainer)
		}
	}

	return containers, initContainers
}

func pathHash(parts ...string) string {
	path := path.Join(parts...)
	hash := sha256.Sum256([]byte(path))
	return hex.EncodeToString(hash[:])[:12]
}

func sanitizeVolumeName(name string) string {
	if strings.Contains(name, "/") {
		return pathHash(name)
	}
	return name
}

func toMounts(app *v1.AppInstance, container v1.Container, interpolation *secrets.Interpolator) (result []corev1.VolumeMount) {
	for _, entry := range typed.Sorted(container.Files) {
		suffix := ""
		if volume.NormalizeMode(entry.Value.Mode) != "" {
			suffix = "-" + entry.Value.Mode
		}
		if entry.Value.Secret.Key == "" || entry.Value.Secret.Name == "" {
			// inline file
			result = append(result, interpolation.ToVolumeMount(entry.Key, entry.Value))
		} else {
			// file pointing to secret
			result = append(result, corev1.VolumeMount{
				Name:      secretPodVolName(entry.Value.Secret.Name + suffix),
				MountPath: path.Join("/", entry.Key),
				SubPath:   entry.Value.Secret.Key,
			})
		}
	}
	helperMounted := false
	for _, entry := range typed.Sorted(container.Dirs) {
		mountPath := entry.Key
		mount := entry.Value
		if mount.ContextDir != "" {
			if !helperMounted && app.Status.GetDevMode() {
				result = append(result, corev1.VolumeMount{
					Name:      sanitizeVolumeName(AcornHelper),
					MountPath: AcornHelperPath,
				})
				helperMounted = true
			}
		} else if mount.Secret.Name == "" {
			result = append(result, corev1.VolumeMount{
				Name:      sanitizeVolumeName(mount.Volume),
				MountPath: path.Join("/", mountPath),
				SubPath:   mount.SubPath,
			})
		} else {
			result = append(result, corev1.VolumeMount{
				Name:      secretPodVolName(mount.Secret.Name),
				MountPath: path.Join("/", mountPath),
			})
		}
	}
	return
}

func toPorts(container v1.Container) []corev1.ContainerPort {
	var (
		ports []corev1.ContainerPort
		seen  = map[struct {
			port  int32
			proto corev1.Protocol
		}]bool{}
	)
	for _, port := range container.Ports {
		protocol := corev1.ProtocolTCP
		if port.Protocol == v1.ProtocolUDP {
			protocol = corev1.ProtocolUDP
		}
		key := struct {
			port  int32
			proto corev1.Protocol
		}{port.TargetPort, protocol}
		if seen[key] {
			continue
		}
		seen[key] = true
		ports = append(ports, corev1.ContainerPort{
			ContainerPort: port.TargetPort,
			Protocol:      protocol,
		})
	}
	return ports
}

func parseURLForProbe(probeURL string) (scheme corev1.URIScheme, host string, port intstr.IntOrString, path string, ok bool) {
	u, err := url.Parse(probeURL)
	if err != nil {
		return
	}
	if u.Scheme == "https" {
		scheme = corev1.URISchemeHTTPS
	}
	host = u.Hostname()
	if host == "localhost" || host == "127.0.0.1" {
		host = ""
	}
	port = intstr.Parse(u.Port())
	if port.IntValue() == 0 {
		if scheme == corev1.URISchemeHTTPS {
			port = intstr.FromInt(443)
		}
		port = intstr.FromInt(80)
	}
	path = u.Path
	ok = true
	return
}

func toProbeHandler(probe v1.Probe) corev1.ProbeHandler {
	var (
		ok bool
		ph corev1.ProbeHandler
	)

	if probe.TCP != nil {
		socket := &corev1.TCPSocketAction{}
		_, socket.Host, socket.Port, _, ok = parseURLForProbe(probe.TCP.URL)
		if ok {
			ph.TCPSocket = socket
		}
	}
	if probe.Exec != nil {
		ph.Exec = &corev1.ExecAction{
			Command: probe.Exec.Command,
		}
	}
	if probe.HTTP != nil {
		http := &corev1.HTTPGetAction{}
		for _, entry := range typed.Sorted(probe.HTTP.Headers) {
			http.HTTPHeaders = append(http.HTTPHeaders, corev1.HTTPHeader{
				Name:  entry.Key,
				Value: entry.Value,
			})
		}
		http.Scheme, http.Host, http.Port, http.Path, ok = parseURLForProbe(probe.HTTP.URL)
		if ok {
			ph.HTTPGet = http
		}
	}
	return ph
}

func toProbe(container v1.Container, probeType v1.ProbeType) *corev1.Probe {
	for _, probe := range container.Probes {
		if probe.Type == probeType {
			return &corev1.Probe{
				ProbeHandler:        toProbeHandler(probe),
				InitialDelaySeconds: probe.InitialDelaySeconds,
				TimeoutSeconds:      probe.TimeoutSeconds,
				PeriodSeconds:       probe.PeriodSeconds,
				SuccessThreshold:    probe.SuccessThreshold,
				FailureThreshold:    probe.FailureThreshold,
			}
		}
	}

	if probeType == v1.ReadinessProbeType &&
		container.Probes == nil {
		for _, port := range container.Ports {
			if port.Protocol == v1.ProtocolUDP || port.Dev {
				continue
			}
			return &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					TCPSocket: &corev1.TCPSocketAction{
						Port: intstr.FromInt(int(port.TargetPort)),
					},
				},
			}
		}
	}

	return nil
}

func toContainer(app *v1.AppInstance, tag name.Reference, containerName string, container v1.Container, interpolator *secrets.Interpolator) corev1.Container {
	containerObject := corev1.Container{
		Name:           containerName,
		Image:          images.ResolveTag(tag, container.Image),
		Command:        container.Entrypoint,
		Args:           container.Command,
		WorkingDir:     container.WorkingDir,
		Env:            toEnv(container.Environment, app.Spec.Environment, interpolator),
		EnvFrom:        toEnvFrom(container.Environment),
		TTY:            container.Interactive,
		Stdin:          container.Interactive,
		Ports:          toPorts(container),
		VolumeMounts:   toMounts(app, container, interpolator),
		LivenessProbe:  toProbe(container, v1.LivenessProbeType),
		StartupProbe:   toProbe(container, v1.StartupProbeType),
		ReadinessProbe: toProbe(container, v1.ReadinessProbeType),
		Resources:      app.Status.Scheduling[containerName].Requirements,
	}

	return containerObject
}

func containerAnnotations(appInstance *v1.AppInstance, container v1.Container, name string) map[string]string {
	return labels.GatherScoped(name, v1.LabelTypeContainer, appInstance.Status.AppSpec.Annotations, container.Annotations, appInstance.Spec.Annotations)
}

func routerAnnotations(appInstance *v1.AppInstance, router v1.Router, name string) map[string]string {
	return labels.GatherScoped(name, v1.LabelTypeRouter, appInstance.Status.AppSpec.Annotations, router.Annotations, appInstance.Spec.Annotations)
}

func jobLabels(appInstance *v1.AppInstance, container v1.Container, name string, kv ...string) map[string]string {
	labelMap := labels.GatherScoped(name, v1.LabelTypeJob, appInstance.Status.AppSpec.Labels, container.Labels, appInstance.Spec.Labels)
	return mergeConLabels(labelMap, appInstance, name, kv...)
}

func containerLabels(appInstance *v1.AppInstance, container v1.Container, name string, kv ...string) map[string]string {
	labelMap := labels.GatherScoped(name, v1.LabelTypeContainer, appInstance.Status.AppSpec.Labels, container.Labels, appInstance.Spec.Labels)
	return mergeConLabels(labelMap, appInstance, name, kv...)
}

func routerLabels(appInstance *v1.AppInstance, router v1.Router, name string, kv ...string) map[string]string {
	labelMap := labels.GatherScoped(name, v1.LabelTypeRouter, appInstance.Status.AppSpec.Labels, router.Labels, appInstance.Spec.Labels)
	return mergeRouterLabels(labelMap, appInstance, name, kv...)
}

func routerSelectorMatchLabels(appInstance *v1.AppInstance, name string, kv ...string) map[string]string {
	return mergeRouterLabels(make(map[string]string), appInstance, name, kv...)
}

func selectorMatchLabels(appInstance *v1.AppInstance, name string, kv ...string) map[string]string {
	return mergeConLabels(make(map[string]string), appInstance, name, kv...)
}

func mergeRouterLabels(labelMap map[string]string, appInstance *v1.AppInstance, name string, kv ...string) map[string]string {
	kv = append([]string{labels.AcornRouterName, name}, kv...)
	return labels.Merge(labelMap, labels.Managed(appInstance, kv...))
}

func mergeConLabels(labelMap map[string]string, appInstance *v1.AppInstance, name string, kv ...string) map[string]string {
	kv = append([]string{labels.AcornContainerName, name}, kv...)
	return labels.Merge(labelMap, labels.Managed(appInstance, kv...))
}

func containerAnnotation(container v1.Container) string {
	// convert to map first to sort keys
	data, _ := convert.EncodeToMap(container)
	json, _ := json.Marshal(data)
	return string(json)
}

func podAnnotations(appInstance *v1.AppInstance, container v1.Container) map[string]string {
	annotations := map[string]string{
		labels.AcornContainerSpec: containerAnnotation(container),
	}
	images := map[string]string{}
	addImageAnnotations(images, appInstance, container)

	if len(images) == 0 {
		return annotations
	}

	data, err := json.Marshal(images)
	if err != nil {
		// this should never happen
		panic(err)
	}

	annotations[labels.AcornImageMapping] = string(data)
	return annotations
}

func addImageAnnotations(annotations map[string]string, appInstance *v1.AppInstance, container v1.Container) {
	if container.Build != nil && container.Build.BaseImage != "" {
		annotations[container.Image] = container.Build.BaseImage
	}

	for _, entry := range typed.Sorted(container.Sidecars) {
		addImageAnnotations(annotations, appInstance, entry.Value)
	}
}

func isStateful(appInstance *v1.AppInstance, container v1.Container) bool {
	for _, dir := range container.Dirs {
		if dir.Secret.Name != "" {
			continue
		}
		for volName, vol := range appInstance.Status.AppSpec.Volumes {
			if vol.Class == "ephemeral" {
				continue
			}
			if dir.Volume == volName {
				if len(vol.AccessModes) == 0 || (len(vol.AccessModes) == 1 && vol.AccessModes[0] == v1.AccessModeReadWriteOnce) {
					return true
				}
			}
		}
	}
	return false
}

func getRevision(req router.Request, namespace, secretName string) (string, error) {
	secret := &corev1.Secret{}
	if err := req.Get(secret, namespace, secretName); err != nil {
		return "0", err
	}
	hash := sha256.New()
	for _, entry := range typed.Sorted(secret.Data) {
		hash.Write([]byte(entry.Key))
		hash.Write([]byte{'\x00'})
		hash.Write(entry.Value)
		hash.Write([]byte{'\x00'})
	}
	d := hash.Sum(nil)
	return hex.EncodeToString(d[:]), nil
}

func getSecretAnnotations(req router.Request, appInstance *v1.AppInstance, container v1.Container, interpolator *secrets.Interpolator) (map[string]string, error) {
	var (
		secrets []string
		result  = map[string]string{}
	)

	for _, env := range container.Environment {
		if env.Secret.OnChange == v1.ChangeTypeRedeploy {
			secrets = append(secrets, env.Secret.Name)
		}
	}
	for _, file := range container.Files {
		if file.Secret.OnChange == v1.ChangeTypeRedeploy {
			secrets = append(secrets, file.Secret.Name)
		}
	}
	for _, dir := range container.Dirs {
		if dir.Secret.OnChange == v1.ChangeTypeRedeploy {
			secrets = append(secrets, dir.Secret.Name)
		}
	}

	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		rev, err := getRevision(req, appInstance.Status.Namespace, secret)
		if apierror.IsNotFound(err) {
			result[apply.AnnotationUpdate] = "false"
			result[apply.AnnotationCreate] = "false"
		} else if err != nil {
			return nil, err
		}
		result[labels.AcornSecretRevPrefix+secret] = rev
	}

	return result, nil
}

func toDeployment(req router.Request, appInstance *v1.AppInstance, tag name.Reference, name string, container v1.Container, pullSecrets *PullSecrets, interpolator *secrets.Interpolator) (*appsv1.Deployment, error) {
	var (
		stateful = isStateful(appInstance, container)
	)

	interpolator = interpolator.ForContainer(name)

	containers, initContainers := toContainers(appInstance, tag, name, container, interpolator)

	secretAnnotations, err := getSecretAnnotations(req, appInstance, container, interpolator)
	if err != nil {
		return nil, err
	}

	volumes, err := toVolumes(appInstance, container, interpolator)
	if err != nil {
		return nil, err
	}

	podLabels := containerLabels(appInstance, container, name, labels.AcornAppPublicName, publicname.Get(appInstance))
	deploymentLabels := containerLabels(appInstance, container, name)
	matchLabels := selectorMatchLabels(appInstance, name)

	deploymentAnnotations := containerAnnotations(appInstance, container, name)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   appInstance.Status.Namespace,
			Labels:      deploymentLabels,
			Annotations: typed.Concat(deploymentAnnotations, getDependencyAnnotations(appInstance, name, container.Dependencies), secretAnnotations),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: container.Scale,
			Selector: &metav1.LabelSelector{
				MatchLabels: matchLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podLabels,
					Annotations: typed.Concat(deploymentAnnotations, podAnnotations(appInstance, container), secretAnnotations),
				},
				Spec: corev1.PodSpec{
					Affinity:                      appInstance.Status.Scheduling[name].Affinity,
					Tolerations:                   appInstance.Status.Scheduling[name].Tolerations,
					TerminationGracePeriodSeconds: &[]int64{5}[0],
					ImagePullSecrets:              pullSecrets.ForContainer(name, append(containers, initContainers...)),
					EnableServiceLinks:            new(bool),
					Containers:                    containers,
					InitContainers:                initContainers,
					Volumes:                       volumes,
					ServiceAccountName:            name,
				},
			},
		},
	}

	if stateful {
		dep.Spec.Replicas = &[]int32{1}[0]
		dep.Spec.Template.Spec.Hostname = dep.Name
		dep.Spec.Strategy.Type = appsv1.RecreateDeploymentStrategyType
	} else if dep.Spec.Replicas == nil || *dep.Spec.Replicas == 1 {
		dep.Spec.Template.Spec.Hostname = dep.Name
	}

	if appInstance.Spec.Stop != nil && *appInstance.Spec.Stop {
		dep.Spec.Replicas = new(int32)
	} else {
		interpolator.AddMissingAnnotations(dep.Annotations)
	}

	return dep, nil
}

func ToDeployments(req router.Request, appInstance *v1.AppInstance, tag name.Reference, pullSecrets *PullSecrets, secrets *secrets.Interpolator) (result []kclient.Object, _ error) {
	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Containers) {
		if ports.IsLinked(appInstance, entry.Key) {
			continue
		}
		dep, err := toDeployment(req, appInstance, tag, entry.Key, entry.Value, pullSecrets, secrets)
		if err != nil {
			return nil, err
		}
		sa, err := toServiceAccount(req, dep.GetName(), dep.GetLabels(), dep.GetAnnotations(), appInstance)
		if err != nil {
			return nil, err
		}
		if perms := v1.FindPermission(dep.GetName(), appInstance.Spec.Permissions); perms.HasRules() {
			result = append(result, toPermissions(perms, dep.GetLabels(), dep.GetAnnotations(), appInstance)...)
		}
		result = append(result, sa, dep, pdb.ToPodDisruptionBudget(dep))
	}
	return result, nil
}
