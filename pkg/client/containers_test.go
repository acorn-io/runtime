package client_test

import (
	"context"
	"fmt"
	"testing"

	v12 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/acorn-io/acorn/pkg/labels"
	scheme2 "github.com/acorn-io/acorn/pkg/scheme"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testcontrollerclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func createMockedDefaultClientContainerLister(t *testing.T, projectName string, containerName string) (client.DefaultClient, v12.ContainerReplicaList, error) {
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

	containerListObj := v12.ContainerReplicaList{
		Items: []v12.ContainerReplica{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      containerName,
					Namespace: projectName,
				},
			},
		},
	}
	testingScheme := scheme2.Scheme

	testK8ClientBuilder := testcontrollerclient.NewClientBuilder()
	testK8ClientBuilder = testK8ClientBuilder.WithScheme(testingScheme)
	testK8ClientBuilder = testK8ClientBuilder.WithObjects(&ns)
	testK8ClientBuilder = testK8ClientBuilder.WithLists(&containerListObj)
	testK8Client := testK8ClientBuilder.Build()
	defaultClient := client.DefaultClient{
		Project:   projectName,
		Namespace: projectName,
		Client:    testK8Client,
	}
	return defaultClient, containerListObj, nil
}

func TestMultiClientContainerCrossProject(t *testing.T) {
	ctx := context.Background()
	// Make two k8 clients and two default clients

	defaultClient1, containerListObj1, err := createMockedDefaultClientContainerLister(t, "test1", "app1")
	assert.NoError(t, err)

	defaultClient2, containerListObj2, err := createMockedDefaultClientContainerLister(t, "test2", "app2")
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

	// Test cross project listing
	listResp, err := mMultiClient.ContainerReplicaList(ctx, &client.ContainerReplicaListOptions{})

	assert.NoError(t, err, "issue calling multi-client AppDelete")
	assert.Len(t, listResp, 2)

	// Test cross project containerGet
	getResp, err := mMultiClient.ContainerReplicaGet(ctx, "test2::"+containerListObj2.Items[0].Name)

	assert.NoError(t, err, "issue calling multi-client cross-project ContainerReplicaGet")

	wantOut := containerListObj2.Items[0]
	wantOut.Name = "test2::" + wantOut.Name

	assert.Equal(t, getResp.Name, wantOut.Name)
	assert.Equal(t, getResp.Namespace, wantOut.Namespace)

	// Test cross project containerGet without correct syntax
	_, err = mMultiClient.ContainerReplicaGet(ctx, "test2:"+containerListObj2.Items[0].Name)

	assert.Error(t, err)

	// Test same project containerGet
	getResp, err = mMultiClient.ContainerReplicaGet(ctx, containerListObj1.Items[0].Name)

	assert.NoError(t, err, "issue calling multi-client ContainerReplicaGet")
	wantOut = containerListObj1.Items[0]
	assert.Equal(t, getResp.Name, wantOut.Name)
	assert.Equal(t, getResp.Namespace, wantOut.Namespace)
}
