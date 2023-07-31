package apps

import (
	"context"

	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/types"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	kclient "github.com/acorn-io/runtime/pkg/k8sclient"
	"github.com/acorn-io/runtime/pkg/labels"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewConfirmUpgrade(c client.WithWatch) rest.Storage {
	return stores.NewBuilder(c.Scheme(), &apiv1.ConfirmUpgrade{}).
		WithCreate(&ConfirmUpgradeStrategy{
			client: c,
		}).WithValidateName(nestedValidator{}).Build()
}

type ConfirmUpgradeStrategy struct {
	client client.WithWatch
}

func (s *ConfirmUpgradeStrategy) Create(ctx context.Context, obj types.Object) (types.Object, error) {
	ri, _ := request.RequestInfoFrom(ctx)

	if ri.Name == "" || ri.Namespace == "" {
		return obj, nil
	}

	// Use app instance here because in Manager this request is forwarded to the workload cluster.
	// The app validation logic should not run there.
	app := &v1.AppInstance{}
	err := s.client.Get(ctx, kclient.ObjectKey{Namespace: ri.Namespace, Name: ri.Name}, app)
	if apierrors.IsNotFound(err) {
		// See if this is a public name
		appList := &v1.AppInstanceList{}
		listErr := s.client.List(ctx, appList, client.MatchingLabels{labels.AcornPublicName: ri.Name}, client.InNamespace(ri.Namespace))
		if listErr != nil {
			return nil, listErr
		}
		if len(appList.Items) != 1 {
			//return the NotFound error we got originally
			return nil, err
		}
		app = &appList.Items[0]
	} else if err != nil {
		return nil, err
	}
	app.Status.AvailableAppImage = app.Status.ConfirmUpgradeAppImage

	err = s.client.Status().Update(ctx, app)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (s *ConfirmUpgradeStrategy) New() types.Object {
	return &apiv1.ConfirmUpgrade{}
}
