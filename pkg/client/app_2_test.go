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

func createMockedDefaultClientAppLister(t *testing.T, projectName string, appName string) (client.DefaultClient, v12.AppList, error) {
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

	appListObj := v12.AppList{
		Items: []v12.App{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      appName,
					Namespace: projectName,
				},
			},
		},
	}
	testingScheme := scheme2.Scheme

	testK8ClientBuilder := testcontrollerclient.NewClientBuilder()
	testK8ClientBuilder = testK8ClientBuilder.WithScheme(testingScheme)
	testK8ClientBuilder = testK8ClientBuilder.WithObjects(&ns)
	testK8ClientBuilder = testK8ClientBuilder.WithLists(&appListObj)
	testK8Client := testK8ClientBuilder.Build()
	defaultClient := client.DefaultClient{
		Project:   projectName,
		Namespace: projectName,
		Client:    testK8Client,
	}
	return defaultClient, appListObj, nil
}

func TestMultiClientAppDeleteCrossProject(t *testing.T) {
	ctx := context.Background()
	// Make two k8 clients and two default clients

	defaultClient1, _, err := createMockedDefaultClientAppLister(t, "test1", "app1")
	assert.NoError(t, err)

	defaultClient2, appListObj2, err := createMockedDefaultClientAppLister(t, "test2", "app2")
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
	mFactory.EXPECT().DefaultProject().Return("test1")

	mMultiClient := client.NewMultiClient("test1", "test1", mFactory)
	// current default project is test1, so try to delete app2 inside of project test2
	deleteResp, err := mMultiClient.AppDelete(ctx, "test2::app2")

	assert.NoError(t, err, "issue calling multi-client AppDelete")

	wantOut := appListObj2.Items[0]
	wantOut.Name = "test2::" + wantOut.Name

	assert.Equal(t, deleteResp.Name, wantOut.Name)
	assert.Equal(t, deleteResp.Namespace, wantOut.Namespace)
}
