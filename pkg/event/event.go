package event

import (
	"context"
	"fmt"

	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
	must := func(err error) {
		if err != nil {
			panic(fmt.Sprintf("failed to add to scheme: %s", err.Error()))
		}
	}
	must(apiv1.AddToScheme(scheme))
	must(v1.AddToScheme(scheme))
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

func ObjectSource(obj kclient.Object) v1.EventSource {
	return v1.EventSource{
		Kind: publicKind(obj),
		Name: obj.GetName(),
		UID:  obj.GetUID(),
	}
}
