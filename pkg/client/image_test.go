package client

import (
	"context"
	"errors"
	"testing"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/images"
	"github.com/acorn-io/acorn/pkg/labels"
	scheme2 "github.com/acorn-io/acorn/pkg/scheme"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testcontrollerclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func createMockedDefaultClient(t *testing.T) (*DefaultClient, error) {
	t.Helper()
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "acorn",
			Labels: map[string]string{
				"test.acorn.io/namespace": "true",
				labels.AcornProject:       "true",
			},
		},
		Spec:   corev1.NamespaceSpec{},
		Status: corev1.NamespaceStatus{},
	}

	testingScheme := scheme2.Scheme
	err := scheme2.AddToScheme(testingScheme)
	if err != nil {
		return &DefaultClient{}, err
	}

	imageListObj := apiv1.ImageList{
		Items: []apiv1.Image{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "43d08e329d682de23ec0c1adb8c487430d946954d27e5d7661b93527cc2dfd5e",
					Namespace: "acorn",
				},
				Digest: "sha256:43d08e329d682de23ec0c1adb8c487430d946954d27e5d7661b93527cc2dfd5e",
				Tags: []string{
					"foo/bar:v1",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "43d07b329d682de23ec0c1adb8c487430d946954d27e5d7661b93527cc2dfd5e",
					Namespace: "acorn",
				},
				Digest: "sha256:43d07b329d682de23ec0c1adb8c487430d946954d27e5d7661b93527cc2dfd5e",
				Tags: []string{
					"spam/eggs:latest",
				},
			},
		},
	}

	testK8ClientBuilder := testcontrollerclient.NewClientBuilder()
	testK8ClientBuilder.WithScheme(testingScheme)
	testK8ClientBuilder.WithObjects(&ns)
	testK8ClientBuilder.WithLists(&imageListObj)
	testK8Client := testK8ClientBuilder.Build()
	defaultClient := DefaultClient{
		Project:    "acorn",
		Namespace:  "acorn",
		Client:     testK8Client,
		RESTConfig: nil,
		RESTClient: nil,
		Dialer:     nil,
	}
	return &defaultClient, nil
}

func TestFindImage(t *testing.T) {
	c, err := createMockedDefaultClient(t)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	testcases := []struct {
		name              string
		imageName         string
		expectedImageName string
		expectedTag       string
		expectError       bool
		expectedError     interface{}
	}{
		{
			name:              "find image by full name (ID)",
			imageName:         "43d08e329d682de23ec0c1adb8c487430d946954d27e5d7661b93527cc2dfd5e",
			expectedImageName: "43d08e329d682de23ec0c1adb8c487430d946954d27e5d7661b93527cc2dfd5e",
			expectedTag:       "",
			expectError:       false,
		},
		{
			name:              "find image by partial name (ID)",
			imageName:         "43d08e",
			expectedImageName: "43d08e329d682de23ec0c1adb8c487430d946954d27e5d7661b93527cc2dfd5e",
			expectedTag:       "",
			expectError:       false,
		},
		{
			name:              "fail to find image by partial name (ID) - not unique",
			imageName:         "43d0",
			expectedImageName: "43d08e329d682de23ec0c1adb8c487430d946954d27e5d7661b93527cc2dfd5e",
			expectedTag:       "",
			expectError:       true,
			expectedError:     &images.ErrImageIdentifierNotUnique{},
		},
		{
			name:              "find image by tag",
			imageName:         "foo/bar:v1",
			expectedImageName: "43d08e329d682de23ec0c1adb8c487430d946954d27e5d7661b93527cc2dfd5e",
			expectedTag:       "foo/bar:v1",
			expectError:       false,
		},
		{
			name:              "find image by digest",
			imageName:         "sha256:43d08e329d682de23ec0c1adb8c487430d946954d27e5d7661b93527cc2dfd5e",
			expectedImageName: "43d08e329d682de23ec0c1adb8c487430d946954d27e5d7661b93527cc2dfd5e",
			expectedTag:       "",
			expectError:       false,
		},
		{
			name:              "fail to find image by autoupgrade pattern",
			imageName:         "foo/bar:**",
			expectedImageName: "",
			expectedTag:       "",
			expectError:       true,
			expectedError:     &images.ErrImageNotFound{},
		},
		{
			name:              "fail to find image by tag - doesn't exist",
			imageName:         "foo/bar:v9",
			expectedImageName: "",
			expectedTag:       "",
			expectError:       true,
			expectedError:     &images.ErrImageNotFound{},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			image, tag, err := FindImage(ctx, c, tc.imageName)
			if err != nil && !tc.expectError {
				t.Fatal(err)
			}

			if err == nil && tc.expectError {
				t.Fatalf("expected error, got none")
			}

			if tc.expectError && tc.expectedError != nil {
				if !errors.As(err, tc.expectedError) {
					t.Fatalf("expected error %s, got %s", tc.expectedError, err)
				}
			}

			if image != nil && image.Name != tc.expectedImageName {
				t.Fatalf("expected image name %s, got %s", tc.expectedImageName, image.Name)
			}

			if tag != tc.expectedTag {
				t.Fatalf("expected tag %s, got %s", tc.expectedTag, tag)
			}
		})
	}
}
