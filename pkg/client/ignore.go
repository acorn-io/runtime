package client

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/AlecAivazis/survey/v2"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client/term"
	"github.com/acorn-io/acorn/pkg/install"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type IgnoreUninstalled struct {
	client Client
}

func (c IgnoreUninstalled) PromptUser(obj string) error {
	return c.client.PromptUser(obj)
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
			installErr := install.Install(ctx, install.DefaultImage(), &install.Options{})
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
					if cause.Type == v1.CauseTypeUnexpectedServerResponse {
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

func (c IgnoreUninstalled) GetNamespace() string {
	return c.client.GetNamespace()
}

func (c IgnoreUninstalled) GetClient() kclient.WithWatch {
	return c.client.GetClient()
}

func (c IgnoreUninstalled) AppList(ctx context.Context) ([]apiv1.App, error) {
	return ignoreUninstalled(c.client.AppList(ctx))
}

func (c IgnoreUninstalled) AppDelete(ctx context.Context, name string) (*apiv1.App, error) {
	return ignoreUninstalled(c.client.AppDelete(ctx, name))
}

func (c IgnoreUninstalled) AppGet(ctx context.Context, name string) (*apiv1.App, error) {
	return c.client.AppGet(ctx, name)
}

func (c IgnoreUninstalled) AppStop(ctx context.Context, name string) error {
	return c.client.AppStop(ctx, name)
}

func (c IgnoreUninstalled) AppStart(ctx context.Context, name string) error {
	return c.client.AppStart(ctx, name)
}

func (c IgnoreUninstalled) AppRun(ctx context.Context, image string, opts *AppRunOptions) (*apiv1.App, error) {
	return promptInstall(ctx, func() (*apiv1.App, error) {
		return c.client.AppRun(ctx, image, opts)
	})
}

func (c IgnoreUninstalled) AppUpdate(ctx context.Context, name string, opts *AppUpdateOptions) (*apiv1.App, error) {
	return c.client.AppUpdate(ctx, name, opts)
}

func (c IgnoreUninstalled) AppPullImage(ctx context.Context, name string) error {
	return c.client.AppPullImage(ctx, name)
}

func (c IgnoreUninstalled) AppConfirmUpgrade(ctx context.Context, name string) error {
	return c.client.AppConfirmUpgrade(ctx, name)
}

func (c *IgnoreUninstalled) AppLog(ctx context.Context, name string, opts *LogOptions) (<-chan apiv1.LogMessage, error) {
	return c.client.AppLog(ctx, name, opts)
}

func (c IgnoreUninstalled) ContainerReplicaList(ctx context.Context, opts *ContainerReplicaListOptions) ([]apiv1.ContainerReplica, error) {
	return ignoreUninstalled(c.client.ContainerReplicaList(ctx, opts))
}

func (c IgnoreUninstalled) ContainerReplicaGet(ctx context.Context, name string) (*apiv1.ContainerReplica, error) {
	return c.client.ContainerReplicaGet(ctx, name)
}

func (c IgnoreUninstalled) ContainerReplicaDelete(ctx context.Context, name string) (*apiv1.ContainerReplica, error) {
	return ignoreUninstalled(c.client.ContainerReplicaDelete(ctx, name))
}

func (c IgnoreUninstalled) ContainerReplicaExec(ctx context.Context, name string, args []string, tty bool, opts *ContainerReplicaExecOptions) (*term.ExecIO, error) {
	return c.client.ContainerReplicaExec(ctx, name, args, tty, opts)
}

func (c IgnoreUninstalled) VolumeList(ctx context.Context) ([]apiv1.Volume, error) {
	return ignoreUninstalled(c.client.VolumeList(ctx))
}

func (c IgnoreUninstalled) VolumeGet(ctx context.Context, name string) (*apiv1.Volume, error) {
	return c.client.VolumeGet(ctx, name)
}

func (c IgnoreUninstalled) VolumeDelete(ctx context.Context, name string) (*apiv1.Volume, error) {
	return ignoreUninstalled(c.client.VolumeDelete(ctx, name))
}

func (c IgnoreUninstalled) ImageList(ctx context.Context) ([]apiv1.Image, error) {
	return ignoreUninstalled(c.client.ImageList(ctx))
}

func (c IgnoreUninstalled) ImageGet(ctx context.Context, name string) (*apiv1.Image, error) {
	return c.client.ImageGet(ctx, name)
}

func (c IgnoreUninstalled) ImageDelete(ctx context.Context, name string) (*apiv1.Image, error) {
	return ignoreUninstalled(c.client.ImageDelete(ctx, name))
}

func (c IgnoreUninstalled) ImagePush(ctx context.Context, tagName string, opts *ImagePushOptions) (<-chan ImageProgress, error) {
	return promptInstall(ctx, func() (<-chan ImageProgress, error) {
		return c.client.ImagePush(ctx, tagName, opts)
	})
}

func (c IgnoreUninstalled) ImagePull(ctx context.Context, name string, opts *ImagePullOptions) (<-chan ImageProgress, error) {
	return promptInstall(ctx, func() (<-chan ImageProgress, error) {
		return c.client.ImagePull(ctx, name, opts)
	})
}

func (c IgnoreUninstalled) ImageTag(ctx context.Context, image, tag string) error {
	return c.client.ImageTag(ctx, image, tag)
}

func (c IgnoreUninstalled) ImageDetails(ctx context.Context, imageName string, opts *ImageDetailsOptions) (*ImageDetails, error) {
	return promptInstall(ctx, func() (*ImageDetails, error) {
		return c.client.ImageDetails(ctx, imageName, opts)
	})
}

func (c IgnoreUninstalled) BuilderCreate(ctx context.Context) (*apiv1.Builder, error) {
	return c.client.BuilderCreate(ctx)
}

func (c IgnoreUninstalled) BuilderGet(ctx context.Context) (*apiv1.Builder, error) {
	return ignoreUninstalled(c.client.BuilderGet(ctx))
}

func (c IgnoreUninstalled) BuilderDelete(ctx context.Context) (*apiv1.Builder, error) {
	return ignoreUninstalled(c.client.BuilderDelete(ctx))
}

func (c IgnoreUninstalled) BuilderDialer(ctx context.Context) (func(ctx context.Context) (net.Conn, error), error) {
	return promptInstall(ctx, func() (func(context.Context) (net.Conn, error), error) {
		return c.client.BuilderDialer(ctx)
	})
}

func (c IgnoreUninstalled) BuilderRegistryDialer(ctx context.Context) (func(ctx context.Context) (net.Conn, error), error) {
	return promptInstall(ctx, func() (func(context.Context) (net.Conn, error), error) {
		return c.client.BuilderRegistryDialer(ctx)
	})
}

func (c IgnoreUninstalled) CredentialCreate(ctx context.Context, serverAddress, username, password string, skipChecks bool) (*apiv1.Credential, error) {
	return promptInstall(ctx, func() (*apiv1.Credential, error) {
		return c.client.CredentialCreate(ctx, serverAddress, username, password, skipChecks)
	})
}

func (c IgnoreUninstalled) CredentialList(ctx context.Context) ([]apiv1.Credential, error) {
	return ignoreUninstalled(c.client.CredentialList(ctx))
}

func (c IgnoreUninstalled) CredentialGet(ctx context.Context, serverAddress string) (*apiv1.Credential, error) {
	return c.client.CredentialGet(ctx, serverAddress)
}

func (c IgnoreUninstalled) CredentialUpdate(ctx context.Context, serverAddress, username, password string, skipChecks bool) (*apiv1.Credential, error) {
	return c.client.CredentialUpdate(ctx, serverAddress, username, password, skipChecks)
}

func (c IgnoreUninstalled) CredentialDelete(ctx context.Context, serverAddress string) (*apiv1.Credential, error) {
	return c.client.CredentialDelete(ctx, serverAddress)
}

func (c IgnoreUninstalled) SecretCreate(ctx context.Context, name, secretType string, data map[string][]byte) (*apiv1.Secret, error) {
	return promptInstall(ctx, func() (*apiv1.Secret, error) {
		return c.client.SecretCreate(ctx, name, secretType, data)
	})
}

func (c IgnoreUninstalled) SecretList(ctx context.Context) ([]apiv1.Secret, error) {
	return ignoreUninstalled(c.client.SecretList(ctx))
}

func (c IgnoreUninstalled) SecretGet(ctx context.Context, name string) (*apiv1.Secret, error) {
	return c.client.SecretGet(ctx, name)
}

func (c IgnoreUninstalled) SecretReveal(ctx context.Context, name string) (*apiv1.Secret, error) {
	return c.client.SecretReveal(ctx, name)
}

func (c IgnoreUninstalled) SecretUpdate(ctx context.Context, name string, data map[string][]byte) (*apiv1.Secret, error) {
	return c.client.SecretUpdate(ctx, name, data)
}

func (c IgnoreUninstalled) SecretDelete(ctx context.Context, name string) (*apiv1.Secret, error) {
	return c.client.SecretDelete(ctx, name)
}

func (c IgnoreUninstalled) Info(ctx context.Context) (*apiv1.Info, error) {
	return promptInstall(ctx, func() (*apiv1.Info, error) {
		return c.client.Info(ctx)
	})
}
