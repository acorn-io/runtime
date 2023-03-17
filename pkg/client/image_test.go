package client_test

import (
	"context"

	v12 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/mocks"
	scheme2 "github.com/acorn-io/acorn/pkg/scheme"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"testing"

	testcontrollerclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type defaultClientImagesConfig struct {
	images      v12.ImageList
	projectName string
	namespace   string
}

func createMockedDefaultClientImageLister(t *testing.T, config defaultClientImagesConfig) (client.DefaultClient, error) {
	t.Helper()
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: config.projectName,
			Labels: map[string]string{
				"test.acorn.io/namespace": "true",
				labels.AcornProject:       "true",
			},
		},
	}

	imageListObj := config.images
	testingScheme := scheme2.Scheme

	testK8ClientBuilder := testcontrollerclient.NewClientBuilder()
	testK8ClientBuilder.WithScheme(testingScheme)
	testK8ClientBuilder.WithObjects(&ns)
	testK8ClientBuilder.WithLists(&imageListObj)
	testK8Client := testK8ClientBuilder.Build()
	defaultClient := client.DefaultClient{
		Project:    config.projectName,
		Namespace:  config.namespace,
		Client:     testK8Client,
		RESTConfig: nil,
		RESTClient: nil,
		Dialer:     nil,
	}
	return defaultClient, nil
}

func TestDefaultClientImageList(t *testing.T) {
	ctx := context.Background()

	mockConfig := defaultClientImagesConfig{
		images: v12.ImageList{
			Items: []v12.Image{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ImageName",
						Namespace: "NS",
					},
					Repo:   "Repo1",
					Digest: "Digest1",
					Tags:   nil,
				},
			},
		},
		projectName: "projName",
		namespace:   "NS",
	}
	defaultClient, err := createMockedDefaultClientImageLister(t, mockConfig)
	assert.NoError(t, err)

	infoResp, err := defaultClient.ImageList(ctx)

	assert.NoError(t, err, "issue calling default-client info")

	assert.ElementsMatch(t, infoResp, mockConfig.images.Items)
}

func TestMultiClientImageListSingle(t *testing.T) {
	ctx := context.Background()
	// Make two k8 clients and two default clients

	mockConfig1 := defaultClientImagesConfig{
		images: v12.ImageList{
			Items: []v12.Image{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ImageName1",
						Namespace: "NS1",
						Labels: map[string]string{
							labels.AcornProject: "NS1",
						},
					},
					Repo:   "Repo1",
					Digest: "Digest1",
					Tags:   nil,
				},
			},
		},
		projectName: "NS1",
		namespace:   "NS1",
	}
	defaultClient1, err := createMockedDefaultClientImageLister(t, mockConfig1)
	assert.NoError(t, err)

	// create factory that can list projects:
	ctrl := gomock.NewController(t)
	mFactory := mocks.NewMockProjectClientFactory(ctrl)
	// Lists default clients to use
	mFactory.EXPECT().List(gomock.Any()).Return([]client.Client{&defaultClient1}, nil)
	// gets default project
	mFactory.EXPECT().DefaultProject().Return("NS1").AnyTimes()
	projectMap := make(map[string]*client.DefaultClient)
	projectMap["projName1"] = &defaultClient1

	mFactory.EXPECT().ForProject(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, projectName string) (client.Client, error) {
		return projectMap[projectName], nil
	}).AnyTimes()

	mMultiClient := client.NewMultiClient("NS1", "NS1", mFactory)
	imageListResp, err := mMultiClient.ImageList(ctx)

	assert.NoError(t, err, "issue calling multi-client info")

	var expectedResponse = []v12.Image{mockConfig1.images.Items[0]}

	assert.EqualValues(t, expectedResponse, imageListResp)
}

func TestMultiClientImageListMuliple(t *testing.T) {
	ctx := context.Background()
	// Make two k8 clients and two default clients

	mockConfig1 := defaultClientImagesConfig{
		images: v12.ImageList{
			Items: []v12.Image{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ImageName1",
						Namespace: "NS1",
						Labels: map[string]string{
							labels.AcornProject: "NS1",
						},
					},
					Repo:   "Repo1",
					Digest: "Digest1",
					Tags:   nil,
				},
			},
		},
		projectName: "NS1",
		namespace:   "NS1",
	}
	defaultClient1, err := createMockedDefaultClientImageLister(t, mockConfig1)
	assert.NoError(t, err)

	mockConfig2 := defaultClientImagesConfig{
		images: v12.ImageList{
			Items: []v12.Image{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ImageName2",
						Namespace: "NS2",
						Labels: map[string]string{
							labels.AcornProject: "NS2",
						},
					},
					Repo:   "Repo2",
					Digest: "Digest2",
					Tags:   nil,
				},
			},
		},
		projectName: "NS2",
		namespace:   "NS2",
	}
	defaultClient2, err := createMockedDefaultClientImageLister(t, mockConfig2)
	assert.NoError(t, err)

	// create factory that can list projects:
	ctrl := gomock.NewController(t)
	mFactory := mocks.NewMockProjectClientFactory(ctrl)
	// Lists default clients to use
	mFactory.EXPECT().List(gomock.Any()).Return([]client.Client{&defaultClient1, &defaultClient2}, nil).AnyTimes()
	// gets default project
	mFactory.EXPECT().DefaultProject().Return("projName1").AnyTimes()
	projectMap := make(map[string]*client.DefaultClient)
	projectMap["NS1"] = &defaultClient1
	projectMap["NS2"] = &defaultClient2

	mFactory.EXPECT().ForProject(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, projectName string) (client.Client, error) {
		return projectMap[projectName], nil
	}).AnyTimes()

	mMultiClient := client.NewMultiClient("NS1", "NS2", mFactory)
	imageListResp, err := mMultiClient.ImageList(ctx)

	assert.NoError(t, err, "issue calling multi-client info")

	var expectedResponse = []v12.Image{mockConfig1.images.Items[0], mockConfig2.images.Items[0]}

	assert.EqualValues(t, expectedResponse, imageListResp)
	assert.ElementsMatch(t, expectedResponse, imageListResp)

}

func TestMultiClientImageListMultipleFQDNClobber(t *testing.T) {
	ctx := context.Background()
	// Make two k8 clients and two default clients

	mockConfig1 := defaultClientImagesConfig{
		images: v12.ImageList{
			Items: []v12.Image{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ImageName1",
						Namespace: "projName1",
						Labels: map[string]string{
							labels.AcornProject: "projName1",
						},
					},
					Repo:   "Repo1",
					Digest: "Digest1",
					Tags:   nil,
				},
			},
		},
		projectName: "projName1",
		namespace:   "projName1",
	}
	defaultClient1, err := createMockedDefaultClientImageLister(t, mockConfig1)
	assert.NoError(t, err)

	mockConfig2 := defaultClientImagesConfig{
		images: v12.ImageList{
			Items: []v12.Image{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ImageName2",
						Namespace: "test2",
						Labels: map[string]string{
							labels.AcornProject: "acorn.io/jacob/test2",
						},
					},
					Repo:   "Repo2",
					Digest: "Digest2",
					Tags:   nil,
				},
			},
		},
		projectName: "acorn.io/jacob/test2",
		namespace:   "test2",
	}
	defaultClient2, err := createMockedDefaultClientImageLister(t, mockConfig2)
	assert.NoError(t, err)

	// create factory that can list projects:
	ctrl := gomock.NewController(t)
	mFactory := mocks.NewMockProjectClientFactory(ctrl)
	// Lists default clients to use
	mFactory.EXPECT().List(gomock.Any()).Return([]client.Client{&defaultClient1, &defaultClient2}, nil).AnyTimes()
	// gets default project
	mFactory.EXPECT().DefaultProject().Return("projName1").AnyTimes()
	projectMap := make(map[string]*client.DefaultClient)
	projectMap["projName1"] = &defaultClient1
	projectMap["acorn.io/jacob/test2"] = &defaultClient2

	mFactory.EXPECT().ForProject(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, projectName string) (client.Client, error) {
		return projectMap[projectName], nil
	}).AnyTimes()

	// Test with default project as projectName1
	mMultiClient := client.NewMultiClient("projName1", "projName1", mFactory)
	imageListResp, err := mMultiClient.ImageList(ctx)
	assert.NoError(t, err, "issue calling multi-client info")
	var expectedResponse = []v12.Image{mockConfig1.images.Items[0], mockConfig2.images.Items[0]}
	assert.EqualValues(t, expectedResponse, imageListResp)

	// Test with default project as acorn.io/jacob/test2
	mMultiClient = client.NewMultiClient("projName2", "projName2", mFactory)
	imageListResp, err = mMultiClient.ImageList(ctx)
	assert.NoError(t, err, "issue calling multi-client info")
	expectedResponse = []v12.Image{mockConfig1.images.Items[0], mockConfig2.images.Items[0]}
	assert.EqualValues(t, expectedResponse, imageListResp)
	assert.ElementsMatch(t, expectedResponse, imageListResp)

}
