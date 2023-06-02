package event

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	alabels "github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/baaah"
	bbackend "github.com/acorn-io/baaah/pkg/backend"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AuditCreateEventType      = "AuditCreate"
	AuditDeleteEventType      = "AuditDelete"
	AuditUpdateEventType      = "AuditUpdate"
	AuditPatchEventType       = "AuditPatch"
	AuditDeleteAllOfEventType = "AuditDeleteAllOf"
)

func WithAuditing(r Recorder, o *baaah.Options) *baaah.Options {
	if o == nil || o.Backend == nil {
		panic("invalid baaah options")
	}

	o.Backend = &backend{
		Backend:  o.Backend,
		recorder: r,
	}

	return o
}

type backend struct {
	bbackend.Backend
	recorder Recorder
}

func (b *backend) Create(ctx context.Context, obj kclient.Object, opts ...kclient.CreateOption) error {
	err := b.Backend.Create(ctx, obj, opts...)
	b.record(ctx, AuditCreateEventType, obj, v1.GenericMap{
		"obj": obj,
		"err": err,
	})

	return err
}

func (b *backend) Delete(ctx context.Context, obj kclient.Object, opts ...kclient.DeleteOption) error {
	err := b.Backend.Delete(ctx, obj, opts...)
	b.record(ctx, AuditDeleteEventType, obj, v1.GenericMap{
		"obj": obj,
		"err": err,
	})

	return err
}

func (b *backend) Update(ctx context.Context, obj kclient.Object, opts ...kclient.UpdateOption) error {
	err := b.Backend.Update(ctx, obj, opts...)
	b.record(ctx, AuditUpdateEventType, obj, v1.GenericMap{
		"obj": obj,
		"err": err,
	})

	return err
}

func (b *backend) Patch(ctx context.Context, obj kclient.Object, patch kclient.Patch, opts ...kclient.PatchOption) error {
	err := b.Backend.Patch(ctx, obj, patch, opts...)
	b.record(ctx, AuditPatchEventType, obj, v1.GenericMap{
		"patch": patch,
		"obj":   obj,
		"err":   err,
	})

	return err
}

func (b *backend) DeleteAllOf(ctx context.Context, obj kclient.Object, opts ...kclient.DeleteAllOfOption) error {
	err := b.Backend.DeleteAllOf(ctx, obj, opts...)
	b.record(ctx, AuditDeleteAllOfEventType, obj, v1.GenericMap{
		"obj": obj,
		"err": err,
	})

	return err
}

func (b *backend) record(ctx context.Context, eventType string, obj kclient.Object, details v1.GenericMap) {
	ns := "acorn"
	if appNS, ok := obj.GetAnnotations()[alabels.AcornAppNamespace]; ok {
		ns = appNS
	}

	e := &apiv1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
		},
		Type:     eventType,
		Severity: v1.EventSeverityInfo,
		Details:  details,
		Actor:    "acorn-system",
		Source:   ObjectSource(obj),
		Observed: metav1.NowMicro(),
	}
	if err := b.recorder.Record(ctx, e); err != nil {
		logrus.WithFields(logrus.Fields{
			"event": e,
		}).Warnln("failed to record event", err)
	}
}
