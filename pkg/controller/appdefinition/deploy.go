package appdefinition

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	url2 "net/url"
	"path"
	"regexp"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/condition"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/install"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/pull"
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

var (
	DigestPattern = regexp.MustCompile(`^sha256:[a-f\d]{64}$`)
)

type ErrMissingSecret struct {
	Name      string
	Namespace string
}

func (e *ErrMissingSecret) Error() string {
	return fmt.Sprintf("missing secret: %s/%s", e.Namespace, e.Name)
}

func DeploySpec(req router.Request, resp router.Response) (err error) {
	defer func() {
		if missing := (*ErrMissingSecret)(nil); errors.As(err, &missing) {
			err = nil
		}
	}()

	appInstance := req.Object.(*v1.AppInstance)
	status := condition.Setter(appInstance, resp, v1.AppInstanceConditionDefined)
	defer func() {
		if err == nil {
			status.Success()
		} else {
			status.Error(err)
		}
	}()

	tag, err := pull.GetTag(req.Ctx, req.Client, appInstance.Namespace, appInstance.Spec.Image)
	if err != nil {
		return err
	}

	pullSecrets, err := NewPullSecrets(req, appInstance)
	if err != nil {
		return err
	}

	cfg, err := config.Get(req.Ctx, req.Client)
	if err != nil {
		return err
	}

	addNamespace(cfg, appInstance, resp)
	if err := addDeployments(req, appInstance, tag, pullSecrets, resp); err != nil {
		return err
	}
	if err := addJobs(req, appInstance, tag, pullSecrets, resp); err != nil {
		return err
	}
	addServices(appInstance, resp)
	addLinks(appInstance, resp)
	if err := addIngress(appInstance, req, resp); err != nil {
		return err
	}
	addPVCs(appInstance, resp)
	if err := addConfigMaps(appInstance, resp); err != nil {
		return err
	}
	addAcorns(appInstance, tag, pullSecrets, resp)

	resp.Objects(pullSecrets.Objects()...)
	return pullSecrets.Err()
}

func addDeployments(req router.Request, appInstance *v1.AppInstance, tag name.Reference, pullSecrets *PullSecrets, resp router.Response) error {
	deps, err := ToDeployments(req, appInstance, tag, pullSecrets)
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

func toEnv(envs []v1.EnvVar) (result []corev1.EnvVar) {
	for _, env := range envs {
		if env.Secret.Name == "" {
			result = append(result, corev1.EnvVar{
				Name:  env.Name,
				Value: env.Value,
			})
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

func toContainers(app *v1.AppInstance, tag name.Reference, name string, container v1.Container) ([]corev1.Container, []corev1.Container) {
	var (
		containers     []corev1.Container
		initContainers []corev1.Container
	)

	if app.Spec.GetDevMode() && hasContextDir(container) {
		initContainers = append(initContainers, corev1.Container{
			Name:            "acorn-helper",
			Image:           install.DefaultImage(),
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

	containers = append(containers, toContainer(app, tag, name, name, container))
	for _, entry := range typed.Sorted(container.Sidecars) {
		newContainer := toContainer(app, tag, name, entry.Key, entry.Value)
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

func toMounts(app *v1.AppInstance, deploymentName, containerName string, container v1.Container) (result []corev1.VolumeMount) {
	for _, entry := range typed.Sorted(container.Files) {
		suffix := ""
		if normalizeMode(entry.Value.Mode) != "" {
			suffix = "-" + entry.Value.Mode
		}
		if entry.Value.Secret.Key == "" || entry.Value.Secret.Name == "" {
			result = append(result, corev1.VolumeMount{
				Name:      "files" + suffix,
				MountPath: path.Join("/", entry.Key),
				SubPath:   pathHash(app.Name, deploymentName, containerName, entry.Key),
			})
		} else {
			result = append(result, corev1.VolumeMount{
				Name:      "secret--" + entry.Value.Secret.Name + suffix,
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
			if !helperMounted && app.Spec.GetDevMode() {
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
				Name:      "secret--" + mount.Secret.Name,
				MountPath: path.Join("/", mountPath),
			})
		}
	}
	return
}

func toPorts(container v1.Container) []corev1.ContainerPort {
	var ports []corev1.ContainerPort
	for _, port := range container.Ports {
		protocol := corev1.ProtocolTCP
		if port.Protocol == v1.ProtocolUDP {
			protocol = corev1.ProtocolUDP
		}
		ports = append(ports, corev1.ContainerPort{
			ContainerPort: port.InternalPort,
			Protocol:      protocol,
		})
	}
	return ports
}

func resolveTag(tag name.Reference, image string) string {
	if DigestPattern.MatchString(image) {
		return tag.Context().Digest(image).String()
	}
	return image
}

func parseURLForProbe(probeURL string) (scheme corev1.URIScheme, host string, port intstr.IntOrString, path string, ok bool) {
	u, err := url2.Parse(probeURL)
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
		len(container.Probes) == 0 &&
		len(container.Ports) > 0 {
		for _, port := range container.Ports {
			if port.Protocol == v1.ProtocolUDP {
				continue
			}
			return &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					TCPSocket: &corev1.TCPSocketAction{
						Port: intstr.FromInt(int(port.InternalPort)),
					},
				},
			}
		}
	}

	return nil
}

func toContainer(app *v1.AppInstance, tag name.Reference, deploymentName, containerName string, container v1.Container) corev1.Container {
	return corev1.Container{
		Name:           containerName,
		Image:          resolveTag(tag, container.Image),
		Command:        container.Entrypoint,
		Args:           container.Command,
		WorkingDir:     container.WorkingDir,
		Env:            toEnv(container.Environment),
		EnvFrom:        toEnvFrom(container.Environment),
		TTY:            container.Interactive,
		Stdin:          container.Interactive,
		Ports:          toPorts(container),
		VolumeMounts:   toMounts(app, deploymentName, containerName, container),
		LivenessProbe:  toProbe(container, v1.LivenessProbeType),
		StartupProbe:   toProbe(container, v1.StartupProbeType),
		ReadinessProbe: toProbe(container, v1.ReadinessProbeType),
	}
}

func containerLabels(appInstance *v1.AppInstance, name string, kv ...string) map[string]string {
	kv = append([]string{labels.AcornContainerName, name}, kv...)
	return labels.Managed(appInstance, kv...)
}

func containerAnnotation(container v1.Container) string {
	// convert to map first to sort keys
	data, _ := convert.EncodeToMap(container)
	json, _ := json.Marshal(data)
	return string(json)
}

func podAnnotations(appInstance *v1.AppInstance, containerName string, container v1.Container) map[string]string {
	annotations := map[string]string{
		labels.AcornContainerSpec: containerAnnotation(container),
	}
	images := map[string]string{}
	addImageAnnotations(images, appInstance, containerName, container)

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

func addImageAnnotations(annotations map[string]string, appInstance *v1.AppInstance, containerName string, container v1.Container) {
	if container.Build != nil && container.Build.BaseImage != "" {
		annotations[container.Image] = container.Build.BaseImage
	}

	for _, entry := range typed.Sorted(container.Sidecars) {
		name, sideCar := entry.Key, entry.Value
		addImageAnnotations(annotations, appInstance, name, sideCar)
	}
}

func isStateful(appInstance *v1.AppInstance, container v1.Container) bool {
	for _, dir := range container.Dirs {
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
	if err := req.Get(secret, namespace, secretName); apierror.IsNotFound(err) {
		return "0", &ErrMissingSecret{Namespace: namespace, Name: secretName}
	} else if err != nil {
		return "0", err
	}
	return secret.ResourceVersion, nil
}

func getSecretAnnotations(req router.Request, appInstance *v1.AppInstance, container v1.Container) (map[string]string, error) {
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
		if err != nil {
			return nil, err
		}
		result[labels.AcornSecretRevPrefix+secret] = rev
	}

	return result, nil
}

func toDeployment(req router.Request, appInstance *v1.AppInstance, tag name.Reference, name string, container v1.Container, pullSecrets *PullSecrets) (*appsv1.Deployment, error) {
	var (
		aliasLabels []string
		stateful    = isStateful(appInstance, container)
	)

	if container.Alias.Name != "" {
		aliasLabels = []string{labels.AcornAlias + container.Alias.Name, "true"}
	}
	containers, initContainers := toContainers(appInstance, tag, name, container)

	secretAnnotations, err := getSecretAnnotations(req, appInstance, container)
	if err != nil {
		return nil, err
	}

	volumes, err := toVolumes(appInstance, container)
	if err != nil {
		return nil, err
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: appInstance.Status.Namespace,
			Labels:    containerLabels(appInstance, name),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: container.Scale,
			Selector: &metav1.LabelSelector{
				MatchLabels: containerLabels(appInstance, name),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: containerLabels(appInstance, name,
						aliasLabels...,
					),
					Annotations: typed.Concat(podAnnotations(appInstance, name, container), secretAnnotations),
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: &[]int64{5}[0],
					ImagePullSecrets:              pullSecrets.ForContainer(name, append(containers, initContainers...)),
					ShareProcessNamespace:         &[]bool{true}[0],
					EnableServiceLinks:            new(bool),
					Containers:                    containers,
					InitContainers:                initContainers,
					Volumes:                       volumes,
					AutomountServiceAccountToken:  new(bool),
				},
			},
		},
	}
	if stateful {
		dep.Spec.Replicas = &[]int32{1}[0]
		dep.Spec.Template.Spec.Hostname = dep.Name
		dep.Spec.Strategy.Type = appsv1.RecreateDeploymentStrategyType
	}
	if appInstance.Spec.Stop != nil && *appInstance.Spec.Stop {
		dep.Spec.Replicas = new(int32)
	}
	return dep, nil
}

func ToDeployments(req router.Request, appInstance *v1.AppInstance, tag name.Reference, pullSecrets *PullSecrets) (result []kclient.Object, _ error) {
	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Containers) {
		if isLinked(appInstance, entry.Key) {
			continue
		}
		dep, err := toDeployment(req, appInstance, tag, entry.Key, entry.Value, pullSecrets)
		if err != nil {
			return nil, err
		}
		result = append(result, dep)
	}
	return result, nil
}

func addNamespace(cfg *apiv1.Config, appInstance *v1.AppInstance, resp router.Response) {
	labels := map[string]string{
		labels.AcornAppName:      appInstance.Name,
		labels.AcornAppNamespace: appInstance.Namespace,
		labels.AcornManaged:      "true",
	}

	if *cfg.SetPodSecurityEnforceProfile {
		labels["pod-security.kubernetes.io/enforce"] = cfg.PodSecurityEnforceProfile
	}

	resp.Objects(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   appInstance.Status.Namespace,
			Labels: labels,
		},
	})
}

func addServices(appInstance *v1.AppInstance, resp router.Response) {
	resp.Objects(toServices(appInstance)...)
}

func addFileContent(configMap *corev1.ConfigMap, appName, deploymentName string, container v1.Container) error {
	data := configMap.BinaryData
	for filePath, file := range container.Files {
		content, err := base64.StdEncoding.DecodeString(file.Content)
		if err != nil {
			return err
		}
		hashPath := pathHash(appName, deploymentName, deploymentName, filePath)
		data[hashPath] = content
		configMap.Annotations[hashPath] = path.Join(appName, deploymentName, deploymentName, filePath)
	}
	for sidecarName, sidecar := range container.Sidecars {
		for filePath, file := range sidecar.Files {
			content, err := base64.StdEncoding.DecodeString(file.Content)
			if err != nil {
				return err
			}
			hashPath := pathHash(appName, deploymentName, sidecarName, filePath)
			data[hashPath] = content
			configMap.Annotations[hashPath] = path.Join(appName, deploymentName, sidecarName, filePath)
		}
	}
	return nil
}

func addConfigMaps(appInstance *v1.AppInstance, resp router.Response) error {
	objs, err := toConfigMaps(appInstance)
	resp.Objects(objs...)
	return err
}

func toConfigMaps(appInstance *v1.AppInstance) (result []kclient.Object, err error) {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "files",
			Namespace: appInstance.Status.Namespace,
			Labels: map[string]string{
				labels.AcornManaged: "true",
			},
			Annotations: map[string]string{},
		},
		BinaryData: map[string][]byte{},
	}
	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Containers) {
		if err := addFileContent(configMap, appInstance.Name, entry.Key, entry.Value); err != nil {
			return nil, err
		}
	}
	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Jobs) {
		if err := addFileContent(configMap, appInstance.Name, entry.Key, entry.Value); err != nil {
			return nil, err
		}
	}
	if len(configMap.BinaryData) == 0 {
		return nil, nil
	}

	fileMode := getFilesFileModesForApp(appInstance)
	for _, mode := range typed.SortedKeys(fileMode) {
		if mode == "" {
			result = append(result, configMap)
		} else {
			copy := configMap.DeepCopy()
			copy.Name += "-" + mode
			result = append(result, copy)
		}
	}

	return result, nil
}
