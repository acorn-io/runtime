package apps

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/types"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/event"
	"github.com/sirupsen/logrus"
	"github.com/wI2L/jsondiff"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	AppCreateEventType     = "AppCreate"
	AppDeleteEventType     = "AppDelete"
	AppSpecUpdateEventType = "AppSpecUpdate"
)

// AppSpecCreateEventDetails captures additional info about the creation of an App.
type AppSpecCreateEventDetails struct {
	// ResourceVersion is the resourceVersion of the App created.
	ResourceVersion string `json:"resourceVersion"`
}

// AppSpecDeleteEventDetails captures additional info about the deletion of an App.
type AppSpecDeleteEventDetails struct {
	// ResourceVersion is the resourceVersion of the App deleted.
	ResourceVersion string `json:"resourceVersion"`
}

// AppSpecUpdateEventDetails captures additional info about an update to an App Spec.
type AppSpecUpdateEventDetails struct {
	// ResourceVersion is the resourceVersion of the updated App.
	ResourceVersion string `json:"resourceVersion"`

	// OldSpec is the spec of the App before the update.
	OldSpec v1.AppInstanceSpec `json:"oldSpec"`

	// Patch is a JSON Patch that describes all changes made to OldSpec by the respective update.
	// See: https://datatracker.ietf.org/doc/html/rfc6902
	Patch json.RawMessage `json:"patch"`
}

type eventRecordingStrategy struct {
	strategy.CompleteStrategy
	recorder event.Recorder
}

func newEventRecordingStrategy(s strategy.CompleteStrategy, recorder event.Recorder) *eventRecordingStrategy {
	return &eventRecordingStrategy{
		CompleteStrategy: s,
		recorder:         recorder,
	}
}

func (s *eventRecordingStrategy) Create(ctx context.Context, obj types.Object) (types.Object, error) {
	created, err := s.CompleteStrategy.Create(ctx, obj)
	if err != nil {
		// Return created because CompleteStrategy.Create is a black box; i.e. we can't assume
		// created is nil when err is non-nil.
		return created, err
	}

	details, err := apiv1.Mapify(AppSpecCreateEventDetails{
		ResourceVersion: created.GetResourceVersion(),
	})
	if err != nil {
		logrus.Warnf("Failed to generate event details, event recording disabled for request: %s", err.Error())
		return created, nil
	}

	if err := s.recorder.Record(ctx, &apiv1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: obj.GetNamespace(),
		},
		Type:        AppCreateEventType,
		Severity:    apiv1.EventSeverityInfo,
		Details:     details,
		Description: fmt.Sprintf("App %s/%s created", obj.GetNamespace(), obj.GetName()),
		AppName:     obj.GetName(),
		Resource:    event.Resource(obj),
		Observed:    apiv1.NowMicro(),
	}); err != nil {
		logrus.Warnf("Failed to record event: %s", err.Error())
	}

	return created, nil
}

func (s *eventRecordingStrategy) Delete(ctx context.Context, obj types.Object) (types.Object, error) {
	deleted, err := s.CompleteStrategy.Delete(ctx, obj)
	if err != nil {
		// Return deleted because CompleteStrategy.Delete is a black box; i.e. we can't assume
		// deleted is nil when err is non-nil.
		return deleted, err
	}

	details, err := apiv1.Mapify(AppSpecDeleteEventDetails{
		ResourceVersion: deleted.GetResourceVersion(),
	})
	if err != nil {
		logrus.Warnf("Failed to generate event details, event recording disabled for request: %s", err.Error())
		return deleted, nil
	}

	if err := s.recorder.Record(ctx, &apiv1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: obj.GetNamespace(),
		},
		Type:        AppDeleteEventType,
		Severity:    apiv1.EventSeverityInfo,
		Details:     details,
		Description: fmt.Sprintf("App %s/%s deleted", obj.GetNamespace(), obj.GetName()),
		AppName:     obj.GetName(),
		Resource:    event.Resource(obj),
		Observed:    apiv1.NowMicro(),
	}); err != nil {
		logrus.Warnf("Failed to record event: %s", err.Error())
	}

	return deleted, nil
}

func (s *eventRecordingStrategy) Update(ctx context.Context, obj types.Object) (types.Object, error) {
	old, err := s.Get(ctx, obj.GetNamespace(), obj.GetName())
	if err != nil {
		logrus.Warnf("Failed to get old object, event recording disabled for request: %s", err)
		return s.CompleteStrategy.Update(ctx, obj)
	}

	updated, err := s.CompleteStrategy.Update(ctx, obj)
	if err != nil {
		// Return updated because CompleteStrategy.Update is a black box; i.e. we can't assume
		// updated is nil when err is non-nil.
		return updated, err
	}

	oldSpec, newSpec := old.(*apiv1.App).Spec, updated.(*apiv1.App).Spec
	patch, err := jsonPatch(oldSpec, newSpec)
	if err != nil {
		logrus.Warnf("Failed to generate app spec patch, event recording disabled for request: %s", err)
		return updated, nil
	}

	if len(patch) < 1 {
		// Update did not change spec, don't record an event
		logrus.Infof("Update did not change app spec, event recording disabled for request")
		return updated, nil
	}

	details, err := apiv1.Mapify(AppSpecUpdateEventDetails{
		ResourceVersion: updated.GetResourceVersion(),
		OldSpec:         oldSpec,
		Patch:           patch,
	})
	if err != nil {
		logrus.Warnf("Failed to generate event details, event recording disabled for request: %s", err)
		return updated, nil
	}

	if err := s.recorder.Record(ctx, &apiv1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: obj.GetNamespace(),
		},
		Type:        AppSpecUpdateEventType,
		Severity:    apiv1.EventSeverityInfo,
		Details:     details,
		Description: fmt.Sprintf("Spec field updated for App %s/%s", obj.GetNamespace(), obj.GetName()),
		AppName:     obj.GetName(),
		Resource:    event.Resource(obj),
		Observed:    apiv1.NowMicro(),
	}); err != nil {
		logrus.Warnf("Failed to record event: %s", err)
	}

	return updated, nil
}

func jsonPatch(from, to any) (json.RawMessage, error) {
	patch, err := jsondiff.Compare(from, to)
	if err != nil {
		return nil, err
	}

	if len(patch) < 1 {
		// Marshaling an empty patch yields "null", which is harder to reason about.
		// Return nil instead.
		return nil, nil
	}

	return json.Marshal(patch)
}
