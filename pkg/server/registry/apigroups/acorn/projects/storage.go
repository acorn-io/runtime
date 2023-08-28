package projects

import (
	"context"
	"fmt"
	"net/http"
	"strings"

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
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c kclient.WithWatch, namespaceCheck bool) (rest.Storage, error) {
	if err := migrateLegacyNamespaces(context.Background(), c); err != nil {
		return nil, err
	}

	remoteResource := translation.NewSimpleTranslationStrategy(&translator{}, remote.NewRemote(&v1.ProjectInstance{}, c))
	validator := &Validator{Client: c}
	return stores.NewBuilder(c.Scheme(), &apiv1.Project{}).
		WithCompleteCRUD(remoteResource).
		WithCreate(&projectCreater{creater: remoteResource, client: c, namespaceCheck: namespaceCheck}).
		WithValidateCreate(validator).
		WithValidateUpdate(validator).
		WithTableConverter(tables.ProjectConverter).
		Build(), nil
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

func migrateLegacyNamespaces(ctx context.Context, c kclient.Client) error {
	// Once a project namespace is managed by the project instance, then this label will be set on it.
	// Therefore, using this label selector ensures the migration only happens once for a namespace.
	notManaged, err := klabels.NewRequirement(labels.AcornManaged, selection.DoesNotExist, nil)
	if err != nil {
		return err
	}

	nsSelector := klabels.SelectorFromSet(map[string]string{
		labels.AcornProject: "true",
	}).Add(*notManaged)
	namespaces := corev1.NamespaceList{}
	if err := c.List(ctx, &namespaces, kclient.MatchingLabelsSelector{Selector: nsSelector}); err != nil {
		return err
	}

	for _, ns := range namespaces.Items {
		project := &v1.ProjectInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns.Name,
			},
			Spec: v1.ProjectInstanceSpec{
				DefaultRegion: ns.Annotations[labels.AcornProjectDefaultRegion],
			},
		}

		if supportedRegions := ns.Annotations[labels.AcornProjectSupportedRegions]; supportedRegions != "" {
			project.Spec.SupportedRegions = strings.Split(supportedRegions, ",")
		}
		if err = c.Create(ctx, project); err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}
	}

	return nil
}
