package appdefinition

import (
	"context"
	"net/http"
	"strings"
	"testing"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestDetermineDesiredImage(t *testing.T) {
	//// Auto-upgrade cases

	// spec.Image is a pattern, status.image.ID not set, availableAppImage set. Expect the desiredImage to match availableAppImage
	testTargetImage(t, app("acorn", "acorn.io/img:#", "", "", "acorn.io/img:1", "", false, false), "acorn.io/img:1", "")

	// status image and availableAppImage set. This is an auto-update re-pull. Expect the desiredImage to match availabeAppImage
	testTargetImage(t, app("acorn", "acorn.io/img:1", "acorn.io/img:1", "", "acorn.io/img:1", "", true, false), "acorn.io/img:1", "")

	// spec.Image is a pattern, confirmUpgradeAppImage set, status Image.ID not set, notifyUpgrade true. Expect desiredImage to match confirmUpgradeAppImage
	testTargetImage(t, app("acorn", "acorn.io/img:#", "", "", "", "acorn.io/img:1", false, true), "acorn.io/img:1", "")

	// spec.Image is a pattern, confirmUpgradeAppImage set, status Image.ID set, notifyUpgrade true. Expect desiredImage to be blank and condition reason to be set
	testTargetImage(t, app("acorn", "acorn.io/img:#", "acorn.io/img:1", "", "", "acorn.io/img:2", false, true), "", "confirm upgrade to acorn.io/img:2")

	// spec.Image is a pattern, nothing else set. Expect desiredImage to be blank and condition reason to be set
	testTargetImage(t, app("acorn", "acorn.io/img:#", "", "", "", "", false, false), "", "waiting for image to satisfy auto-upgrade tag #")

	// spec.Image is a pattern, status.image.ID is set. Expect desiredImage and conditionReason to be blank
	testTargetImage(t, app("acorn", "acorn.io/img:#", "acorn.io/image:1", "", "", "", false, false), "", "")

	// spec.Image not a pattern, but autoUpgrade == true, spec.Image set, status.Image.ID matches it. Expect desiredImage and reason to be blank
	testTargetImage(t, app("acorn", "acorn.io/img:1", "acorn.io/img:1", "", "", "", true, false), "", "")

	// spec.Image not a pattern, but autoUpgrade == true, spec.Image set, but status.Image.ID doesn't matches it. Expect desiredImage to match spec.Image and reason to be blank
	testTargetImage(t, app("acorn", "acorn.io/img:2", "acorn.io/img:1", "", "", "", true, false), "acorn.io/img:2", "")

	//// Non-auto-upgrade cases

	// spec.Image and status.image.ID match. Expect a blank desiredImage
	testTargetImage(t, app("acorn", "acorn.io/img:1", "acorn.io/img:1", "", "", "", false, false), "", "")

	// spec.Image and status.image.ID don't match. Expect desiredImage to match spec.image
	testTargetImage(t, app("acorn", "acorn.io/img:2", "acorn.io/img:1", "", "", "", false, false), "acorn.io/img:2", "")

	// status.image.ID not set. Expect the desiredImage to match spec and condition to remain unset
	testTargetImage(t, app("acorn", "acorn.io/img:1", "", "", "", "", false, false), "acorn.io/img:1", "")
}

func testTargetImage(t *testing.T, appInstance *v1.AppInstance, expectedTagetImage string, expectedUnknownReason string) {
	t.Helper()
	actualTargetImage, actualUnknownReason := determineTargetImage(appInstance)
	assert.Equal(t, expectedTagetImage, actualTargetImage)
	assert.Equal(t, expectedUnknownReason, actualUnknownReason)
}

func app(namespace, specImage, statusImageID, statusImageDigest, statusAvailableImage, statusConfirmUpgradeImage string, autoUpgrade, notifyUpgrade bool) *v1.AppInstance {
	appInstance := &v1.AppInstance{
		Spec: v1.AppInstanceSpec{
			Image:         specImage,
			AutoUpgrade:   &autoUpgrade,
			NotifyUpgrade: &notifyUpgrade,
		},
		Status: v1.AppInstanceStatus{
			AppImage: v1.AppImage{
				ID:     statusImageID,
				Name:   statusImageID,
				Digest: statusImageDigest,
			},
			AvailableAppImage:      statusAvailableImage,
			ConfirmUpgradeAppImage: statusConfirmUpgradeImage,
		},
	}
	appInstance.SetNamespace(namespace)
	return appInstance
}

func returnImage(appInstance v1.AppInstance) *apiv1.Image {
	return &apiv1.Image{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ReplaceAll(appInstance.Status.AppImage.Name, "/", "+"),
			Namespace: appInstance.Namespace,
		},
		Digest: appInstance.Status.AppImage.Digest,
	}
}

type mockImagePuller struct {
	transport          http.RoundTripper
	returnRemoteDigest string
	returnPullAppImg   *v1.AppImage
	imagepulled        bool
}

func (m *mockImagePuller) remoteImageDigest(_ context.Context, _ client.Reader, _, _ string) (string, error) {
	return m.returnRemoteDigest, nil
}
func (m *mockImagePuller) imagePullAppImage(_ context.Context, _ client.Reader, _, _, _ string) (*v1.AppImage, error) {
	m.imagepulled = true
	return m.returnPullAppImg, nil
}

func TestPullAppImage(t *testing.T) {
	testCases := []struct {
		name             string
		appInstance      *v1.AppInstance
		mockImagePuller  *mockImagePuller
		expectedAppImage v1.AppImage
		expectPull       bool
	}{
		{
			name:             "Image pull is prevented if status image digest is found to be the same as the one returned by the remote registry",
			appInstance:      app("acorn", "acorn.io/img:v1.0.0", "acorn.io/img:v1.0.0", "sha256:a450", "acorn.io/img:v1.0.0", "", false, false),
			mockImagePuller:  &mockImagePuller{transport: http.RoundTripper(&http.Transport{}), returnRemoteDigest: "sha256:a450", imagepulled: false},
			expectedAppImage: v1.AppImage{},
			expectPull:       false,
		},
		{
			name:             "Image pull is not prevented if status image digest is different than the one returned by the remote registry",
			appInstance:      app("acorn", "acorn.io/img:v1.0.0", "acorn.io/img:v1.0.0", "", "acorn.io/img:v1.0.0", "", false, false),
			mockImagePuller:  &mockImagePuller{transport: http.RoundTripper(&http.Transport{}), returnRemoteDigest: "sha256:111new", imagepulled: false, returnPullAppImg: &v1.AppImage{ID: "acorn.io/img:v1.0.1", Name: "acorn.io/img:v1.0.1", Digest: "sha256:111new"}},
			expectedAppImage: v1.AppImage{ID: "acorn.io/img:v1.0.1", Name: "acorn.io/img:v1.0.1", Digest: "sha256:111new"},
			expectPull:       true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// create a new router request and response for testing purposes
			req := tester.NewRequest(t, scheme.Scheme, tc.appInstance, returnImage(*tc.appInstance))
			resp := &tester.Response{}

			err := pullAppImage(tc.mockImagePuller)(req, resp)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			assert.Equal(t, tc.expectPull, tc.mockImagePuller.imagepulled)
			if tc.expectPull {
				assert.Equal(t, tc.expectedAppImage.ID, tc.appInstance.Status.AppImage.ID)
			}
		})
	}
}
