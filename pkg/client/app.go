package client

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/run"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func ToApp(namespace, image string, opts *AppRunOptions) *apiv1.App {
	if opts == nil {
		opts = &AppRunOptions{}
	}

	apiVersion, kind := apiv1.SchemeGroupVersion.WithKind("App").ToAPIVersionAndKind()
	name := opts.Name
	if name == "" {
		name = run.NameGenerator.Generate()
	}

	return &apiv1.App{
		TypeMeta: metav1.TypeMeta{
			Kind:       kind,
			APIVersion: apiVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: appScoped(opts.Annotations),
			Labels:      appScoped(opts.Labels),
		},
		Spec: v1.AppInstanceSpec{
			Image:           image,
			PublishMode:     opts.PublishMode,
			DeployArgs:      opts.DeployArgs,
			Volumes:         opts.Volumes,
			Secrets:         opts.Secrets,
			Links:           opts.Links,
			Ports:           opts.Ports,
			Profiles:        opts.Profiles,
			DevMode:         opts.DevMode,
			Permissions:     opts.Permissions,
			Environment:     opts.Env,
			Labels:          opts.Labels,
			Annotations:     opts.Annotations,
			TargetNamespace: opts.TargetNamespace,
		},
	}
}

func (c *client) AppRun(ctx context.Context, image string, opts *AppRunOptions) (*apiv1.App, error) {
	app := ToApp(c.Namespace, image, opts)
	return app, translatePermissions(c.Client.Create(ctx, app))
}

func (c *client) AppUpdate(ctx context.Context, name string, opts *AppUpdateOptions) (result *apiv1.App, err error) {
	for i := 0; i < 5; i++ {
		result, err = c.appUpdate(ctx, name, opts)
		if apierrors.IsConflict(err) {
			continue
		}
		return
	}
	return
}

func ToAppUpdate(ctx context.Context, c Client, name string, opts *AppUpdateOptions) (*apiv1.App, error) {

	app, err := c.AppGet(ctx, name)
	if err != nil {
		return nil, err
	}

	if opts == nil {
		return app, nil
	}

	if opts.Image != "" {
		app.Spec.Image = opts.Image
	}

	// Reset Mode (Not patch mode)
	if opts.Reset {

		o := opts.ToRun()

		nApp := ToApp(app.Namespace, opts.Image, &o)

		nApp.Name = app.Name
		nApp.ObjectMeta.UID = app.ObjectMeta.UID
		nApp.ObjectMeta.ResourceVersion = app.ObjectMeta.ResourceVersion

		return nApp, nil

	}

	app.Labels = typed.Concat(app.Labels, appScoped(opts.Labels))
	app.Annotations = typed.Concat(app.Annotations, appScoped(opts.Annotations))
	app.Spec.Volumes = mergeVolumes(app.Spec.Volumes, opts.Volumes)
	app.Spec.Secrets = mergeSecrets(app.Spec.Secrets, opts.Secrets)
	app.Spec.Links = mergeServices(app.Spec.Links, opts.Links)
	app.Spec.Ports = mergePorts(app.Spec.Ports, opts.Ports)
	app.Spec.Environment = mergeEnv(app.Spec.Environment, opts.Env)
	app.Spec.Labels = mergeLabels(app.Spec.Labels, opts.Labels)
	app.Spec.Annotations = mergeLabels(app.Spec.Annotations, opts.Annotations)
	app.Spec.DeployArgs = typed.Concat(app.Spec.DeployArgs, opts.DeployArgs)
	if len(opts.Profiles) > 0 {
		app.Spec.Profiles = opts.Profiles
	}
	if opts.DevMode != nil {
		app.Spec.DevMode = opts.DevMode
	}
	if opts.PublishMode != "" {
		app.Spec.PublishMode = opts.PublishMode
	}
	if opts.Permissions != nil {
		app.Spec.Permissions = opts.Permissions
	}
	if opts.TargetNamespace != "" {
		app.Spec.TargetNamespace = opts.TargetNamespace
	}

	return app, nil
}

func (c *client) appUpdate(ctx context.Context, name string, opts *AppUpdateOptions) (*apiv1.App, error) {
	app, err := ToAppUpdate(ctx, c, name, opts)
	if err != nil {
		return nil, err
	}
	return app, translatePermissions(c.Client.Update(ctx, app))
}

func translatePermissions(err error) error {
	if err == nil {
		return err
	}
	if i := strings.Index(err.Error(), PrefixErrRulesNeeded); i != -1 {
		perms := v1.Permissions{}
		marshalErr := json.Unmarshal([]byte(err.Error()[i+len(PrefixErrRulesNeeded):]), &perms)
		if marshalErr == nil {
			return &ErrRulesNeeded{
				Permissions: perms,
			}
		}
	}
	return err
}

func (c *client) AppLog(ctx context.Context, name string, opts *LogOptions) (<-chan apiv1.LogMessage, error) {
	appName, _, _ := strings.Cut(name, ".")

	app, err := c.AppGet(ctx, appName)
	if err != nil {
		return nil, err
	}

	if opts == nil {
		opts = &LogOptions{}
	}

	if name != appName && opts.ContainerReplica == "" {
		opts.ContainerReplica = name
	}

	url := c.RESTClient.Get().
		Namespace(app.Namespace).
		Resource("apps").
		Name(app.Name).
		SubResource("log").
		VersionedParams((*apiv1.LogOptions)(opts), scheme.ParameterCodec).
		URL()

	conn, err := c.Dialer.DialWebsocket(ctx, url.String(), nil)
	if err != nil {
		return nil, err
	}

	result := make(chan apiv1.LogMessage)
	go func() {
		defer close(result)
		defer conn.Close()
		for {
			_, data, err := conn.ReadMessage()
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				break
			} else if err != nil {
				logrus.Errorf("error reading websocket: %v", err)
				break
			}
			message := apiv1.LogMessage{}
			if err := json.Unmarshal(data, &message); err == nil {
				result <- message
			} else {
				result <- apiv1.LogMessage{
					Error: err.Error(),
				}
			}
		}
	}()

	return result, nil
}

func mergeEnv(appEnv, optsEnv []v1.NameValue) []v1.NameValue {
	for _, newEnv := range optsEnv {
		found := false
		for i, existingEnv := range appEnv {
			if existingEnv.Name == newEnv.Name {
				appEnv[i] = newEnv
				found = true
				break
			}
		}
		if !found {
			appEnv = append(appEnv, newEnv)
		}
	}

	return appEnv
}

func mergePorts(appPorts, optsPorts []v1.PortBinding) []v1.PortBinding {
	for _, newPort := range optsPorts {
		found := false
		for i, existingPort := range appPorts {
			if existingPort.TargetPort == newPort.TargetPort {
				appPorts[i] = newPort
				found = true
				break
			}
		}
		if !found {
			appPorts = append(appPorts, newPort)
		}
	}

	return appPorts
}

func mergeServices(appServices, optsServices []v1.ServiceBinding) []v1.ServiceBinding {
	for _, newService := range optsServices {
		found := false
		for i, existingService := range appServices {
			if existingService.Target == newService.Target {
				appServices[i] = newService
				found = true
				break
			}
		}
		if !found {
			appServices = append(appServices, newService)
		}
	}

	return appServices
}

func mergeSecrets(appSecrets, optsSecrets []v1.SecretBinding) []v1.SecretBinding {
	for _, newSecret := range optsSecrets {
		found := false
		for i, existingSecret := range appSecrets {
			if existingSecret.Target == newSecret.Target {
				appSecrets[i] = newSecret
				found = true
				break
			}
		}
		if !found {
			appSecrets = append(appSecrets, newSecret)
		}
	}

	return appSecrets
}

func mergeVolumes(appVolumes, optsVolumes []v1.VolumeBinding) []v1.VolumeBinding {
	for _, newVolume := range optsVolumes {
		found := false
		for i, existingVolume := range appVolumes {
			if existingVolume.Target == newVolume.Target {
				appVolumes[i] = newVolume
				found = true
				break
			}
		}
		if !found {
			appVolumes = append(appVolumes, newVolume)
		}
	}

	return appVolumes
}

func mergeLabels(appLabels, optsLabels []v1.ScopedLabel) []v1.ScopedLabel {
	for _, newLabel := range optsLabels {
		found := false
		for i, existingLabel := range appLabels {
			if existingLabel.ResourceType == newLabel.ResourceType && existingLabel.ResourceName == newLabel.ResourceName &&
				existingLabel.Key == newLabel.Key {
				appLabels[i] = newLabel
				found = true
				break
			}
		}
		if !found {
			appLabels = append(appLabels, newLabel)
		}
	}

	return appLabels
}

func appScoped(scoped []v1.ScopedLabel) map[string]string {
	labels := make(map[string]string)
	for _, s := range scoped {
		if s.ResourceType == v1.LabelTypeMeta || (s.ResourceType == "" && s.ResourceName == "") {
			labels[s.Key] = s.Value
		}
	}
	return labels
}

func (c *client) AppDelete(ctx context.Context, name string) (*apiv1.App, error) {
	app, err := c.AppGet(ctx, name)
	if apierrors.IsNotFound(err) {
		return nil, nil
	}

	return app, c.Client.Delete(ctx, &apiv1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Namespace,
		},
	})
}

func (c *client) AppGet(ctx context.Context, name string) (*apiv1.App, error) {
	app := &apiv1.App{}
	err := c.Client.Get(ctx, kclient.ObjectKey{
		Name:      name,
		Namespace: c.Namespace,
	}, app)
	if err != nil {
		return nil, err
	}

	return app, nil
}

func (c *client) AppList(ctx context.Context) ([]apiv1.App, error) {
	apps := &apiv1.AppList{}
	err := c.Client.List(ctx, apps, &kclient.ListOptions{
		Namespace: c.Namespace,
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(apps.Items, func(i, j int) bool {
		if apps.Items[i].CreationTimestamp.Time == apps.Items[j].CreationTimestamp.Time {
			return apps.Items[i].Name < apps.Items[j].Name
		}
		return apps.Items[i].CreationTimestamp.After(apps.Items[j].CreationTimestamp.Time)
	})

	return apps.Items, nil
}

func (c *client) AppStart(ctx context.Context, name string) (err error) {
	for i := 0; i < 5; i++ {
		err = c.appStart(ctx, name)
		if apierrors.IsConflict(err) {
			continue
		}
		return
	}
	return
}

func (c *client) appStart(ctx context.Context, name string) error {
	app := &apiv1.App{}
	err := c.Client.Get(ctx, kclient.ObjectKey{
		Name:      name,
		Namespace: c.Namespace,
	}, app)
	if err != nil {
		return err
	}
	if app.Spec.Stop != nil && *app.Spec.Stop {
		app.Spec.Stop = new(bool)
		return c.Client.Update(ctx, app)
	}
	return nil
}

func (c *client) AppStop(ctx context.Context, name string) (err error) {
	for i := 0; i < 5; i++ {
		err = c.appStop(ctx, name)
		if apierrors.IsConflict(err) {
			continue
		}
		return
	}
	return
}

func (c *client) appStop(ctx context.Context, name string) error {
	app := &apiv1.App{}
	err := c.Client.Get(ctx, kclient.ObjectKey{
		Name:      name,
		Namespace: c.Namespace,
	}, app)
	if apierrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}
	if app.Spec.Stop == nil || !*app.Spec.Stop {
		app.Spec.Stop = &[]bool{true}[0]
		return c.Client.Update(ctx, app)
	}
	return nil
}
