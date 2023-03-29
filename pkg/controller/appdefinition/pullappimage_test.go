package appdefinition

import (
	"testing"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/stretchr/testify/assert"
)

func TestDetermineDesiredImage(t *testing.T) {
	//// Auto-upgrade cases

	// spec.Image is a pattern, status.image.ID not set, availableAppImage set. Expect the desiredImage to match availableAppImage
	testTargetImage(t, app("acorn.io/img:#", "", "acorn.io/img:1", "", false, false), "acorn.io/img:1", "")

	// status image and availableAppImage set. This is an auto-update re-pull. Expect the desiredImage to match availabeAppImage
	testTargetImage(t, app("acorn.io/img:1", "acorn.io/img:1", "acorn.io/img:1", "", true, false), "acorn.io/img:1", "")

	// spec.Image is a pattern, confirmUpgradeAppImage set, status Image.ID not set, notifyUpgrade true. Expect desiredImage to match confirmUpgradeAppImage
	testTargetImage(t, app("acorn.io/img:#", "", "", "acorn.io/img:1", false, true), "acorn.io/img:1", "")

	// spec.Image is a pattern, confirmUpgradeAppImage set, status Image.ID set, notifyUpgrade true. Expect desiredImage to be blank and condition reason to be set
	testTargetImage(t, app("acorn.io/img:#", "acorn.io/img:1", "", "acorn.io/img:2", false, true), "", "confirm upgrade to acorn.io/img:2")

	// spec.Image is a pattern, nothing else set. Expect desiredImage to be blank and condition reason to be set
	testTargetImage(t, app("acorn.io/img:#", "", "", "", false, false), "", "waiting for image to satisfy auto-upgrade tag #")

	// spec.Image is a pattern, status.image.ID is set. Expect desiredImage and conditionReason to be blank
	testTargetImage(t, app("acorn.io/img:#", "acorn.io/image:1", "", "", false, false), "", "")

	// spec.Image not a pattern, but autoUpgrade == true, spec.Image set, status.Image.ID matches it. Expect desiredImage and reason to be blank
	testTargetImage(t, app("acorn.io/img:1", "acorn.io/img:1", "", "", true, false), "", "")

	// spec.Image not a pattern, but autoUpgrade == true, spec.Image set, but status.Image.ID doesn't matches it. Expect desiredImage to match spec.Image and reason to be blank
	testTargetImage(t, app("acorn.io/img:2", "acorn.io/img:1", "", "", true, false), "acorn.io/img:2", "")

	//// Non-auto-upgrade cases

	// spec.Image and status.image.ID match. Expect a blank desiredImage
	testTargetImage(t, app("acorn.io/img:1", "acorn.io/img:1", "", "", false, false), "", "")

	// spec.Image and status.image.ID don't match. Expect desiredImage to match spec.image
	testTargetImage(t, app("acorn.io/img:2", "acorn.io/img:1", "", "", false, false), "acorn.io/img:2", "")

	// status.image.ID not set. Expect the desiredImage to match spec and condition to remain unset
	testTargetImage(t, app("acorn.io/img:1", "", "", "", false, false), "acorn.io/img:1", "")
}

func testTargetImage(t *testing.T, appInstance *v1.AppInstance, expectedTagetImage string, expectedUnknownReason string) {
	t.Helper()
	actualTargetImage, actualUnknownReason := determineTargetImage(appInstance)
	assert.Equal(t, expectedTagetImage, actualTargetImage)
	assert.Equal(t, expectedUnknownReason, actualUnknownReason)
}

func app(specImage, statusImageID, statusAvailableImage, statusConfirmUpgradeImage string, autoUpgrade, notifyUpgrade bool) *v1.AppInstance {
	return &v1.AppInstance{
		Spec: v1.AppInstanceSpec{
			Image:         specImage,
			AutoUpgrade:   &autoUpgrade,
			NotifyUpgrade: &notifyUpgrade,
		},
		Status: v1.AppInstanceStatus{
			AppImage: v1.AppImage{
				ID:   statusImageID,
				Name: statusImageID,
			},
			AvailableAppImage:      statusAvailableImage,
			ConfirmUpgradeAppImage: statusConfirmUpgradeImage,
		},
	}
}
