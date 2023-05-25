package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client/term"
	"github.com/acorn-io/acorn/pkg/install"
	"github.com/acorn-io/acorn/pkg/system"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type IgnoreUninstalled struct {
	Client Client
}

type twoFunc[V any] func() (V, error)

func promptInstall[V any](ctx context.Context, f twoFunc[V]) (V, error) {
	v, err := f()
	if isNotInstalled(err) {
		var shouldInstall = false
		surveyErr := survey.AskOne(&survey.Confirm{
			Message: "Acorn is not installed, do you want to install it now?:",
			Default: false,
		}, &shouldInstall)
		if surveyErr != nil {
			return v, surveyErr
		}

		if shouldInstall {
			installErr := install.Install(ctx, system.DefaultImage(), &install.Options{})
			if installErr != nil {
				return v, installErr
			}
			v, err = f()
		} else {
			return v, fmt.Errorf("action aborted because Acorn is not installed")
		}
	}
	return v, err
}

func isNotInstalled(err error) bool {
	if kindErr := (*meta.NoKindMatchError)(nil); errors.As(err, &kindErr) {
		return true
	}
	if apierrors.IsNotFound(err) {
		if errStatus, ok := err.(*apierrors.StatusError); ok {
			if errStatus.ErrStatus.Details != nil {
				for _, cause := range errStatus.ErrStatus.Details.Causes {
					if cause.Type == metav1.CauseTypeUnexpectedServerResponse {
						return true
					}
				}
			}
		}
	}
	return false
}

func ignoreUninstalled[V any](arg V, err error) (V, error) {
	if isNotInstalled(err) {
		return arg, nil
	}
	return arg, err
}

func (c IgnoreUninstalled) GetProject() string {
	return c.Client.GetProject()
}

func (c IgnoreUninstalled) GetNamespace() string {
	return c.Client.GetNamespace()
}

func (c IgnoreUninstalled) GetClient() (kclient.WithWatch, error) {
	return c.Client.GetClient()
}

func (c IgnoreUninstalled) AppList(ctx context.Context) ([]apiv1.App, error) {
	return ignoreUninstalled(c.Client.AppList(ctx))
}

func (c IgnoreUninstalled) AppDelete(ctx context.Context, name string) (*apiv1.App, error) {
	return ignoreUninstalled(c.Client.AppDelete(ctx, name))
}

func (c IgnoreUninstalled) AppGet(ctx context.Context, name string) (*apiv1.App, error) {
	return c.Client.AppGet(ctx, name)
}

func (c IgnoreUninstalled) AppStop(ctx context.Context, name string) error {
	return c.Client.AppStop(ctx, name)
}

func (c IgnoreUninstalled) AppStart(ctx context.Context, name string) error {
	return c.Client.AppStart(ctx, name)
}

func (c IgnoreUninstalled) AppRun(ctx context.Context, image string, opts *AppRunOptions) (*apiv1.App, error) {
	return promptInstall(ctx, func() (*apiv1.App, error) {
		return c.Client.AppRun(ctx, image, opts)
	})
}

func (c IgnoreUninstalled) AppUpdate(ctx context.Context, name string, opts *AppUpdateOptions) (*apiv1.App, error) {
	return c.Client.AppUpdate(ctx, name, opts)
}

func (c IgnoreUninstalled) AppPullImage(ctx context.Context, name string) error {
	return c.Client.AppPullImage(ctx, name)
}

func (c IgnoreUninstalled) AppConfirmUpgrade(ctx context.Context, name string) error {
	return c.Client.AppConfirmUpgrade(ctx, name)
}

func (c *IgnoreUninstalled) AppLog(ctx context.Context, name string, opts *LogOptions) (<-chan apiv1.LogMessage, error) {
	return c.Client.AppLog(ctx, name, opts)
}

func (c *IgnoreUninstalled) DevSessionRenew(ctx context.Context, name string, client v1.DevSessionInstanceClient) error {
	return c.Client.DevSessionRenew(ctx, name, client)
}

func (c *IgnoreUninstalled) DevSessionRelease(ctx context.Context, name string) error {
	return c.Client.DevSessionRelease(ctx, name)
}

func (c IgnoreUninstalled) ContainerReplicaList(ctx context.Context, opts *ContainerReplicaListOptions) ([]apiv1.ContainerReplica, error) {
	return ignoreUninstalled(c.Client.ContainerReplicaList(ctx, opts))
}

func (c IgnoreUninstalled) ContainerReplicaGet(ctx context.Context, name string) (*apiv1.ContainerReplica, error) {
	return c.Client.ContainerReplicaGet(ctx, name)
}

func (c IgnoreUninstalled) ContainerReplicaDelete(ctx context.Context, name string) (*apiv1.ContainerReplica, error) {
	return ignoreUninstalled(c.Client.ContainerReplicaDelete(ctx, name))
}

func (c IgnoreUninstalled) ContainerReplicaExec(ctx context.Context, name string, args []string, tty bool, opts *ContainerReplicaExecOptions) (*term.ExecIO, error) {
	return c.Client.ContainerReplicaExec(ctx, name, args, tty, opts)
}

func (c IgnoreUninstalled) ContainerReplicaPortForward(ctx context.Context, name string, port int) (PortForwardDialer, error) {
	return c.Client.ContainerReplicaPortForward(ctx, name, port)
}

func (c IgnoreUninstalled) VolumeList(ctx context.Context) ([]apiv1.Volume, error) {
	return ignoreUninstalled(c.Client.VolumeList(ctx))
}

func (c IgnoreUninstalled) VolumeGet(ctx context.Context, name string) (*apiv1.Volume, error) {
	return c.Client.VolumeGet(ctx, name)
}

func (c IgnoreUninstalled) VolumeDelete(ctx context.Context, name string) (*apiv1.Volume, error) {
	return ignoreUninstalled(c.Client.VolumeDelete(ctx, name))
}

func (c IgnoreUninstalled) ImageList(ctx context.Context) ([]apiv1.Image, error) {
	return ignoreUninstalled(c.Client.ImageList(ctx))
}

func (c IgnoreUninstalled) ImageGet(ctx context.Context, name string) (*apiv1.Image, error) {
	return c.Client.ImageGet(ctx, name)
}

func (c IgnoreUninstalled) ImageDelete(ctx context.Context, name string, opts *ImageDeleteOptions) (*apiv1.Image, []string, error) {
	i, t, err := c.Client.ImageDelete(ctx, name, opts)
	_, err = ignoreUninstalled("", err)
	return i, t, err
}

func (c IgnoreUninstalled) ImagePush(ctx context.Context, tagName string, opts *ImagePushOptions) (<-chan ImageProgress, error) {
	return promptInstall(ctx, func() (<-chan ImageProgress, error) {
		return c.Client.ImagePush(ctx, tagName, opts)
	})
}

func (c IgnoreUninstalled) ImagePull(ctx context.Context, name string, opts *ImagePullOptions) (<-chan ImageProgress, error) {
	return promptInstall(ctx, func() (<-chan ImageProgress, error) {
		return c.Client.ImagePull(ctx, name, opts)
	})
}

func (c IgnoreUninstalled) ImageTag(ctx context.Context, image, tag string) error {
	return c.Client.ImageTag(ctx, image, tag)
}

func (c IgnoreUninstalled) ImageDetails(ctx context.Context, imageName string, opts *ImageDetailsOptions) (*ImageDetails, error) {
	return promptInstall(ctx, func() (*ImageDetails, error) {
		return c.Client.ImageDetails(ctx, imageName, opts)
	})
}

func (c IgnoreUninstalled) AcornImageBuild(ctx context.Context, file string, opts *AcornImageBuildOptions) (*v1.AppImage, error) {
	return promptInstall(ctx, func() (*v1.AppImage, error) {
		return c.Client.AcornImageBuild(ctx, file, opts)
	})
}

func (c IgnoreUninstalled) AcornImageBuildGet(ctx context.Context, name string) (*apiv1.AcornImageBuild, error) {
	return ignoreUninstalled(c.Client.AcornImageBuildGet(ctx, name))
}

func (c IgnoreUninstalled) AcornImageBuildDelete(ctx context.Context, name string) (*apiv1.AcornImageBuild, error) {
	return ignoreUninstalled(c.Client.AcornImageBuildDelete(ctx, name))
}

func (c IgnoreUninstalled) AcornImageBuildList(ctx context.Context) ([]apiv1.AcornImageBuild, error) {
	return ignoreUninstalled(c.Client.AcornImageBuildList(ctx))
}

func (c IgnoreUninstalled) CredentialCreate(ctx context.Context, serverAddress, username, password string, skipChecks bool) (*apiv1.Credential, error) {
	return promptInstall(ctx, func() (*apiv1.Credential, error) {
		return c.Client.CredentialCreate(ctx, serverAddress, username, password, skipChecks)
	})
}

func (c IgnoreUninstalled) CredentialList(ctx context.Context) ([]apiv1.Credential, error) {
	return ignoreUninstalled(c.Client.CredentialList(ctx))
}

func (c IgnoreUninstalled) CredentialGet(ctx context.Context, serverAddress string) (*apiv1.Credential, error) {
	return c.Client.CredentialGet(ctx, serverAddress)
}

func (c IgnoreUninstalled) CredentialUpdate(ctx context.Context, serverAddress, username, password string, skipChecks bool) (*apiv1.Credential, error) {
	return c.Client.CredentialUpdate(ctx, serverAddress, username, password, skipChecks)
}

func (c IgnoreUninstalled) CredentialDelete(ctx context.Context, serverAddress string) (*apiv1.Credential, error) {
	return c.Client.CredentialDelete(ctx, serverAddress)
}

func (c IgnoreUninstalled) SecretCreate(ctx context.Context, name, secretType string, data map[string][]byte) (*apiv1.Secret, error) {
	return promptInstall(ctx, func() (*apiv1.Secret, error) {
		return c.Client.SecretCreate(ctx, name, secretType, data)
	})
}

func (c IgnoreUninstalled) SecretList(ctx context.Context) ([]apiv1.Secret, error) {
	return ignoreUninstalled(c.Client.SecretList(ctx))
}

func (c IgnoreUninstalled) SecretGet(ctx context.Context, name string) (*apiv1.Secret, error) {
	return c.Client.SecretGet(ctx, name)
}

func (c IgnoreUninstalled) SecretReveal(ctx context.Context, name string) (*apiv1.Secret, error) {
	return c.Client.SecretReveal(ctx, name)
}

func (c IgnoreUninstalled) SecretUpdate(ctx context.Context, name string, data map[string][]byte) (*apiv1.Secret, error) {
	return c.Client.SecretUpdate(ctx, name, data)
}

func (c IgnoreUninstalled) SecretDelete(ctx context.Context, name string) (*apiv1.Secret, error) {
	return c.Client.SecretDelete(ctx, name)
}

func (c *IgnoreUninstalled) ProjectGet(ctx context.Context, name string) (*apiv1.Project, error) {
	return c.Client.ProjectGet(ctx, name)
}

func (c *IgnoreUninstalled) ProjectList(ctx context.Context) ([]apiv1.Project, error) {
	return ignoreUninstalled(c.Client.ProjectList(ctx))
}

func (c *IgnoreUninstalled) ProjectCreate(ctx context.Context, name, defaultRegion string, supportedRegions []string) (*apiv1.Project, error) {
	return promptInstall(ctx, func() (*apiv1.Project, error) {
		return c.Client.ProjectCreate(ctx, name, defaultRegion, supportedRegions)
	})
}

func (c *IgnoreUninstalled) ProjectUpdate(ctx context.Context, project *apiv1.Project, defaultRegion string, supportedRegions []string) (*apiv1.Project, error) {
	return promptInstall(ctx, func() (*apiv1.Project, error) {
		return c.Client.ProjectUpdate(ctx, project, defaultRegion, supportedRegions)
	})
}

func (c *IgnoreUninstalled) ProjectDelete(ctx context.Context, name string) (*apiv1.Project, error) {
	return ignoreUninstalled(c.Client.ProjectDelete(ctx, name))
}

func (c IgnoreUninstalled) Info(ctx context.Context) ([]apiv1.Info, error) {
	return promptInstall(ctx, func() ([]apiv1.Info, error) {
		return c.Client.Info(ctx)
	})
}

func (c IgnoreUninstalled) VolumeClassList(ctx context.Context) ([]apiv1.VolumeClass, error) {
	return ignoreUninstalled(c.Client.VolumeClassList(ctx))
}

func (c IgnoreUninstalled) VolumeClassGet(ctx context.Context, name string) (*apiv1.VolumeClass, error) {
	return c.Client.VolumeClassGet(ctx, name)
}
