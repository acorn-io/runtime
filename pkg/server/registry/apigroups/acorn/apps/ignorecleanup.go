package apps

import (
	"context"
	"fmt"
	"strings"

	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/types"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/controller/jobs"
	kclient "github.com/acorn-io/runtime/pkg/k8sclient"
	"github.com/acorn-io/runtime/pkg/labels"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/utils/strings/slices"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewIgnoreCleanup(c client.WithWatch) rest.Storage {
	return stores.NewBuilder(c.Scheme(), &apiv1.IgnoreCleanup{}).
		WithCreate(&ignoreCleanupStrategy{
			client: c,
		}).
		WithValidateName(ignoreCleanupValidator{}).
		Build()
}

type ignoreCleanupValidator struct{}

func (s ignoreCleanupValidator) ValidateName(ctx context.Context, _ runtime.Object) (result field.ErrorList) {
	ri, _ := request.RequestInfoFrom(ctx)
	for _, piece := range strings.Split(ri.Name, ".") {
		if errs := validation.IsDNS1035Label(piece); len(errs) > 0 {
			result = append(result, field.Invalid(field.NewPath("metadata", "name"), ri.Name, strings.Join(errs, ",")))
		}
	}
	return
}

type ignoreCleanupStrategy struct {
	client client.WithWatch
}

func (s *ignoreCleanupStrategy) Create(ctx context.Context, obj types.Object) (types.Object, error) {
	ri, _ := request.RequestInfoFrom(ctx)

	if ri.Name == "" || ri.Namespace == "" {
		return obj, nil
	}

	// Use app instance here because in Manager this request is forwarded to the workload cluster.
	// The app validation logic should not run there.
	app := &v1.AppInstance{}
	err := s.client.Get(ctx, kclient.ObjectKey{Namespace: ri.Namespace, Name: ri.Name}, app)
	if err != nil && apierrors.IsNotFound(err) {
		// See if this is a public name
		appList := &v1.AppInstanceList{}
		listErr := s.client.List(ctx, appList, client.MatchingLabels{labels.AcornPublicName: ri.Name}, client.InNamespace(ri.Namespace))
		if listErr != nil {
			return nil, listErr
		}
		if len(appList.Items) != 1 {
			// return the NotFound error we got originally
			return nil, err
		}
		app = &appList.Items[0]
	} else if err != nil {
		return nil, err
	}

	if app.DeletionTimestamp.IsZero() {
		return nil, fmt.Errorf("cannot force delete app %s because it is not being deleted", app.Name)
	}

	// If the app has the destroy job finalizer, remove it to force delete
	if idx := slices.Index(app.Finalizers, jobs.DestroyJobFinalizer); idx >= 0 {
		app.Finalizers = append(app.Finalizers[:idx], app.Finalizers[idx+1:]...)

		if err = s.client.Update(ctx, app); err != nil {
			return nil, err
		}
	}

	return obj, nil
}

func (s *ignoreCleanupStrategy) New() types.Object {
	return &apiv1.IgnoreCleanup{}
}
