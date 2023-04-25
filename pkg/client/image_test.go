package client_test

import (
	"context"
	"fmt"
	"testing"

	v12 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/mocks"
	scheme2 "github.com/acorn-io/acorn/pkg/scheme"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testcontrollerclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func createMockedDefaultClientImageLister(t *testing.T, projectName string, imageName string) (client.DefaultClient, v12.ImageList, error) {
	t.Helper()
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: projectName,
			Labels: map[string]string{
				"test.acorn.io/namespace": "true",
				labels.AcornProject:       "true",
			},
		},
	}

	imageListObj := v12.ImageList{
		Items: []v12.Image{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      imageName,
					Namespace: projectName,
				},
			},
		},
	}

	var imageDetailsObj []v12.ImageDetails
	for i := range imageListObj.Items {
		imageDetailsObj = append(imageDetailsObj, v12.ImageDetails{AppImage: v1.AppImage{
			ID:   "1234",
			Name: imageListObj.Items[i].Name,
		}})
	}

	testingScheme := scheme2.Scheme

	testK8ClientBuilder := testcontrollerclient.NewClientBuilder()
	testK8ClientBuilder = testK8ClientBuilder.WithScheme(testingScheme)
	testK8ClientBuilder = testK8ClientBuilder.WithObjects(&ns)
	for i := range imageDetailsObj {
		testK8ClientBuilder = testK8ClientBuilder.WithObjects(&imageDetailsObj[i])
	}
	testK8ClientBuilder = testK8ClientBuilder.WithLists(&imageListObj)

	testK8Client := testK8ClientBuilder.Build()

	defaultClient := client.DefaultClient{
		Project:    projectName,
		Namespace:  projectName,
		Client:     testK8Client,
		RESTClient: nil,
	}
	return defaultClient, imageListObj, nil
}

func TestMultiClientImagesCrossProject(t *testing.T) {
	ctx := context.Background()
	// Make two k8 clients and two default clients

	defaultClient1, _, err := createMockedDefaultClientImageLister(t, "test1", "image1")
	assert.NoError(t, err)

	defaultClient2, imageListObj2, err := createMockedDefaultClientImageLister(t, "test2", "image2")
	assert.NoError(t, err)

	// create factory that can list projects:
	ctrl := gomock.NewController(t)
	mFactory := mocks.NewMockProjectClientFactory(ctrl)
	mFactory.EXPECT().List(gomock.Any()).Return([]client.Client{&defaultClient1, &defaultClient2}, nil).AnyTimes()
	projectMap := make(map[string]*client.DefaultClient)
	projectMap["test1"] = &defaultClient1
	projectMap["test2"] = &defaultClient2
	mFactory.EXPECT().ForProject(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, project string) (client.Client, error) {
		if projectClient, ok := projectMap[project]; ok {
			return projectClient, nil
		} else {
			return nil, fmt.Errorf("project %v not found", project)
		}
	}).AnyTimes()
	mFactory.EXPECT().DefaultProject().Return("test1").AnyTimes()

	mMultiClient := client.NewMultiClient("test1", "test1", mFactory)
	// current default project is test1, so try to delete app2 inside of project test2

	// Test ImageGEt

	getResp, err := mMultiClient.ImageGet(ctx, "test2::image2")

	assert.NoError(t, err, "issue calling multi-client ImageGet")

	wantOut := imageListObj2.Items[0]

	assert.Equal(t, getResp.Name, wantOut.Name)
	assert.Equal(t, getResp.Namespace, wantOut.Namespace)

	// Test delete

	deleteResp, err := mMultiClient.ImageDelete(ctx, "test2::image2", &client.ImageDeleteOptions{})

	assert.NoError(t, err, "issue calling multi-client ImageDelete")

	assert.Equal(t, deleteResp.Name, wantOut.Name)
	assert.Equal(t, deleteResp.Namespace, wantOut.Namespace)
}
