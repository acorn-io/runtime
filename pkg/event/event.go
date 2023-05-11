package event

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO(njhale): Make this recorder non-blocking.
type RecorderFunc func(context.Context, *apiv1.Event) error

func (r RecorderFunc) Record(ctx context.Context, e *apiv1.Event) error {
	return r(ctx, e)
}

func NewBlockingRecorder(c kclient.Client) RecorderFunc {
	return func(ctx context.Context, e *apiv1.Event) error {
		if e.Name == "" && e.GenerateName == "" { // TODO(njhale): This is really kludgey
			e.Name = "placeholder"
		}
		return c.Create(ctx, e)
	}
}
