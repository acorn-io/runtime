package apps

import (
	"context"
	"encoding/json"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/event"
	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/types"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/sirupsen/logrus"
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

	// Patch is a JSON Merge Patch that describes all changes made to OldSpec by the respective update.
	// See: https://datatracker.ietf.org/doc/html/rfc7386
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
		// updated is nil when err is non-nil.
		return created, err
	}

	defer func() {
		details, err := v1.Mapify(AppSpecCreateEventDetails{
			ResourceVersion: created.GetResourceVersion(),
		})
		if err != nil {
			logrus.Warnf("Failed to generate event details, event recording disabled for request: %s", err.Error())
			return
		}
		if err := s.recorder.Record(ctx, &apiv1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: obj.GetNamespace(),
			},
			Type:     AppCreateEventType,
			Severity: v1.EventSeverityInfo,
			Details:  details,
			Source:   event.ObjectSource(obj),
			Observed: metav1.Now(),
		}); err != nil {
			logrus.Warnf("Failed to record event: %s", err.Error())
		}
	}()

	return created, nil
}

func (s *eventRecordingStrategy) Delete(ctx context.Context, obj types.Object) (types.Object, error) {
	deleted, err := s.CompleteStrategy.Delete(ctx, obj)
	if err != nil {
		// Return deleted because CompleteStrategy.Delete is a black box; i.e. we can't assume
		// updated is nil when err is non-nil.
		return deleted, err
	}

	defer func() {
		details, err := v1.Mapify(AppSpecDeleteEventDetails{
			ResourceVersion: deleted.GetResourceVersion(),
		})
		if err != nil {
			logrus.Warnf("Failed to generate event details, event recording disabled for request: %s", err.Error())
			return
		}
		if err := s.recorder.Record(ctx, &apiv1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: obj.GetNamespace(),
			},
			Type:     AppDeleteEventType,
			Severity: v1.EventSeverityInfo,
			Details:  details,
			Source:   event.ObjectSource(obj),
			Observed: metav1.Now(),
		}); err != nil {
			logrus.Warnf("Failed to record event: %s", err.Error())
		}
	}()

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

	defer func() {
		oldSpec, newSpec := old.(*v1.AppInstance).Spec, updated.(*v1.AppInstance).Spec
		patch, err := mergePatch(oldSpec, newSpec)
		if err != nil {
			logrus.Warnf("Failed to generate app spec patch, event recording disabled for request: %s", err)
			return
		}

		if len(patch) < 1 {
			// Update did not change spec, don't record an event
			logrus.Infof("Update did not change app spec, event recording disabled for request")
			return
		}

		details, err := v1.Mapify(AppSpecUpdateEventDetails{
			ResourceVersion: updated.GetResourceVersion(),
			OldSpec:         obj.(*v1.AppInstance).Spec,
			Patch:           patch,
		})
		if err != nil {
			logrus.Warnf("Failed to generate event details, event recording disabled for request: %s", err)
			return
		}

		if err := s.recorder.Record(ctx, &apiv1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: obj.GetNamespace(),
			},
			Type:     AppSpecUpdateEventType,
			Severity: v1.EventSeverityInfo,
			Details:  details,
			Source:   event.ObjectSource(obj),
			Observed: metav1.Now(),
		}); err != nil {
			logrus.Warnf("Failed to record event: %s", err)
		}
	}()

	return updated, nil
}

func mergePatch(from, to any) (json.RawMessage, error) {
	fromBytes, err := json.Marshal(from)
	if err != nil {
		return nil, err
	}

	toBytes, err := json.Marshal(to)
	if err != nil {
		return nil, err
	}

	patch, err := jsonpatch.CreateMergePatch(fromBytes, toBytes)
	if err != nil {
		return nil, err
	}

	return patch, nil
}
