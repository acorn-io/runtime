package event

import (
	"context"
	"fmt"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/sirupsen/logrus"
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
			logrus.Debug("No Actor set, attempting to set default from ctx")
			if user, ok := request.UserFrom(ctx); ok {
				e.Actor = user.GetName()
			} else {
				logrus.Debug("Ctx has no user info, generating anonymous event")
			}
		}

		// Set a generated name based on the event content.
		// NOTE: This is validated server-side, so it's important that this is done, just before sending the request.
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
	for i, k := range kinds {
		switch k.Kind {
		case "App", "AppInstance":
			return "app"
		}

		if i == len(kinds)-1 {
			// TODO: Remove this hack
			return strings.ToLower(k.GroupKind().String())
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
