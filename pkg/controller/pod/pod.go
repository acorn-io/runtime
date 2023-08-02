package pod

import (
	"strconv"

	"github.com/acorn-io/baaah/pkg/name"
	"github.com/acorn-io/baaah/pkg/router"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/event"
	"github.com/acorn-io/runtime/pkg/server/registry/apigroups/acorn/containers"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
)

const (
	deletedEventType   = "ContainerReplicaDeleted"
	createdEventType   = "ContainerReplicaCreated"
	restartedEventType = "ContainerReplicaRestarted"
)

func NewEventHandler(recorder event.Recorder) *EventHandler {
	return &EventHandler{
		recorder: recorder,
	}
}

type EventHandler struct {
	recorder event.Recorder
}

func (h *EventHandler) Handle(req router.Request, _ router.Response) error {
	p := req.Object.(*corev1.Pod)
	var events []*apiv1.Event
	for _, cr := range containers.PodToContainerReplicas(p) {

		var e apiv1.Event
		switch {
		case isDelete(&cr):
			e.Name = name.SafeHashConcatName(
				"d",
				cr.Name,
			)
			e.Type = deletedEventType
			e.Observed = apiv1.NewMicroTime(e.DeletionTimestamp.Time)
		case isCreate(&cr):
			e.Name = name.SafeHashConcatName(
				"c",
				cr.Name,
			)
			e.Type = createdEventType
			e.Observed = apiv1.NewMicroTime(e.CreationTimestamp.Time)
		case isRestart(&cr):
			e.Name = name.SafeHashConcatName(
				"r",
				strconv.FormatInt(int64(cr.Status.RestartCount), 10),
				cr.Name,
			)
			e.Type = restartedEventType
			e.Observed = apiv1.NewMicroTime(e.CreationTimestamp.Time)
		default:
			// Not an event we care to generate, skip it
			continue
		}

		// Set common fields
		e.Namespace = cr.Namespace
		e.Severity = apiv1.EventSeverityInfo
		e.AppName = cr.Spec.AppName
		e.ServiceName = cr.Spec.ContainerName
		e.Resource = event.Resource(&cr)

		events = append(events, &e)
	}

	for _, e := range events {
		// If the event is already recorded, ignore it.
		if err := h.recorder.Record(req.Ctx, e); err != nil && !apierror.IsAlreadyExists(err) {
			// Log recording errors instead of retrying
			logrus.WithError(err).Warn("failed to record containerreplica event")
		}
	}

	return nil
}

func isDelete(cr *apiv1.ContainerReplica) bool {
	return cr.DeletionTimestamp != nil && !cr.DeletionTimestamp.IsZero() && len(cr.Finalizers) == 0
}

func isCreate(cr *apiv1.ContainerReplica) bool {
	return cr.Status.RestartCount == 0 && cr.Status.State == corev1.ContainerState{}
}

func isRestart(cr *apiv1.ContainerReplica) bool {
	return cr.Status.RestartCount > 0
}