package event

import (
	"context"
	"fmt"

	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/z"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Recorder interface {
	Record(context.Context, *apiv1.Event) error
}

type RecorderFunc func(context.Context, *apiv1.Event) error

func (r RecorderFunc) Record(ctx context.Context, e *apiv1.Event) error {
	return r(ctx, e)
}

func NewRecorder(c kclient.Client) RecorderFunc {
	return func(ctx context.Context, e *apiv1.Event) error {
		if e.Actor == "" {
			// Set actor from ctx if possible
			if user, ok := request.UserFrom(ctx); ok {
				e.Actor = user.GetName()
			}
		}

		// Set a generated name based on the event content.
		id, err := ContentID(e)
		if err != nil {
			return fmt.Errorf("failed to generate event name from content: %w", err)
		}
		e.Name = id

		return c.Create(ctx, e)
	}
}

var (
	scheme = runtime.NewScheme()
)

func init() {
	z.Must(
		apiv1.AddToScheme(scheme),
		internalv1.AddToScheme(scheme),
	)
}

func publicKind(obj runtime.Object) string {
	kinds, _, _ := scheme.ObjectKinds(obj)
	for _, k := range kinds {
		switch k.Kind {
		case "App", "AppInstance":
			return "app"
		}
	}
	return ""
}

// Resource returns a non-nil pointer to a v1.EventResource for the given object.
func Resource(obj kclient.Object) *apiv1.EventResource {
	return &apiv1.EventResource{
		Kind: publicKind(obj),
		Name: obj.GetName(),
		UID:  obj.GetUID(),
	}
}
