package projects

import (
	"context"
	"fmt"
	"net/http"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"github.com/acorn-io/mink/pkg/strategy/translation"
	"github.com/acorn-io/mink/pkg/types"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/tables"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c kclient.WithWatch, namespaceCheck bool) rest.Storage {
	remoteResource := translation.NewSimpleTranslationStrategy(&translator{}, remote.NewRemote(&v1.ProjectInstance{}, c))
	validator := &Validator{Client: c}
	return stores.NewBuilder(c.Scheme(), &apiv1.Project{}).
		WithCompleteCRUD(remoteResource).
		WithCreate(&projectCreater{creater: remoteResource, client: c, namespaceCheck: namespaceCheck}).
		WithValidateCreate(validator).
		WithValidateUpdate(validator).
		WithTableConverter(tables.ProjectConverter).
		Build()
}

type projectCreater struct {
	namespaceCheck bool
	creater        strategy.Creater
	client         kclient.Client
}

func (pr *projectCreater) New() types.Object {
	return pr.creater.New()
}

func (pr *projectCreater) Create(ctx context.Context, object types.Object) (types.Object, error) {
	if pr.namespaceCheck {
		ns := &corev1.Namespace{}
		err := pr.client.Get(ctx, router.Key("", object.GetName()), ns)
		if err == nil {
			// Project corresponds to a labeled namespace
			if ns.Labels[labels.AcornProject] != "true" {
				qualifiedResource := schema.GroupResource{
					Resource: "namespaces",
				}
				return nil, &apierrors.StatusError{
					ErrStatus: metav1.Status{
						Status: metav1.StatusFailure,
						Code:   http.StatusConflict,
						Reason: metav1.StatusReasonAlreadyExists,
						Details: &metav1.StatusDetails{
							Group: qualifiedResource.Group,
							Kind:  qualifiedResource.Resource,
							Name:  object.GetName(),
						},
						Message: fmt.Sprintf("%s %q already exists but does not contain the %s=true label",
							qualifiedResource.String(), object.GetName(), labels.AcornProject),
					},
				}
			}
		} else if !apierrors.IsNotFound(err) {
			return nil, err
		}
	}

	return pr.creater.Create(ctx, object)
}
