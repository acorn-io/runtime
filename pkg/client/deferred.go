package client

import (
	"context"
	"sync"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client/term"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeferredClient struct {
	Project    string
	Namespace  string
	New        func() (Client, error)
	Client     Client
	createLock sync.Mutex
}

func (d *DeferredClient) create() error {
	d.createLock.Lock()
	defer d.createLock.Unlock()
	if d.Client == nil {
		c, err := d.New()
		if err != nil {
			return err
		}
		d.Client = c
	}
	return nil
}

func (d *DeferredClient) AppList(ctx context.Context) ([]apiv1.App, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.AppList(ctx)
}

func (d *DeferredClient) AppDelete(ctx context.Context, name string) (*apiv1.App, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.AppDelete(ctx, name)
}

func (d *DeferredClient) AppGet(ctx context.Context, name string) (*apiv1.App, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.AppGet(ctx, name)
}

func (d *DeferredClient) AppStop(ctx context.Context, name string) error {
	if err := d.create(); err != nil {
		return err
	}
	return d.Client.AppStop(ctx, name)
}

func (d *DeferredClient) AppStart(ctx context.Context, name string) error {
	if err := d.create(); err != nil {
		return err
	}
	return d.Client.AppStart(ctx, name)
}

func (d *DeferredClient) AppRun(ctx context.Context, image string, opts *AppRunOptions) (*apiv1.App, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.AppRun(ctx, image, opts)
}

func (d *DeferredClient) AppUpdate(ctx context.Context, name string, opts *AppUpdateOptions) (*apiv1.App, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.AppUpdate(ctx, name, opts)
}

func (d *DeferredClient) AppLog(ctx context.Context, name string, opts *LogOptions) (<-chan apiv1.LogMessage, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.AppLog(ctx, name, opts)
}

func (d *DeferredClient) AppConfirmUpgrade(ctx context.Context, name string) error {
	if err := d.create(); err != nil {
		return err
	}
	return d.Client.AppConfirmUpgrade(ctx, name)
}

func (d *DeferredClient) AppPullImage(ctx context.Context, name string) error {
	if err := d.create(); err != nil {
		return err
	}
	return d.Client.AppPullImage(ctx, name)
}

func (d *DeferredClient) CredentialCreate(ctx context.Context, serverAddress, username, password string, skipChecks bool) (*apiv1.Credential, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.CredentialCreate(ctx, serverAddress, username, password, skipChecks)
}

func (d *DeferredClient) CredentialList(ctx context.Context) ([]apiv1.Credential, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.CredentialList(ctx)
}

func (d *DeferredClient) CredentialGet(ctx context.Context, serverAddress string) (*apiv1.Credential, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.CredentialGet(ctx, serverAddress)
}

func (d *DeferredClient) CredentialUpdate(ctx context.Context, serverAddress, username, password string, skipChecks bool) (*apiv1.Credential, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.CredentialUpdate(ctx, serverAddress, username, password, skipChecks)
}

func (d *DeferredClient) CredentialDelete(ctx context.Context, serverAddress string) (*apiv1.Credential, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.CredentialDelete(ctx, serverAddress)
}

func (d *DeferredClient) SecretCreate(ctx context.Context, name, secretType string, data map[string][]byte) (*apiv1.Secret, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.SecretCreate(ctx, name, secretType, data)
}

func (d *DeferredClient) SecretList(ctx context.Context) ([]apiv1.Secret, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.SecretList(ctx)
}

func (d *DeferredClient) SecretGet(ctx context.Context, name string) (*apiv1.Secret, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.SecretGet(ctx, name)
}

func (d *DeferredClient) SecretReveal(ctx context.Context, name string) (*apiv1.Secret, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.SecretReveal(ctx, name)
}

func (d *DeferredClient) SecretUpdate(ctx context.Context, name string, data map[string][]byte) (*apiv1.Secret, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.SecretUpdate(ctx, name, data)
}

func (d *DeferredClient) SecretDelete(ctx context.Context, name string) (*apiv1.Secret, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.SecretDelete(ctx, name)
}

func (d *DeferredClient) ContainerReplicaList(ctx context.Context, opts *ContainerReplicaListOptions) ([]apiv1.ContainerReplica, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.ContainerReplicaList(ctx, opts)
}

func (d *DeferredClient) ContainerReplicaGet(ctx context.Context, name string) (*apiv1.ContainerReplica, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.ContainerReplicaGet(ctx, name)
}

func (d *DeferredClient) ContainerReplicaDelete(ctx context.Context, name string) (*apiv1.ContainerReplica, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.ContainerReplicaDelete(ctx, name)
}

func (d *DeferredClient) ContainerReplicaExec(ctx context.Context, name string, args []string, tty bool, opts *ContainerReplicaExecOptions) (*term.ExecIO, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.ContainerReplicaExec(ctx, name, args, tty, opts)
}

func (d *DeferredClient) VolumeList(ctx context.Context) ([]apiv1.Volume, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.VolumeList(ctx)
}

func (d *DeferredClient) VolumeGet(ctx context.Context, name string) (*apiv1.Volume, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.VolumeGet(ctx, name)
}

func (d *DeferredClient) VolumeDelete(ctx context.Context, name string) (*apiv1.Volume, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.VolumeDelete(ctx, name)
}

func (d *DeferredClient) ImageList(ctx context.Context) ([]apiv1.Image, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.ImageList(ctx)
}

func (d *DeferredClient) ImageGet(ctx context.Context, name string) (*apiv1.Image, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.ImageGet(ctx, name)
}

func (d *DeferredClient) ImageDelete(ctx context.Context, name string, opts *ImageDeleteOptions) (*apiv1.Image, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.ImageDelete(ctx, name, opts)
}

func (d *DeferredClient) ImagePush(ctx context.Context, tagName string, opts *ImagePushOptions) (<-chan ImageProgress, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.ImagePush(ctx, tagName, opts)
}

func (d *DeferredClient) ImagePull(ctx context.Context, name string, opts *ImagePullOptions) (<-chan ImageProgress, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.ImagePull(ctx, name, opts)
}

func (d *DeferredClient) ImageTag(ctx context.Context, image, tag string) error {
	if err := d.create(); err != nil {
		return err
	}
	return d.Client.ImageTag(ctx, image, tag)
}

func (d *DeferredClient) ImageDetails(ctx context.Context, imageName string, opts *ImageDetailsOptions) (*ImageDetails, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.ImageDetails(ctx, imageName, opts)
}

func (d *DeferredClient) AcornImageBuildGet(ctx context.Context, name string) (*apiv1.AcornImageBuild, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.AcornImageBuildGet(ctx, name)
}

func (d *DeferredClient) AcornImageBuildList(ctx context.Context) ([]apiv1.AcornImageBuild, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.AcornImageBuildList(ctx)
}

func (d *DeferredClient) AcornImageBuildDelete(ctx context.Context, name string) (*apiv1.AcornImageBuild, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.AcornImageBuildDelete(ctx, name)
}

func (d *DeferredClient) AcornImageBuild(ctx context.Context, file string, opts *AcornImageBuildOptions) (*v1.AppImage, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.AcornImageBuild(ctx, file, opts)
}

func (d *DeferredClient) ProjectGet(ctx context.Context, name string) (*apiv1.Project, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.ProjectGet(ctx, name)
}

func (d *DeferredClient) ProjectList(ctx context.Context) ([]apiv1.Project, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.ProjectList(ctx)
}

func (d *DeferredClient) ProjectCreate(ctx context.Context, name string) (*apiv1.Project, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.ProjectCreate(ctx, name)
}

func (d *DeferredClient) ProjectDelete(ctx context.Context, name string) (*apiv1.Project, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.ProjectDelete(ctx, name)
}

func (d *DeferredClient) WorkloadClassGet(ctx context.Context, name string) (*apiv1.WorkloadClass, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.WorkloadClassGet(ctx, name)
}

func (d *DeferredClient) WorkloadClassList(ctx context.Context) ([]apiv1.WorkloadClass, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.WorkloadClassList(ctx)
}

func (d *DeferredClient) Info(ctx context.Context) ([]apiv1.Info, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.Info(ctx)
}

func (d *DeferredClient) VolumeClassList(ctx context.Context) ([]apiv1.VolumeClass, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.VolumeClassList(ctx)
}

func (d *DeferredClient) VolumeClassGet(ctx context.Context, name string) (*apiv1.VolumeClass, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.VolumeClassGet(ctx, name)
}

func (d *DeferredClient) GetProject() string {
	return d.Project
}

func (d *DeferredClient) GetNamespace() string {
	return d.Namespace
}

func (d *DeferredClient) GetClient() (client.WithWatch, error) {
	if err := d.create(); err != nil {
		return nil, err
	}
	return d.Client.GetClient()
}
