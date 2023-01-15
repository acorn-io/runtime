package client

import (
	"context"
	"reflect"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client/term"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type MultiClient struct {
	project   string
	namespace string
	factory   ProjectClientFactory
}

func NewMultiClient(project, namespace string, factory ProjectClientFactory) *MultiClient {
	return &MultiClient{
		project:   project,
		namespace: namespace,
		factory:   factory,
	}
}

type ProjectClientFactory interface {
	ForProject(ctx context.Context, project string) (Client, error)
	List(ctx context.Context) ([]Client, error)
	DefaultProject() string
}

type ObjectPointer[T any] interface {
	kclient.Object
	*T
}

func aggregate[T any, V ObjectPointer[T]](ctx context.Context, factory ProjectClientFactory, cb func(client Client) ([]T, error)) ([]T, error) {
	var result []T
	clients, err := factory.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, client := range clients {
		items, err := cb(client)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if client.GetProject() != factory.DefaultProject() {
				p := (V)(&item)
				p.SetName(client.GetProject() + "/" + p.GetName())
			}
			result = append(result, item)
		}
	}
	return result, nil
}

func isNil(obj kclient.Object) bool {
	return obj == nil || reflect.ValueOf(obj).IsNil()
}

func onOne[T kclient.Object](ctx context.Context, factory ProjectClientFactory, name string, cb func(name string, client Client) (T, error)) (T, error) {
	var (
		result      T
		projectName = ""
	)
	i := strings.LastIndex(name, "/")
	if i != -1 {
		projectName = name[0:i]
		name = name[i+1:]
	}
	client, err := factory.ForProject(ctx, projectName)
	if err != nil {
		return result, err
	}

	result, err = cb(name, client)
	if err != nil || isNil(result) {
		return result, err
	}
	if client.GetProject() != factory.DefaultProject() {
		result.SetName(client.GetProject() + "/" + result.GetName())
	}
	return result, nil
}

func (m *MultiClient) AppList(ctx context.Context) ([]apiv1.App, error) {
	return aggregate(ctx, m.factory, func(client Client) ([]apiv1.App, error) {
		return client.AppList(ctx)
	})
}

func (m *MultiClient) AppDelete(ctx context.Context, name string) (*apiv1.App, error) {
	return onOne(ctx, m.factory, name, func(name string, c Client) (*apiv1.App, error) {
		return c.AppDelete(ctx, name)
	})
}

func (m *MultiClient) AppGet(ctx context.Context, name string) (*apiv1.App, error) {
	return onOne(ctx, m.factory, name, func(name string, c Client) (*apiv1.App, error) {
		return c.AppGet(ctx, name)
	})
}

func (m *MultiClient) AppStop(ctx context.Context, name string) error {
	_, err := onOne(ctx, m.factory, name, func(name string, c Client) (*apiv1.App, error) {
		return &apiv1.App{}, c.AppStop(ctx, name)
	})
	return err
}

func (m *MultiClient) AppStart(ctx context.Context, name string) error {
	_, err := onOne(ctx, m.factory, name, func(name string, c Client) (*apiv1.App, error) {
		return &apiv1.App{}, c.AppStart(ctx, name)
	})
	return err
}

func (m *MultiClient) AppRun(ctx context.Context, image string, opts *AppRunOptions) (*apiv1.App, error) {
	name := ""
	if opts != nil {
		name = opts.Name
	}
	return onOne(ctx, m.factory, name, func(name string, c Client) (*apiv1.App, error) {
		opts.Name = name
		return c.AppRun(ctx, image, opts)
	})
}

func (m *MultiClient) AppUpdate(ctx context.Context, name string, opts *AppUpdateOptions) (*apiv1.App, error) {
	return onOne(ctx, m.factory, name, func(name string, c Client) (*apiv1.App, error) {
		return c.AppUpdate(ctx, name, opts)
	})
}

func (m *MultiClient) AppLog(ctx context.Context, name string, opts *LogOptions) (<-chan apiv1.LogMessage, error) {
	var (
		msgs    <-chan apiv1.LogMessage
		project string
		err     error
	)
	_, err = onOne(ctx, m.factory, name, func(name string, c Client) (kclient.Object, error) {
		project = c.GetProject()
		msgs, err = c.AppLog(ctx, name, opts)
		return &apiv1.App{}, err
	})
	if err != nil {
		return nil, err
	}
	result := make(chan apiv1.LogMessage)
	go func() {
		defer close(result)
		for msg := range msgs {
			msg.AppName = project + "/" + msg.AppName
			result <- msg
		}
	}()
	return result, nil
}

func (m *MultiClient) AppConfirmUpgrade(ctx context.Context, name string) error {
	_, err := onOne(ctx, m.factory, name, func(name string, c Client) (*apiv1.App, error) {
		return &apiv1.App{}, c.AppConfirmUpgrade(ctx, name)
	})
	return err
}

func (m *MultiClient) AppPullImage(ctx context.Context, name string) error {
	_, err := onOne(ctx, m.factory, name, func(name string, c Client) (*apiv1.App, error) {
		return &apiv1.App{}, c.AppPullImage(ctx, name)
	})
	return err
}

func (m *MultiClient) CredentialCreate(ctx context.Context, serverAddress, username, password string, skipChecks bool) (*apiv1.Credential, error) {
	return onOne(ctx, m.factory, serverAddress, func(name string, c Client) (*apiv1.Credential, error) {
		return c.CredentialCreate(ctx, name, username, password, skipChecks)
	})
}

func (m *MultiClient) CredentialList(ctx context.Context) ([]apiv1.Credential, error) {
	return aggregate(ctx, m.factory, func(client Client) ([]apiv1.Credential, error) {
		return client.CredentialList(ctx)
	})
}

func (m *MultiClient) CredentialGet(ctx context.Context, serverAddress string) (*apiv1.Credential, error) {
	return onOne(ctx, m.factory, serverAddress, func(name string, c Client) (*apiv1.Credential, error) {
		return c.CredentialGet(ctx, name)
	})
}

func (m *MultiClient) CredentialUpdate(ctx context.Context, serverAddress, username, password string, skipChecks bool) (*apiv1.Credential, error) {
	return onOne(ctx, m.factory, serverAddress, func(name string, c Client) (*apiv1.Credential, error) {
		return c.CredentialUpdate(ctx, name, username, password, skipChecks)
	})
}

func (m *MultiClient) CredentialDelete(ctx context.Context, serverAddress string) (*apiv1.Credential, error) {
	return onOne(ctx, m.factory, serverAddress, func(name string, c Client) (*apiv1.Credential, error) {
		return c.CredentialDelete(ctx, name)
	})
}

func (m *MultiClient) SecretCreate(ctx context.Context, name, secretType string, data map[string][]byte) (*apiv1.Secret, error) {
	return onOne(ctx, m.factory, name, func(name string, c Client) (*apiv1.Secret, error) {
		return c.SecretCreate(ctx, name, secretType, data)
	})
}

func (m *MultiClient) SecretList(ctx context.Context) ([]apiv1.Secret, error) {
	return aggregate(ctx, m.factory, func(c Client) ([]apiv1.Secret, error) {
		return c.SecretList(ctx)
	})
}

func (m *MultiClient) SecretGet(ctx context.Context, name string) (*apiv1.Secret, error) {
	return onOne(ctx, m.factory, name, func(name string, c Client) (*apiv1.Secret, error) {
		return c.SecretGet(ctx, name)
	})
}

func (m *MultiClient) SecretReveal(ctx context.Context, name string) (*apiv1.Secret, error) {
	return onOne(ctx, m.factory, name, func(name string, c Client) (*apiv1.Secret, error) {
		return c.SecretReveal(ctx, name)
	})
}

func (m *MultiClient) SecretUpdate(ctx context.Context, name string, data map[string][]byte) (*apiv1.Secret, error) {
	return onOne(ctx, m.factory, name, func(name string, c Client) (*apiv1.Secret, error) {
		return c.SecretUpdate(ctx, name, data)
	})
}

func (m *MultiClient) SecretDelete(ctx context.Context, name string) (*apiv1.Secret, error) {
	return onOne(ctx, m.factory, name, func(name string, c Client) (*apiv1.Secret, error) {
		return c.SecretDelete(ctx, name)
	})
}

func (m *MultiClient) ContainerReplicaList(ctx context.Context, opts *ContainerReplicaListOptions) ([]apiv1.ContainerReplica, error) {
	return aggregate(ctx, m.factory, func(c Client) ([]apiv1.ContainerReplica, error) {
		return c.ContainerReplicaList(ctx, opts)
	})
}

func (m *MultiClient) ContainerReplicaGet(ctx context.Context, name string) (*apiv1.ContainerReplica, error) {
	return onOne(ctx, m.factory, name, func(name string, c Client) (*apiv1.ContainerReplica, error) {
		return c.ContainerReplicaGet(ctx, name)
	})
}

func (m *MultiClient) ContainerReplicaDelete(ctx context.Context, name string) (*apiv1.ContainerReplica, error) {
	return onOne(ctx, m.factory, name, func(name string, c Client) (*apiv1.ContainerReplica, error) {
		return c.ContainerReplicaDelete(ctx, name)
	})
}

func (m *MultiClient) ContainerReplicaExec(ctx context.Context, name string, args []string, tty bool, opts *ContainerReplicaExecOptions) (exec *term.ExecIO, err error) {
	_, err = onOne(ctx, m.factory, name, func(name string, c Client) (*apiv1.ContainerReplica, error) {
		exec, err = c.ContainerReplicaExec(ctx, name, args, tty, opts)
		return &apiv1.ContainerReplica{}, err
	})
	return exec, err
}

func (m *MultiClient) VolumeList(ctx context.Context) ([]apiv1.Volume, error) {
	return aggregate(ctx, m.factory, func(c Client) ([]apiv1.Volume, error) {
		return c.VolumeList(ctx)
	})
}

func (m *MultiClient) VolumeGet(ctx context.Context, name string) (*apiv1.Volume, error) {
	return onOne(ctx, m.factory, name, func(name string, c Client) (*apiv1.Volume, error) {
		return c.VolumeGet(ctx, name)
	})
}

func (m *MultiClient) VolumeDelete(ctx context.Context, name string) (*apiv1.Volume, error) {
	return onOne(ctx, m.factory, name, func(name string, c Client) (*apiv1.Volume, error) {
		return c.VolumeDelete(ctx, name)
	})
}

func (m *MultiClient) ImageList(ctx context.Context) ([]apiv1.Image, error) {
	c, err := m.factory.ForProject(ctx, m.factory.DefaultProject())
	if err != nil {
		return nil, err
	}
	return c.ImageList(ctx)
}

func (m *MultiClient) ImageGet(ctx context.Context, name string) (*apiv1.Image, error) {
	c, err := m.factory.ForProject(ctx, m.factory.DefaultProject())
	if err != nil {
		return nil, err
	}
	return c.ImageGet(ctx, name)
}

func (m *MultiClient) ImageDelete(ctx context.Context, name string, opts *ImageDeleteOptions) (*apiv1.Image, error) {
	c, err := m.factory.ForProject(ctx, m.factory.DefaultProject())
	if err != nil {
		return nil, err
	}
	return c.ImageDelete(ctx, name, opts)
}

func (m *MultiClient) ImagePush(ctx context.Context, tagName string, opts *ImagePushOptions) (result <-chan ImageProgress, err error) {
	c, err := m.factory.ForProject(ctx, m.factory.DefaultProject())
	if err != nil {
		return nil, err
	}
	return c.ImagePush(ctx, tagName, opts)
}

func (m *MultiClient) ImagePull(ctx context.Context, name string, opts *ImagePullOptions) (result <-chan ImageProgress, err error) {
	c, err := m.factory.ForProject(ctx, m.factory.DefaultProject())
	if err != nil {
		return nil, err
	}
	return c.ImagePull(ctx, name, opts)
}

func (m *MultiClient) ImageTag(ctx context.Context, image, tag string) error {
	c, err := m.factory.ForProject(ctx, m.factory.DefaultProject())
	if err != nil {
		return err
	}
	return c.ImageTag(ctx, image, tag)
}

func (m *MultiClient) ImageDetails(ctx context.Context, imageName string, opts *ImageDetailsOptions) (result *ImageDetails, err error) {
	c, err := m.factory.ForProject(ctx, m.factory.DefaultProject())
	if err != nil {
		return nil, err
	}
	return c.ImageDetails(ctx, imageName, opts)
}

func (m *MultiClient) AcornImageBuildGet(ctx context.Context, name string) (*apiv1.AcornImageBuild, error) {
	c, err := m.factory.ForProject(ctx, m.factory.DefaultProject())
	if err != nil {
		return nil, err
	}
	return c.AcornImageBuildGet(ctx, name)
}

func (m *MultiClient) AcornImageBuildList(ctx context.Context) ([]apiv1.AcornImageBuild, error) {
	c, err := m.factory.ForProject(ctx, m.factory.DefaultProject())
	if err != nil {
		return nil, err
	}
	return c.AcornImageBuildList(ctx)
}

func (m *MultiClient) AcornImageBuildDelete(ctx context.Context, name string) (*apiv1.AcornImageBuild, error) {
	c, err := m.factory.ForProject(ctx, m.factory.DefaultProject())
	if err != nil {
		return nil, err
	}
	return c.AcornImageBuildDelete(ctx, name)
}

func (m *MultiClient) AcornImageBuild(ctx context.Context, file string, opts *AcornImageBuildOptions) (result *v1.AppImage, err error) {
	c, err := m.factory.ForProject(ctx, m.factory.DefaultProject())
	if err != nil {
		return nil, err
	}
	return c.AcornImageBuild(ctx, file, opts)
}

func (m *MultiClient) ProjectList(ctx context.Context) ([]apiv1.Project, error) {
	return aggregate(ctx, m.factory, func(c Client) ([]apiv1.Project, error) {
		projs, err := c.ProjectList(ctx)
		for i := range projs {
			idx := strings.LastIndex(c.GetProject(), "/")
			if idx > 1 {
				projs[i].Name = c.GetProject()[:idx] + "/" + projs[i].Name
			}
		}
		return projs, err
	})
}

func (m *MultiClient) Info(ctx context.Context) (*apiv1.Info, error) {
	c, err := m.factory.ForProject(ctx, m.factory.DefaultProject())
	if err != nil {
		return nil, err
	}
	return c.Info(ctx)
}

func (m *MultiClient) GetProject() string {
	return m.project
}

func (m *MultiClient) GetNamespace() string {
	return m.namespace
}

func (m *MultiClient) GetClient() kclient.WithWatch {
	c, err := m.factory.ForProject(context.Background(), m.factory.DefaultProject())
	if err != nil {
		panic(err)
	}
	return c.GetClient()
}
