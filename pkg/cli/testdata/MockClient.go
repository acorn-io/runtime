package testdata

import (
	"context"
	"fmt"
	"net"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/client/term"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type MockClientFactory struct {
	AppList       []apiv1.App
	ContainerList []apiv1.ContainerReplica
}

func (dc *MockClientFactory) CreateDefault() (client.Client, error) {
	return &MockClient{
		Apps:       dc.AppList,
		Containers: dc.ContainerList,
	}, nil
}

type MockClient struct {
	Apps       []apiv1.App
	Containers []apiv1.ContainerReplica
}

func (m *MockClient) AppPullImage(ctx context.Context, name string) error {
	return nil
}

func (m *MockClient) AppList(ctx context.Context) ([]apiv1.App, error) {
	if m.Apps != nil {
		return m.Apps, nil
	}
	return []apiv1.App{{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{Name: "found"},
		Spec:       v1.AppInstanceSpec{Secrets: []v1.SecretBinding{{Secret: "found.secret", Target: "found"}}},
		Status:     v1.AppInstanceStatus{},
	}}, nil
}

func (m *MockClient) AppDelete(ctx context.Context, name string) (*apiv1.App, error) {
	switch name {
	case "dne":
		return nil, fmt.Errorf("error: app %s does not exist", name)
	case "found":
		return &apiv1.App{}, nil
	case "found.container":
		return &apiv1.App{}, nil
	}
	return nil, nil
}

func (m *MockClient) AppGet(ctx context.Context, name string) (*apiv1.App, error) {
	switch name {
	case "dne":
		return nil, fmt.Errorf("error: app %s does not exist", name)
	case "found":
		return &apiv1.App{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{Name: "found"},
			Spec:       v1.AppInstanceSpec{Secrets: []v1.SecretBinding{{Secret: "found.secret", Target: "found"}}},
			Status:     v1.AppInstanceStatus{Ready: true},
		}, nil
	case "found.container":
		return &apiv1.App{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{Name: "found.container"},
			Spec:       v1.AppInstanceSpec{Secrets: []v1.SecretBinding{{Secret: "found.secret", Target: "found"}}},
			Status:     v1.AppInstanceStatus{},
		}, nil
	}
	return nil, nil
}

func (m *MockClient) AppStop(ctx context.Context, name string) error {
	switch name {
	case "dne":
		return fmt.Errorf("error: app %s does not exist", name)
	case "found":
		return nil
	case "found.container":
		return nil
	}
	return fmt.Errorf("error: app %s does not exist", name)
}

func (m *MockClient) AppStart(ctx context.Context, name string) error {
	switch name {
	case "dne":
		return fmt.Errorf("error: app %s does not exist", name)
	case "found":
		return nil
	case "found.container":
		return nil
	}
	return fmt.Errorf("error: app %s does not exist", name)
}

func (m *MockClient) AppRun(ctx context.Context, image string, opts *client.AppRunOptions) (*apiv1.App, error) {
	switch image {
	case "dne":
		return nil, fmt.Errorf("error: app %s does not exist", image)
	case "found":
		return &apiv1.App{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{Name: "found"},
			Spec:       v1.AppInstanceSpec{Secrets: []v1.SecretBinding{v1.SecretBinding{Secret: "found.secret", Target: "found"}}},
			Status:     v1.AppInstanceStatus{Ready: true},
		}, nil
	case "found.container":
		return &apiv1.App{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{Name: "found.container"},
			Spec:       v1.AppInstanceSpec{Secrets: []v1.SecretBinding{v1.SecretBinding{Secret: "found.secret", Target: "found"}}},
			Status:     v1.AppInstanceStatus{},
		}, nil
	}
	return nil, fmt.Errorf("error: app %s does not exist", image)
}

func (m *MockClient) AppUpdate(ctx context.Context, name string, opts *client.AppUpdateOptions) (*apiv1.App, error) {
	switch name {
	case "dne":
		return nil, fmt.Errorf("error: app %s does not exist", name)
	case "found":
		return &apiv1.App{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{Name: "found"},
			Spec:       v1.AppInstanceSpec{Secrets: []v1.SecretBinding{v1.SecretBinding{Secret: "found.secret", Target: "found"}}},
			Status:     v1.AppInstanceStatus{},
		}, nil
	case "found.container":
		return &apiv1.App{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{Name: "found.container"},
			Spec:       v1.AppInstanceSpec{Secrets: []v1.SecretBinding{v1.SecretBinding{Secret: "found.secret", Target: "found"}}},
			Status:     v1.AppInstanceStatus{},
		}, nil
	}
	return nil, fmt.Errorf("error: app %s does not exist", name)
}

func (m *MockClient) AppLog(ctx context.Context, name string, opts *client.LogOptions) (<-chan apiv1.LogMessage, error) {
	switch name {
	case "found":
		progresses := make(chan apiv1.LogMessage)
		close(progresses)
		return progresses, nil
	case "dne":
		progresses := make(chan apiv1.LogMessage)
		close(progresses)
		return progresses, fmt.Errorf("error: tag %s does not exist", name)
	default:
		progresses := make(chan apiv1.LogMessage)
		close(progresses)
		return progresses, fmt.Errorf("error: tag %s does not exist", name)
	}
}

func (m *MockClient) CredentialCreate(ctx context.Context, serverAddress, username, password string, skipChecks bool) (*apiv1.Credential, error) {
	return nil, nil
}

func (m *MockClient) CredentialList(ctx context.Context) ([]apiv1.Credential, error) {
	return []apiv1.Credential{apiv1.Credential{
		TypeMeta:      metav1.TypeMeta{},
		ObjectMeta:    metav1.ObjectMeta{Name: "test-cred"},
		ServerAddress: "test-server-address",
		Username:      "",
		Password:      nil,
	}}, nil

}

func (m *MockClient) CredentialGet(ctx context.Context, serverAddress string) (*apiv1.Credential, error) {
	switch serverAddress {
	case "dne":
		return nil, fmt.Errorf("error: cred %s does not exist", serverAddress)
	case "found":
		return &apiv1.Credential{
			TypeMeta:      metav1.TypeMeta{},
			ObjectMeta:    metav1.ObjectMeta{Name: "test-cred"},
			ServerAddress: "test-server-address",
			Username:      "",
			Password:      nil,
		}, nil
	default:
		return nil, fmt.Errorf("error: cred %s does not exist", serverAddress)
	}

}

func (m *MockClient) CredentialUpdate(ctx context.Context, serverAddress, username, password string, skipChecks bool) (*apiv1.Credential, error) {
	return nil, nil
}

func (m *MockClient) CredentialDelete(ctx context.Context, serverAddress string) (*apiv1.Credential, error) {
	return nil, nil
}

func (m *MockClient) SecretCreate(ctx context.Context, name, secretType string, data map[string][]byte) (*apiv1.Secret, error) {
	return nil, nil
}

func (m *MockClient) SecretList(ctx context.Context) ([]apiv1.Secret, error) {
	return []apiv1.Secret{apiv1.Secret{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{Name: "found.secret"},
		Type:       "",
		Data:       nil,
		Keys:       nil,
	}}, nil
}

func (m *MockClient) SecretGet(ctx context.Context, name string) (*apiv1.Secret, error) {
	switch name {
	case "dne":
		return nil, fmt.Errorf("error: Secret %s does not exist", name)
	case "found.secret":
		return &apiv1.Secret{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{Name: "found.secret"},
			Type:       "",
			Data:       nil,
			Keys:       nil,
		}, nil
	}
	return nil, nil
}

func (m *MockClient) SecretReveal(ctx context.Context, name string) (*apiv1.Secret, error) {
	switch name {
	case "dne":
		return nil, fmt.Errorf("error: Secret %s does not exist", name)
	case "found.secret":
		return &apiv1.Secret{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{Name: "found.secret"},
			Type:       "",
			Data:       nil,
			Keys:       nil,
		}, nil
	}
	return nil, nil
}

func (m *MockClient) SecretUpdate(ctx context.Context, name string, data map[string][]byte) (*apiv1.Secret, error) {
	return nil, nil
}

func (m *MockClient) SecretDelete(ctx context.Context, name string) (*apiv1.Secret, error) {
	switch name {
	case "dne":
		return nil, nil
	case "found.secret":
		return &apiv1.Secret{}, nil
	}
	return nil, nil
}

func (m *MockClient) ContainerReplicaList(ctx context.Context, opts *client.ContainerReplicaListOptions) ([]apiv1.ContainerReplica, error) {
	if m.Containers != nil {
		if opts == nil {
			return m.Containers, nil
		}
		// Do the filtering to make testing simpler
		result := make([]apiv1.ContainerReplica, 0, len(m.Containers))
		for _, c := range m.Containers {
			if c.Spec.AppName == opts.App {
				result = append(result, c)
			}
		}
		return result, nil
	}
	return []apiv1.ContainerReplica{{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{Name: "found.container"},
		Spec:       apiv1.ContainerReplicaSpec{AppName: "found"},
		Status:     apiv1.ContainerReplicaStatus{},
	}}, nil
}

func (m *MockClient) ContainerReplicaGet(ctx context.Context, name string) (*apiv1.ContainerReplica, error) {
	switch name {
	case "dne":
		return nil, fmt.Errorf("error: container %s does not exist", name)
	case "found":
		return &apiv1.ContainerReplica{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{Name: "found.container"},
			Spec:       apiv1.ContainerReplicaSpec{AppName: "found"},
			Status:     apiv1.ContainerReplicaStatus{},
		}, nil

	case "found.container":
		return &apiv1.ContainerReplica{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{Name: "found.container"},
			Spec:       apiv1.ContainerReplicaSpec{AppName: "found"},
			Status:     apiv1.ContainerReplicaStatus{},
		}, nil
	}
	return nil, nil
}

func (m *MockClient) ContainerReplicaDelete(ctx context.Context, name string) (*apiv1.ContainerReplica, error) {
	switch name {
	case "dne":
		return nil, nil
	case "found.container":
		return &apiv1.ContainerReplica{}, nil
	}
	return nil, nil
}

func (m *MockClient) ContainerReplicaExec(ctx context.Context, name string, args []string, tty bool, opts *client.ContainerReplicaExecOptions) (*term.ExecIO, error) {
	return nil, nil
}

func (m *MockClient) VolumeList(ctx context.Context) ([]apiv1.Volume, error) {
	return []apiv1.Volume{apiv1.Volume{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{Name: "found.volume"},
		Spec:       apiv1.VolumeSpec{},
		Status:     apiv1.VolumeStatus{AppName: "found", VolumeName: "found.volume"},
	}}, nil
}

func (m *MockClient) VolumeGet(ctx context.Context, name string) (*apiv1.Volume, error) {
	switch name {
	case "dne":
		return nil, fmt.Errorf("error: volume %s does not exist", name)
	case "found.volume":
		return &apiv1.Volume{TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{Name: "found.volume"},
			Spec:       apiv1.VolumeSpec{},
			Status:     apiv1.VolumeStatus{AppName: "found", VolumeName: "found.volume"},
		}, nil
	}
	return nil, nil
}

func (m *MockClient) VolumeDelete(ctx context.Context, name string) (*apiv1.Volume, error) {
	switch name {
	case "dne":
		return nil, nil
	case "found.volume":
		return &apiv1.Volume{}, nil
	}
	return nil, nil
}

func (m *MockClient) ImageList(ctx context.Context) ([]apiv1.Image, error) {
	return []apiv1.Image{{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{Name: "found-image1234567"},
		Tags:       []string{"testtag:latest"},
		Digest:     "1234567890asdfghkl",
	}, {
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{Name: "found-image-no-tag"},
		Digest:     "lkjhgfdsa0987654321",
	}, {
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{Name: "found-image-two-tags1234567"},
		Tags:       []string{"testtag1:latest", "testtag2:v1"},
		Digest:     "lkjhgfdsa1234567890",
	}}, nil
}

func (m *MockClient) ImageGet(ctx context.Context, name string) (*apiv1.Image, error) {
	switch name {
	case "dne":
		return nil, nil
	case "found.image":
		return &apiv1.Image{}, nil
	case "found.image-two-tags":
		return &apiv1.Image{}, nil
	}
	return nil, nil
}

func (m *MockClient) ImageDelete(ctx context.Context, name string, opts *client.ImageDeleteOptions) (*apiv1.Image, error) {
	switch name {
	case "dne":
		return nil, nil
	case "found-image1234567":
		return &apiv1.Image{}, nil
	case "found-image-two-tags1234567":
		if !opts.Force {
			return nil, fmt.Errorf("unable to delete %s (must be forced) - image is referenced in multiple repositories", name)
		} else {
			return &apiv1.Image{TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: "found-image-two-tags1234567"},
				Tags:       []string{"testtag1:latest", "testtag2:latest"},
			}, nil
		}
	}
	return nil, nil
}

func (m *MockClient) ImagePush(ctx context.Context, tagName string, opts *client.ImagePushOptions) (<-chan client.ImageProgress, error) {
	switch tagName {
	case "found":
		progresses := make(chan client.ImageProgress)
		close(progresses)
		return progresses, nil

	case "dne":
		progresses := make(chan client.ImageProgress)
		close(progresses)
		return progresses, fmt.Errorf("error: tag %s does not exist", tagName)
	default:
		progresses := make(chan client.ImageProgress)
		close(progresses)
		return progresses, fmt.Errorf("error: tag %s does not exist", tagName)
	}

}

func (m *MockClient) ImagePull(ctx context.Context, name string, opts *client.ImagePullOptions) (<-chan client.ImageProgress, error) {
	switch name {
	case "found":
		progresses := make(chan client.ImageProgress)
		close(progresses)
		return progresses, nil
	case "dne":
		progresses := make(chan client.ImageProgress)
		close(progresses)
		return progresses, fmt.Errorf("error: tag %s does not exist", name)
	default:
		progresses := make(chan client.ImageProgress)
		close(progresses)
		return progresses, fmt.Errorf("error: tag %s does not exist", name)
	}
}

func (m *MockClient) ImageTag(ctx context.Context, image, tag string) error {
	switch image {
	case "dne":
		return fmt.Errorf("error: tag %s does not exist", image)
	case "source":
		return nil
	}
	return nil
}

func (m *MockClient) ImageDetails(ctx context.Context, imageName string, opts *client.ImageDetailsOptions) (*client.ImageDetails, error) {
	return &client.ImageDetails{
		AppImage: v1.AppImage{ID: imageName, ImageData: v1.ImagesData{
			Containers: map[string]v1.ContainerData{"test-image-running-container": v1.ContainerData{
				Image:    "test-image-running-container",
				Sidecars: nil,
			}},
			Jobs:   nil,
			Images: nil,
		}},
		AppSpec:    nil,
		Params:     nil,
		ParseError: "",
	}, nil
}

func (m *MockClient) BuilderCreate(ctx context.Context) (*apiv1.Builder, error) { return nil, nil }

func (m *MockClient) BuilderGet(ctx context.Context) (*apiv1.Builder, error) { return nil, nil }

func (m *MockClient) BuilderDelete(ctx context.Context) (*apiv1.Builder, error) { return nil, nil }

func (m *MockClient) BuilderDialer(ctx context.Context) (func(ctx context.Context) (net.Conn, error), error) {
	return nil, nil
}
func (m *MockClient) BuilderRegistryDialer(ctx context.Context) (func(ctx context.Context) (net.Conn, error), error) {
	return nil, nil
}

func (m *MockClient) Info(ctx context.Context) (*apiv1.Info, error) {
	return &apiv1.Info{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Spec:       apiv1.InfoSpec{},
	}, nil
}

func (m *MockClient) GetNamespace() string { return "" }

func (m *MockClient) GetClient() kclient.WithWatch { return nil }

func (m *MockClient) PromptUser(s string) error {
	return nil
}

func (m *MockClient) AppConfirmUpgrade(ctx context.Context, name string) error {
	return nil
}

func (m *MockClient) AcornImageBuildGet(ctx context.Context, name string) (*apiv1.AcornImageBuild, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockClient) AcornImageBuildList(ctx context.Context) ([]apiv1.AcornImageBuild, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockClient) AcornImageBuildDelete(ctx context.Context, name string) (*apiv1.AcornImageBuild, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockClient) AcornImageBuild(ctx context.Context, file string, opts *client.AcornImageBuildOptions) (*v1.AppImage, error) {
	//TODO implement me
	panic("implement me")
}
