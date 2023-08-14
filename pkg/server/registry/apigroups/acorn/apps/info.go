package apps

import (
	"context"

	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/types"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	kclient "github.com/acorn-io/runtime/pkg/k8sclient"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/secrets"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewInfo(c client.WithWatch) rest.Storage {
	return stores.NewBuilder(c.Scheme(), &apiv1.AppPullImage{}).
		WithGet(&InfoStrategy{
			client: c,
		}).
		Build()
}

type InfoStrategy struct {
	client client.WithWatch
}

func (s *InfoStrategy) Get(ctx context.Context, namespace, name string) (types.Object, error) {
	ri, _ := request.RequestInfoFrom(ctx)

	app := &apiv1.App{}
	err := s.client.Get(ctx, kclient.ObjectKey{Namespace: ri.Namespace, Name: ri.Name}, app)
	if err != nil {
		return nil, err
	}

	appInstances := &v1.AppInstanceList{}
	err = s.client.List(ctx, appInstances, &kclient.ListOptions{
		LabelSelector: klabels.SelectorFromSet(klabels.Set{
			labels.AcornPublicName: name,
		}),
		Namespace: ri.Namespace,
	})
	if err != nil {
		return nil, err
	}

	if len(appInstances.Items) != 1 {
		return nil, apierrors.NewNotFound(schema.GroupResource{
			Group:    apiv1.SchemeGroupVersion.Group,
			Resource: "apps",
		}, name)
	}

	appInstance := &appInstances.Items[0]

	resp := &apiv1.AppInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ri.Name,
			Namespace: ri.Namespace,
		},
	}
	resp.Info, err = secrets.NewInterpolator(ctx, s.client, appInstance).Replace(appInstance.Status.AppSpec.Info)
	if err != nil {
		resp.InterpolationError = err.Error()
	}
	return resp, nil
}

func (s *InfoStrategy) New() types.Object {
	return &apiv1.AppInfo{}
}
