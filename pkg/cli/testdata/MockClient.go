package testdata

import (
	"context"
	"fmt"
	"net"

	"github.com/acorn-io/baaah/pkg/typed"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/client/term"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/project"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type MockClientFactoryManual struct {
	AcornConfig string
	Client      client.Client
}

func (dc *MockClientFactoryManual) Options() project.Options {
	return project.Options{
		AcornConfig: dc.AcornConfig,
	}
}

func (dc *MockClientFactoryManual) CreateDefault() (client.Client, error) {
	return dc.Client, nil
}

func (dc *MockClientFactoryManual) CreateWithAllProjects() (client.Client, error) {
	return dc.Client, nil
}

type MockClientFactory struct {
	AcornConfig      string
	AppList          []apiv1.App
	AppItem          *apiv1.App
	ContainerList    []apiv1.ContainerReplica
	ContainerItem    *apiv1.ContainerReplica
	JobList          []apiv1.Job
	JobItem          *apiv1.Job
	CredentialList   []apiv1.Credential
	CredentialItem   *apiv1.Credential
	VolumeList       []apiv1.Volume
	VolumeItem       *apiv1.Volume
	SecretList       []apiv1.Secret
	SecretItem       *apiv1.Secret
	ImageList        []apiv1.Image
	ImageItem        *apiv1.Image
	ProjectList      []apiv1.Project
	ProjectItem      *apiv1.Project
	VolumeClassList  []apiv1.VolumeClass
	VolumeClassItem  *apiv1.VolumeClass
	ComputeClassList []apiv1.ComputeClass
	ComputeClassItem *apiv1.ComputeClass
	RegionList       []apiv1.Region
	RegionItem       *apiv1.Region
	EventList        []apiv1.Event
	EventItem        *apiv1.Event
}

func (dc *MockClientFactory) Options() project.Options {
	return project.Options{
		AcornConfig: dc.AcornConfig,
	}
}

func (dc *MockClientFactory) CreateDefault() (client.Client, error) {
	return &MockClient{
		Apps:             dc.AppList,
		Containers:       dc.ContainerList,
		Jobs:             dc.JobList,
		Credentials:      dc.CredentialList,
		Volumes:          dc.VolumeList,
		Secrets:          dc.SecretList,
		Images:           dc.ImageList,
		Projects:         dc.ProjectList,
		VolumeClasses:    dc.VolumeClassList,
		AppItem:          dc.AppItem,
		ContainerItem:    dc.ContainerItem,
		JobItem:          dc.JobItem,
		CredentialItem:   dc.CredentialItem,
		VolumeItem:       dc.VolumeItem,
		SecretItem:       dc.SecretItem,
		ImageItem:        dc.ImageItem,
		ProjectItem:      dc.ProjectItem,
		VolumeClassItem:  dc.VolumeClassItem,
		ComputeClasses:   dc.ComputeClassList,
		ComputeClassItem: dc.ComputeClassItem,
		Regions:          dc.RegionList,
		RegionItem:       dc.RegionItem,
		Events:           dc.EventList,
		EventItem:        dc.EventItem,
	}, nil
}

func (dc *MockClientFactory) CreateWithAllProjects() (client.Client, error) {
	return dc.CreateDefault()
}

type MockClient struct {
	Apps             []apiv1.App
	AppItem          *apiv1.App
	Containers       []apiv1.ContainerReplica
	ContainerItem    *apiv1.ContainerReplica
	Jobs             []apiv1.Job
	JobItem          *apiv1.Job
	Credentials      []apiv1.Credential
	CredentialItem   *apiv1.Credential
	Volumes          []apiv1.Volume
	VolumeItem       *apiv1.Volume
	Secrets          []apiv1.Secret
	SecretItem       *apiv1.Secret
	Images           []apiv1.Image
	ImageItem        *apiv1.Image
	Projects         []apiv1.Project
	ProjectItem      *apiv1.Project
	VolumeClasses    []apiv1.VolumeClass
	VolumeClassItem  *apiv1.VolumeClass
	ComputeClasses   []apiv1.ComputeClass
	ComputeClassItem *apiv1.ComputeClass
	Regions          []apiv1.Region
	RegionItem       *apiv1.Region
	Events           []apiv1.Event
	EventItem        *apiv1.Event
}

func (m *MockClient) KubeConfig(ctx context.Context, opts *client.KubeProxyAddressOptions) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockClient) KubeProxyAddress(ctx context.Context, opts *client.KubeProxyAddressOptions) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockClient) DevSessionRenew(ctx context.Context, name string, client v1.DevSessionInstanceClient) error {
	//TODO implement me
	panic("implement me")
}

func (m *MockClient) DevSessionRelease(ctx context.Context, name string) error {
	//TODO implement me
	panic("implement me")
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
	if m.AppItem != nil {
		return m.AppItem, nil
	}
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

func (m *MockClient) AppIgnoreDeleteCleanup(ctx context.Context, name string) error {
	return nil
}

func (m *MockClient) AppGet(ctx context.Context, name string) (*apiv1.App, error) {
	if m.AppItem != nil {
		return m.AppItem, nil
	}
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
	if m.AppItem != nil {
		return m.AppItem, nil
	}
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
	if m.AppItem != nil {
		return m.AppItem, nil
	}
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
	if m.Credentials != nil {
		return m.Credentials, nil
	}
	return []apiv1.Credential{{
		TypeMeta:      metav1.TypeMeta{},
		ObjectMeta:    metav1.ObjectMeta{Name: "test-cred"},
		ServerAddress: "test-server-address",
		Username:      "",
		Password:      nil,
	}}, nil

}

func (m *MockClient) CredentialGet(ctx context.Context, serverAddress string) (*apiv1.Credential, error) {
	if m.CredentialItem != nil {
		return m.CredentialItem, nil
	}
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
	if m.CredentialItem != nil {
		return m.CredentialItem, nil
	}
	return nil, nil
}

func (m *MockClient) CredentialDelete(ctx context.Context, serverAddress string) (*apiv1.Credential, error) {
	if m.CredentialItem != nil {
		return m.CredentialItem, nil
	}
	return nil, nil
}

func (m *MockClient) SecretCreate(ctx context.Context, name, secretType string, data map[string][]byte) (*apiv1.Secret, error) {
	if m.SecretItem != nil {
		return m.SecretItem, nil
	}
	return nil, nil
}

func (m *MockClient) SecretList(ctx context.Context) ([]apiv1.Secret, error) {
	if m.Secrets != nil {
		return m.Secrets, nil
	}
	return []apiv1.Secret{{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{Name: "found.secret"},
		Type:       "",
		Data:       nil,
		Keys:       nil,
	}}, nil
}

func (m *MockClient) SecretGet(ctx context.Context, name string) (*apiv1.Secret, error) {
	if m.SecretItem != nil {
		return m.SecretItem, nil
	}
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
	if m.SecretItem != nil {
		return m.SecretItem, nil
	}
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
	case "secret.withdata":
		return &apiv1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "secret.withdata",
			},
			Data: map[string][]byte{
				"foo": []byte("bar"),
				"baz": []byte("qux"),
			},
		}, nil
	case "secret.withdata2":
		return &apiv1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "secret.withdata2",
			},
			Data: map[string][]byte{
				"spam": []byte("eggs"),
			},
		}, nil

	}
	return nil, nil
}

func (m *MockClient) SecretUpdate(ctx context.Context, name string, data map[string][]byte) (*apiv1.Secret, error) {
	if m.SecretItem != nil {
		return m.SecretItem, nil
	}
	return nil, nil
}

func (m *MockClient) SecretDelete(ctx context.Context, name string) (*apiv1.Secret, error) {
	if m.SecretItem != nil {
		return m.SecretItem, nil
	}
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
	if m.ContainerItem != nil {
		return m.ContainerItem, nil
	}
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
	if m.ContainerItem != nil {
		return m.ContainerItem, nil
	}
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

func (m *MockClient) ContainerReplicaPortForward(ctx context.Context, name string, port int) (client.PortForwardDialer, error) {
	return nil, nil
}

func (m *MockClient) JobList(ctx context.Context, opts *client.JobListOptions) ([]apiv1.Job, error) {
	if m.Jobs != nil {
		if opts == nil {
			return m.Jobs, nil
		}
		// Do the filtering to make testing simpler
		result := make([]apiv1.Job, 0, len(m.Jobs))
		for _, c := range m.Jobs {
			if c.Spec.AppName == opts.App {
				result = append(result, c)
			}
		}
		return result, nil
	}
	return []apiv1.Job{{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{Name: "found.job"},
		Spec:       apiv1.JobSpec{AppName: "found"},
		Status:     v1.JobStatus{},
	}}, nil
}

func (m *MockClient) JobGet(ctx context.Context, name string) (*apiv1.Job, error) {
	if m.JobItem != nil {
		return m.JobItem, nil
	}
	switch name {
	case "dne":
		return nil, fmt.Errorf("error: job %s does not exist", name)
	case "found", "found.job":
		return &apiv1.Job{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{Name: "found.job"},
			Spec:       apiv1.JobSpec{AppName: "found"},
			Status:     v1.JobStatus{},
		}, nil
	}
	return nil, nil
}

func (m *MockClient) JobRestart(ctx context.Context, name string) error {
	return nil
}

func (m *MockClient) VolumeList(ctx context.Context) ([]apiv1.Volume, error) {
	if m.Volumes != nil {
		return m.Volumes, nil
	}
	return []apiv1.Volume{{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{Name: "found.vol",
			Labels: map[string]string{
				labels.AcornVolumeName: "vol",
				labels.AcornAppName:    "found",
			}},
		Spec: apiv1.VolumeSpec{},
		Status: apiv1.VolumeStatus{
			AppPublicName: "found",
			AppName:       "found",
			VolumeName:    "vol",
		},
	}}, nil
}

func (m *MockClient) VolumeGet(ctx context.Context, name string) (*apiv1.Volume, error) {
	if m.VolumeItem != nil {
		return m.VolumeItem, nil
	}
	potentialVol := apiv1.Volume{TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{Name: "found.vol",
			Labels: map[string]string{
				labels.AcornVolumeName: "vol",
				labels.AcornAppName:    "found",
			}},
		Spec: apiv1.VolumeSpec{},
		Status: apiv1.VolumeStatus{
			AppName:       "found",
			AppPublicName: "found",
			VolumeName:    "vol",
		},
	}

	switch name {
	case "dne":
		return nil, fmt.Errorf("error: volume %s does not exist", name)
	case "volume":
		return &potentialVol, nil
	case "found.vol":
		return &potentialVol, nil
	}
	return nil, nil
}

func (m *MockClient) VolumeDelete(ctx context.Context, name string) (*apiv1.Volume, error) {
	if m.VolumeItem != nil {
		return m.VolumeItem, nil
	}
	switch name {
	case "dne":
		return nil, nil
	case "volume":
		return &apiv1.Volume{}, nil
	case "found.vol":
		return &apiv1.Volume{}, nil
	}
	return nil, nil
}

func (m *MockClient) ImageList(ctx context.Context) ([]apiv1.Image, error) {
	if m.Images != nil {
		return m.Images, nil
	}
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
	if m.ImageItem != nil {
		return m.ImageItem, nil
	}
	return &apiv1.Image{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{Name: "found-image1234567"},
		Tags:       []string{"testtag:latest"},
		Digest:     "1234567890asdfghkl",
	}, nil
}

func (m *MockClient) ImageDelete(ctx context.Context, name string, opts *client.ImageDeleteOptions) (*apiv1.Image, []string, error) {
	if m.ImageItem != nil {
		return m.ImageItem, m.ImageItem.Tags, nil
	}

	img := &apiv1.Image{TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{Name: "ff12345", DeletionTimestamp: typed.Pointer(metav1.Now())},
		Tags:       []string{"testtag1:latest", "testtag2:latest", "foo:v1", "foo:v2"},
	}

	switch name {
	case "ff12345":
		if !opts.Force {
			return nil, nil, fmt.Errorf("unable to delete %s (must be forced) - image is referenced in multiple repositories", name)
		} else {

			return img, img.Tags, nil
		}
	case "foo:v1": // just remove a tag
		return nil, []string{"foo:v1"}, nil
	}
	return nil, nil, apierrors.NewNotFound(schema.GroupResource{Group: "acorn", Resource: "images"}, name)
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

func (m *MockClient) ImageCopy(_ context.Context, srcImage, _ string, _ *client.ImageCopyOptions) (<-chan client.ImageProgress, error) {
	switch srcImage {
	case "found":
		progresses := make(chan client.ImageProgress)
		close(progresses)
		return progresses, nil
	case "dne":
		progresses := make(chan client.ImageProgress)
		close(progresses)
		return progresses, fmt.Errorf("error: tag %s does not exist", srcImage)
	default:
		progresses := make(chan client.ImageProgress)
		close(progresses)
		return progresses, fmt.Errorf("error: tag %s does not exist", srcImage)
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

func (m *MockClient) ImageSign(ctx context.Context, image string, payload []byte, signatureB64 string, opts *client.ImageSignOptions) (*apiv1.ImageSignature, error) {
	return &apiv1.ImageSignature{
		TypeMeta:        metav1.TypeMeta{},
		ObjectMeta:      metav1.ObjectMeta{Name: "found-image1234567"},
		SignatureDigest: "1234abcd",
	}, nil
}

func (m *MockClient) ImageVerify(ctx context.Context, image string, opts *client.ImageVerifyOptions) (*apiv1.ImageSignature, error) {
	return nil, nil
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

func (m *MockClient) Info(ctx context.Context) ([]apiv1.Info, error) {
	return []apiv1.Info{
		{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{},
		},
	}, nil
}

func (m *MockClient) GetNamespace() string { return "" }

func (m *MockClient) GetClient() (kclient.WithWatch, error) { return nil, nil }

func (m *MockClient) PromptUser(s string) error {
	return nil
}

func (m *MockClient) AppConfirmUpgrade(ctx context.Context, name string) error {
	return nil
}

func (m *MockClient) AcornImageBuildGet(ctx context.Context, name string) (*apiv1.AcornImageBuild, error) {
	// TODO implement me
	panic("implement me")
}

func (m *MockClient) AcornImageBuildList(ctx context.Context) ([]apiv1.AcornImageBuild, error) {
	// TODO implement me
	panic("implement me")
}

func (m *MockClient) AcornImageBuildDelete(ctx context.Context, name string) (*apiv1.AcornImageBuild, error) {
	// TODO implement me
	panic("implement me")
}

func (m *MockClient) AcornImageBuild(ctx context.Context, file string, opts *client.AcornImageBuildOptions) (*v1.AppImage, error) {
	// TODO implement me
	panic("implement me")
}

func (m *MockClient) ProjectList(ctx context.Context) ([]apiv1.Project, error) {
	if m.Projects != nil {
		return m.Projects, nil
	}
	return []apiv1.Project{{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{Name: "project"},
	}}, nil
}

func (m *MockClient) ComputeClassList(_ context.Context) ([]apiv1.ComputeClass, error) {
	return m.ComputeClasses, nil
}

func (m *MockClient) ComputeClassGet(_ context.Context, name string) (*apiv1.ComputeClass, error) {
	if m.ComputeClassItem != nil {
		return m.ComputeClassItem, nil
	}

	for _, s := range m.ComputeClasses {
		if s.Name == name {
			return &s, nil
		}
	}

	return nil, apierrors.NewNotFound(schema.GroupResource{
		Group:    "api.acorn.io",
		Resource: "computeclasses",
	}, name)
}

func (m *MockClient) GetProject() string {
	if m.ProjectItem != nil {
		return m.ProjectItem.Name
	}
	return ""
}

func (m *MockClient) ProjectGet(ctx context.Context, name string) (*apiv1.Project, error) {
	if m.ProjectItem != nil {
		return m.ProjectItem, nil
	}
	return nil, nil
}

func (m *MockClient) ProjectCreate(ctx context.Context, name string, region string, supportedRegions []string) (*apiv1.Project, error) {
	// TODO implement me
	panic("implement me")
}

func (m *MockClient) ProjectDelete(ctx context.Context, name string) (*apiv1.Project, error) {
	// TODO implement me
	panic("implement me")
}

func (m *MockClient) ProjectUpdate(ctx context.Context, project *apiv1.Project, defaultRegion string, supportedRegions []string) (*apiv1.Project, error) {
	// TODO implement me
	panic("implement me")
}

func (m *MockClient) VolumeClassList(context.Context) ([]apiv1.VolumeClass, error) {
	return m.VolumeClasses, nil
}

func (m *MockClient) VolumeClassGet(_ context.Context, name string) (*apiv1.VolumeClass, error) {
	if m.VolumeClassItem != nil {
		return m.VolumeClassItem, nil
	}

	for _, s := range m.VolumeClasses {
		if s.Name == name {
			return &s, nil
		}
	}

	return nil, apierrors.NewNotFound(schema.GroupResource{
		Group:    "api.acorn.io",
		Resource: "volumeclasses",
	}, name)
}

func (m *MockClient) RegionList(context.Context) ([]apiv1.Region, error) {
	return m.Regions, nil
}

func (m *MockClient) RegionGet(_ context.Context, name string) (*apiv1.Region, error) {
	if m.RegionItem != nil {
		return m.RegionItem, nil
	}

	for _, s := range m.Regions {
		if s.Name == name {
			return &s, nil
		}
	}

	return nil, apierrors.NewNotFound(schema.GroupResource{
		Group:    "api.acorn.io",
		Resource: "regions",
	}, name)
}

func (m *MockClient) EventStream(context.Context, *client.EventStreamOptions) (<-chan apiv1.Event, error) {
	// TODO: Implement me
	return nil, nil
}
