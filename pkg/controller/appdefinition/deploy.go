package appdefinition

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"path"

	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/condition"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/pull"
	"github.com/acorn-io/baaah/pkg/meta"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/rancher/wrangler/pkg/data/convert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func DeploySpec(req router.Request, resp router.Response) (err error) {
	appInstance := req.Object.(*v1.AppInstance)
	status := condition.Setter(appInstance, resp, v1.AppInstanceConditionDefined)
	defer func() {
		if err == nil {
			status.Success()
		} else {
			status.Error(err)
		}
	}()

	tag, err := pull.GetTag(req.Ctx, router.ToReader(req.Client), appInstance.Namespace, appInstance.Spec.Image)
	if err != nil {
		return err
	}

	pullSecrets, err := pullSecrets(req, appInstance, resp)
	if err != nil {
		return err
	}

	addNamespace(appInstance, resp)
	addDeployments(appInstance, tag, pullSecrets, resp)
	addJobs(appInstance, tag, pullSecrets, resp)
	addServices(appInstance, resp)
	if err := addIngress(appInstance, req, resp); err != nil {
		return err
	}
	addPVCs(appInstance, resp)
	if err := addConfigMaps(appInstance, resp); err != nil {
		return err
	}
	addAcorns(appInstance, tag, pullSecrets, resp)
	return nil
}

func addDeployments(appInstance *v1.AppInstance, tag name.Reference, pullSecrets []corev1.LocalObjectReference, resp router.Response) {
	resp.Objects(toDeployments(appInstance, tag, pullSecrets)...)
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
					Optional: env.Secret.Optional,
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
						Key:      env.Secret.Key,
						Optional: env.Secret.Optional,
					},
				},
			})
		}
	}
	return
}

func toContainers(app *v1.AppInstance, tag name.Reference, name string, container v1.Container) ([]corev1.Container, []corev1.Container) {
	var (
		containers     []corev1.Container
		initContainers []corev1.Container
	)

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

func toMounts(appName, deploymentName, containerName string, container v1.Container) (result []corev1.VolumeMount) {
	for _, entry := range typed.Sorted(container.Files) {
		if entry.Value.Secret.Key == "" || entry.Value.Secret.Name == "" {
			result = append(result, corev1.VolumeMount{
				Name:      "files",
				MountPath: path.Join("/", entry.Key),
				SubPath:   pathHash(appName, deploymentName, containerName, entry.Key),
			})
		} else {
			result = append(result, corev1.VolumeMount{
				Name:      "secret--" + entry.Value.Secret.Name,
				MountPath: path.Join("/", entry.Key),
				SubPath:   entry.Value.Secret.Key,
			})
		}
	}
	for _, entry := range typed.Sorted(container.Dirs) {
		mountPath := entry.Key
		mount := entry.Value
		if mount.ContextDir != "" {
			continue
		}
		if mount.Secret.Name == "" {
			result = append(result, corev1.VolumeMount{
				Name:      mount.Volume,
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
			ContainerPort: port.ContainerPort,
			Protocol:      protocol,
		})
	}
	return ports
}

func resolveTag(app *v1.AppInstance, tag name.Reference, containerName string, container v1.Container) string {
	override := app.Spec.Images[containerName]
	if override != "" {
		return override
	}
	return tag.Context().Digest(container.Image).String()
}

func toContainer(app *v1.AppInstance, tag name.Reference, deploymentName, containerName string, container v1.Container) corev1.Container {
	return corev1.Container{
		Name:         containerName,
		Image:        resolveTag(app, tag, containerName, container),
		Command:      container.Entrypoint,
		Args:         container.Command,
		WorkingDir:   container.WorkingDir,
		Env:          toEnv(container.Environment),
		EnvFrom:      toEnvFrom(container.Environment),
		TTY:          container.Interactive,
		Stdin:        container.Interactive,
		Ports:        toPorts(container),
		VolumeMounts: toMounts(app.Name, deploymentName, containerName, container),
	}
}

func containerLabels(appInstance *v1.AppInstance, name string, kv ...string) map[string]string {
	labels := map[string]string{
		labels.AcornAppName:       appInstance.Name,
		labels.AcornAppNamespace:  appInstance.Namespace,
		labels.AcornContainerName: name,
		labels.AcornManaged:       "true",
	}
	for i := 0; i+1 < len(kv); i += 2 {
		if kv[i+1] == "" {
			delete(labels, kv[i])
		} else {
			labels[kv[i]] = kv[i+1]
		}
	}
	return labels
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
		override := appInstance.Spec.Images[containerName]
		if override == "" {
			annotations[container.Image] = container.Build.BaseImage
		}
	}

	for _, entry := range typed.Sorted(container.Sidecars) {
		name, sideCar := entry.Key, entry.Value
		addImageAnnotations(annotations, appInstance, name, sideCar)
	}
}

func toDeployment(appInstance *v1.AppInstance, tag name.Reference, name string, container v1.Container, pullSecrets []corev1.LocalObjectReference) *appsv1.Deployment {
	var replicas *int32
	if appInstance.Spec.Stop != nil && *appInstance.Spec.Stop {
		replicas = new(int32)
	}
	containers, initContainers := toContainers(appInstance, tag, name, container)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: appInstance.Status.Namespace,
			Labels: containerLabels(appInstance, name,
				labels.AcornManaged, "true",
			),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: containerLabels(appInstance, name),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: containerLabels(appInstance, name,
						labels.AcornManaged, "true",
					),
					Annotations: podAnnotations(appInstance, name, container),
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets:             pullSecrets,
					ShareProcessNamespace:        &[]bool{true}[0],
					EnableServiceLinks:           new(bool),
					Containers:                   containers,
					InitContainers:               initContainers,
					Volumes:                      toVolumes(appInstance, container),
					AutomountServiceAccountToken: new(bool),
				},
			},
		},
	}
}

func toDeployments(appInstance *v1.AppInstance, tag name.Reference, pullSecrets []corev1.LocalObjectReference) (result []meta.Object) {
	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Containers) {
		result = append(result, toDeployment(appInstance, tag, entry.Key, entry.Value, pullSecrets))
	}
	return result
}

func addNamespace(appInstance *v1.AppInstance, resp router.Response) {
	resp.Objects(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: appInstance.Status.Namespace,
			Labels: map[string]string{
				labels.AcornAppName:                  appInstance.Name,
				labels.AcornAppNamespace:             appInstance.Namespace,
				labels.AcornManaged:                  "true",
				"pod-security.kubernetes.io/enforce": "baseline",
			},
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

func toConfigMaps(appInstance *v1.AppInstance) (result []meta.Object, err error) {
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
	return []meta.Object{configMap}, nil
}
