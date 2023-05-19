package appdefinition

import (
	"context"
	"fmt"
	"testing"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/event"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
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

func testTargetImage(t *testing.T, appInstance *v1.AppInstance, expectedTargetImage string, expectedUnknownReason string) {
	t.Helper()
	actualTargetImage, actualUnknownReason := determineTargetImage(appInstance)
	assert.Equal(t, expectedTargetImage, actualTargetImage)
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

func withMeta[T kclient.Object](name, uid, resourceVersion string, obj T) T {
	obj.SetName(name)
	obj.SetUID(types.UID(uid))
	obj.SetResourceVersion(resourceVersion)

	return obj
}

func TestPullAppImageEvents(t *testing.T) {

	// Test cases below this comment ensure the handler produces the correct events

	now := metav1.Now()
	// Manual upgrade should record an event
	testRecordPullEvent(t,
		"ImageChange",
		withMeta("foo", "foo-uid", "1", app("acorn.io/img:1", "acorn.io/img:0", "", "", false, false)),
		resolveImageErr(nil),
		pullImageTo(&v1.AppImage{Name: "acorn.io/img:1", Digest: "sha256:abcd", VCS: v1.VCS{Revision: "r", Modified: false}}, nil),
		now,
		&apiv1.Event{
			Type:        AppImagePullSuccessEventType,
			Severity:    v1.EventSeverityInfo,
			Description: "Pulled acorn.io/img:1",
			Source:      v1.EventSource{Kind: "App", Name: "foo", UID: types.UID("foo-uid")},
			Observed:    now,
			Details: mustMapify(t, AppImagePullEventDetails{
				ResourceVersion: "1",
				AutoUpgrade:     false,
				Previous:        ImageSummary{Name: "acorn.io/img:0"},
				Target:          ImageSummary{Name: "acorn.io/img:1", Digest: "sha256:abcd", VCS: v1.VCS{Revision: "r", Modified: false}},
			}),
		},
	)

	// Auto-upgrade should record an event when re-pulling
	testRecordPullEvent(t,
		"AutoUpgrade",
		withMeta("foo", "", "", app("acorn.io/img:1", "acorn.io/img:1", "acorn.io/img:1", "", true, false)),
		resolveImageErr(nil),
		pullImageTo(&v1.AppImage{Name: "acorn.io/img:1", Digest: "sha256:abcd", VCS: v1.VCS{Revision: "r", Modified: false}}, nil),
		now,
		&apiv1.Event{
			Type:        AppImagePullSuccessEventType,
			Severity:    v1.EventSeverityInfo,
			Description: "Pulled acorn.io/img:1",
			Source:      v1.EventSource{Kind: "App", Name: "foo"},
			Observed:    now,
			Details: mustMapify(t, AppImagePullEventDetails{
				AutoUpgrade: true,
				Previous:    ImageSummary{Name: "acorn.io/img:1"},
				Target:      ImageSummary{Name: "acorn.io/img:1", Digest: "sha256:abcd", VCS: v1.VCS{Revision: "r", Modified: false}},
			}),
		},
	)

	testRecordPullEvent(t,
		"Pattern",
		withMeta("foo", "", "", app("acorn.io/img:#", "", "", "acorn.io/img:1", false, true)),
		resolveImageErr(nil),
		pullImageTo(&v1.AppImage{Name: "acorn.io/img:1"}, nil),
		now,
		&apiv1.Event{
			Type:        AppImagePullSuccessEventType,
			Severity:    v1.EventSeverityInfo,
			Description: "Pulled acorn.io/img:1",
			Source:      v1.EventSource{Kind: "App", Name: "foo"},
			Observed:    now,
			Details: mustMapify(t, AppImagePullEventDetails{
				AutoUpgrade: true,
				Target:      ImageSummary{Name: "acorn.io/img:1"},
			}),
		},
	)

	// Errors on image pull records an event
	testRecordPullEvent(t,
		"PullError",
		withMeta("foo", "", "", app("acorn.io/img:1", "", "", "", false, false)),
		resolveImageTo("acorn.io/img:1", false, nil),
		pullImageTo(nil, fmt.Errorf("pull error!")),
		now,
		&apiv1.Event{
			Type:        AppImagePullFailureEventType,
			Severity:    v1.EventSeverityWarn,
			Description: "Failed to pull acorn.io/img:1",
			Source:      v1.EventSource{Kind: "App", Name: "foo"},
			Observed:    now,
			Details: mustMapify(t, AppImagePullEventDetails{
				Target: ImageSummary{Name: "acorn.io/img:1"},
				Err:    "pull error!",
			}),
		},
	)

	// Test cases below this comment should produce no events

	// No image change records NO events
	testRecordPullEvent(t,
		"NoImageChange",
		withMeta("foo", "", "", app("acorn.io/img:1", "acorn.io/img:1", "", "", false, false)),
		nil,
		nil,
		now,
		nil,
	)

	// Error during local image resolution records NO events
	testRecordPullEvent(t,
		"ResolutionError",
		withMeta("foo", "", "", app("acorn.io/img:1", "", "", "", false, false)),
		resolveImageTo("", false, fmt.Errorf("resolution error!")),
		nil,
		now,
		nil,
	)
	// testTargetImage(t, app("acorn.io/img:#", "acorn.io/img:1", "", "acorn.io/img:2", false, true), "", "confirm upgrade to acorn.io/img:2")

	testRecordPullEvent(t,
		"Pattern/2",
		withMeta("foo", "", "", app("acorn.io/img:#", "acorn.io/img:1", "", "acorn.io/img:2", false, true)),
		resolveImageErr(nil),
		pullImageTo(&v1.AppImage{Name: "acorn.io/img:2"}, nil),
		now,
		nil,
	)
}

func mustMapify(t *testing.T, obj any) v1.GenericMap {
	t.Helper()
	m, err := v1.Mapify(obj)
	require.NoError(t, err)
	return m
}

func resolveImageTo(resolved string, isLocal bool, err error) resolveImageFunc {
	return func(context.Context, kclient.Client, string, string) (string, bool, error) {
		return resolved, isLocal, err
	}
}

func resolveImageErr(err error) resolveImageFunc {
	return func(context.Context, kclient.Client, string, string) (string, bool, error) {
		return "", false, err
	}
}

func pullImageTo(image *v1.AppImage, err error) pullImageFunc {
	return func(context.Context, kclient.Reader, string, string, string, ...remote.Option) (*v1.AppImage, error) {
		return image, err
	}
}

func testRecordPullEvent(t *testing.T, testName string, appInstance *v1.AppInstance, resolve resolveImageFunc, pull pullImageFunc, now metav1.Time, expect *apiv1.Event) {
	t.Helper()
	var recording []*apiv1.Event
	fakeRecorder := func(_ context.Context, e *apiv1.Event) error {
		recording = append(recording, e)
		return nil
	}

	handler := pullAppImage(nil, pullClient{
		recorder: event.RecorderFunc(fakeRecorder),
		resolve:  resolve,
		pull:     pull,
		now: func() metav1.Time {
			return now
		},
	})

	t.Run(testName, func(t *testing.T) {
		results, err := (&tester.Harness{
			Scheme: scheme.Scheme,
		}).Invoke(t, appInstance, handler)

		require.NoError(t, err)
		assert.Zero(t, results.Client.Created)

		if expect == nil {
			assert.Len(t, recording, 0)
			return
		}

		assert.Len(t, recording, 1)
		assert.EqualValues(t, expect, recording[0])
	})
}

// func testRecordPullEvent(t *testing.T, obj kclient.Object, autoUpgradeOn bool, err error, previousImage, targetImage v1.AppImage, expect *apiv1.Event) {
// 	var recording []*apiv1.Event
// 	fakeRecorder := func(_ context.Context, e *apiv1.Event) error {
// 		recording = append(recording, e)
// 		return nil
// 	}
//
// 	recordPullEvent(context.Background(), event.RecorderFunc(fakeRecorder), obj, autoUpgradeOn, err, previousImage, targetImage)
// 	if expect == nil {
// 		assert.Empty(t, recording)
// 		return
// 	}
//
// 	assert.Len(t, recording, 1)
// 	assert.EqualValues(t, expect, recording[0])
//
// }
