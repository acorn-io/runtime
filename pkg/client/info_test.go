package client_test

// TODO: Add failure states to be tested

import (
	"context"
	"testing"

	v12 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
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

func createMockedDefaultClientInfoLister(t *testing.T, projectName string, namespace string) (client.DefaultClient, v12.InfoList, error) {
	t.Helper()
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: projectName,
			Labels: map[string]string{
				"test.acorn.io/namespace": "true",
				labels.AcornProject:       "true",
			},
		},
		Spec:   corev1.NamespaceSpec{},
		Status: corev1.NamespaceStatus{},
	}

	infoListObj := v12.InfoList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "",
			APIVersion: "",
		},
		ListMeta: metav1.ListMeta{},
		Items: []v12.Info{
			{
				TypeMeta: metav1.TypeMeta{
					Kind:       "",
					APIVersion: "",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      projectName,
					Namespace: namespace,
				},
				Spec: v12.InfoSpec{},
			},
		},
	}
	testingScheme := scheme2.Scheme
	err := scheme2.AddToScheme(testingScheme)
	if err != nil {
		return client.DefaultClient{}, v12.InfoList{}, err
	}

	testK8ClientBuilder := testcontrollerclient.NewClientBuilder()
	testK8ClientBuilder.WithScheme(testingScheme)
	testK8ClientBuilder.WithObjects(&ns)
	testK8ClientBuilder.WithLists(&infoListObj)
	testK8Client := testK8ClientBuilder.Build()
	defaultClient := client.DefaultClient{
		Project:    projectName,
		Namespace:  namespace,
		Client:     testK8Client,
		RESTConfig: nil,
		RESTClient: nil,
		Dialer:     nil,
	}
	return defaultClient, infoListObj, nil
}

func TestDefaultClientInfo(t *testing.T) {
	ctx := context.Background()

	defaultClient, infoListObj, err := createMockedDefaultClientInfoLister(t, "test1", "test1")
	assert.NoError(t, err)

	infoResp, err := defaultClient.Info(ctx)

	assert.NoError(t, err, "issue calling default-client info")

	assert.ElementsMatch(t, infoResp, infoListObj.Items)
}

func TestMultiClientInfo(t *testing.T) {
	ctx := context.Background()
	// Make two k8 clients and two default clients

	defaultClient1, infoListObj1, err := createMockedDefaultClientInfoLister(t, "test1", "test1")
	assert.NoError(t, err)

	defaultClient2, infoListObj2, err := createMockedDefaultClientInfoLister(t, "test2", "test2")
	assert.NoError(t, err)

	// create factory that can list projects:
	ctrl := gomock.NewController(t)
	mFactory := mocks.NewMockProjectClientFactory(ctrl)
	mFactory.EXPECT().List(gomock.Any()).Return([]client.Client{&defaultClient1, &defaultClient2}, nil)
	projectMap := make(map[string]*client.DefaultClient)
	projectMap["test1"] = &defaultClient1
	projectMap["test2"] = &defaultClient2

	mMultiClient := client.NewMultiClient("test1", "test1", mFactory)
	infoResp, err := mMultiClient.Info(ctx)

	assert.NoError(t, err, "issue calling multi-client info")

	var expectedResponse = []v12.Info{infoListObj1.Items[0], infoListObj2.Items[0]}

	assert.ElementsMatch(t, infoResp, expectedResponse)
}

func TestMultiClientFQDNClobberingInfo(t *testing.T) {
	ctx := context.Background()
	// Make two k8 clients and two default clients

	defaultClient1, infoListObj1, err := createMockedDefaultClientInfoLister(t, "test1", "test1")
	assert.NoError(t, err)

	defaultClient2, infoListObj2, err := createMockedDefaultClientInfoLister(t, "acorn.io/jacob/test1", "test1")
	assert.NoError(t, err)

	// create factory that can list projects:
	ctrl := gomock.NewController(t)
	mFactory := mocks.NewMockProjectClientFactory(ctrl)
	mFactory.EXPECT().List(gomock.Any()).Return([]client.Client{&defaultClient1, &defaultClient2}, nil)
	projectMap := make(map[string]*client.DefaultClient)
	projectMap["test1"] = &defaultClient1
	projectMap["acorn.io/jacob/test1"] = &defaultClient2

	mMultiClient := client.NewMultiClient("test1", "test1", mFactory)
	infoResp, err := mMultiClient.Info(ctx)

	assert.NoError(t, err, "issue calling multi-client info")

	var expectedResponse = []v12.Info{infoListObj1.Items[0], infoListObj2.Items[0]}

	assert.ElementsMatch(t, infoResp, expectedResponse)
}
