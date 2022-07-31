package apps

import (
	"context"
	"strings"

	api "github.com/acorn-io/acorn/pkg/apis/api.acorn.io"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/namespace"
	"github.com/acorn-io/acorn/pkg/run"
	"github.com/acorn-io/acorn/pkg/server/registry/images"
	"github.com/acorn-io/acorn/pkg/tables"
	tags2 "github.com/acorn-io/acorn/pkg/tags"
	"github.com/acorn-io/acorn/pkg/watcher"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c client.WithWatch, images *images.Storage, imageDetails *images.ImageDetails) *Storage {
	return &Storage{
		TableConvertor: tables.AppConverter,
		client:         c,
		images:         images,
		imageDetails:   imageDetails,
	}
}

type Storage struct {
	rest.TableConvertor

	client       client.WithWatch
	images       *images.Storage
	imageDetails *images.ImageDetails
}

func (s *Storage) NewList() runtime.Object {
	return &apiv1.AppList{}
}

func (s *Storage) NamespaceScoped() bool {
	return true
}

func (s *Storage) New() runtime.Object {
	return &apiv1.App{}
}

func (s *Storage) List(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {
	apps := &v1.AppInstanceList{}
	err := s.client.List(ctx, apps, &client.ListOptions{
		LabelSelector: namespace.Selector(ctx),
	})
	if err != nil {
		return nil, err
	}

	tagCache := map[string]map[string][]string{}
	result := &apiv1.AppList{
		ListMeta: apps.ListMeta,
	}

	for _, app := range apps.Items {
		result.Items = append(result.Items, *s.appToApp(ctx, app, tagCache))
	}

	return result, nil
}

func (s *Storage) appToApp(ctx context.Context, app v1.AppInstance, tagCache map[string]map[string][]string) *apiv1.App {
	rootNS := app.Labels[labels.AcornRootNamespace]
	tags, ok := tagCache[rootNS]
	if !ok {
		cfg, err := tags2.Get(ctx, s.client, rootNS)
		if err == nil {
			tags = cfg
			if tagCache != nil {
				tagCache[rootNS] = cfg
			}
		}
	}
	possibleTags := tags[app.Spec.Image]
	if len(possibleTags) > 0 {
		app.Spec.Image = possibleTags[0]
	}

	app.Namespace, app.Name = namespace.NormalizedName(app.ObjectMeta)
	app.OwnerReferences = nil
	app.UID = app.UID + "-a"
	return &apiv1.App{
		ObjectMeta: app.ObjectMeta,
		Spec:       app.Spec,
		Status:     app.Status,
	}
}

func notFoundApp(name string) error {
	return apierrors.NewNotFound(schema.GroupResource{
		Group:    api.Group,
		Resource: "apps",
	}, name)
}

func (s *Storage) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	app, err := s.get(ctx, name)
	if err != nil {
		return nil, err
	}
	return s.appToApp(ctx, *app, nil), nil
}

func (s *Storage) get(ctx context.Context, name string) (*v1.AppInstance, error) {
	ns, appName, err := namespace.DenormalizeName(ctx, s.client, name)
	if apierrors.IsNotFound(err) {
		return nil, notFoundApp(name)
	} else if err != nil {
		return nil, err
	}

	app := &v1.AppInstance{}
	err = s.client.Get(ctx, client.ObjectKey{
		Name:      appName,
		Namespace: ns,
	}, app)
	if apierrors.IsNotFound(err) {
		return nil, notFoundApp(name)
	} else if err != nil {
		return nil, err
	}

	return app, nil
}

func (s *Storage) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	if createValidation != nil {
		if err := createValidation(ctx, obj); err != nil {
			return nil, err
		}
	}

	params := obj.(*apiv1.App)
	app := &v1.AppInstance{
		ObjectMeta: params.ObjectMeta,
		Spec:       params.Spec,
	}

	image, err := s.resolveTag(ctx, params.Namespace, params.Spec.Image)
	if err != nil {
		return nil, err
	}

	perms, err := s.getPermissions(ctx, image)
	if err != nil {
		return nil, err
	}

	if err := s.compareAndCheckPermissions(ctx, perms, app.Spec.Permissions); err != nil {
		return nil, err
	}

	app.Spec.Image = image

	app, err = run.Run(ctx, s.client, app)
	if err != nil {
		return nil, err
	}

	return s.appToApp(ctx, *app, nil), err
}

func (s *Storage) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	oldAppInstance, err := s.get(ctx, name)
	if err != nil {
		return nil, false, err
	}

	appToUpdate := &apiv1.App{
		ObjectMeta: oldAppInstance.ObjectMeta,
		Spec:       oldAppInstance.Spec,
	}
	appToUpdate.UID = appToUpdate.UID + "-a"
	appToUpdate.Namespace, appToUpdate.Name = namespace.NormalizedName(appToUpdate.ObjectMeta)

	newObj, err := objInfo.UpdatedObject(ctx, appToUpdate)
	if err != nil {
		return nil, false, err
	}
	newApp := newObj.(*apiv1.App)

	if updateValidation != nil {
		err := updateValidation(ctx, newObj, appToUpdate)
		if err != nil {
			return nil, false, err
		}
	}

	if newApp.Spec.Image != oldAppInstance.Spec.Image {
		image, err := s.resolveTag(ctx, appToUpdate.Namespace, newApp.Spec.Image)
		if err != nil {
			return nil, false, err
		}
		newApp.Spec.Image = image
	}

	updatedAppInstance := &v1.AppInstance{
		ObjectMeta: newApp.ObjectMeta,
		Spec:       newApp.Spec,
	}
	updatedAppInstance.Name = oldAppInstance.Name
	updatedAppInstance.Namespace = oldAppInstance.Namespace
	updatedAppInstance.UID = types.UID(strings.TrimSuffix(string(updatedAppInstance.UID), "-a"))

	perms, err := s.getPermissions(ctx, updatedAppInstance.Spec.Image)
	if err != nil {
		return nil, false, err
	}

	if err := s.compareAndCheckPermissions(ctx, perms, updatedAppInstance.Spec.Permissions); err != nil {
		return nil, false, err
	}

	if err := s.client.Update(ctx, updatedAppInstance); err != nil {
		return nil, false, err
	}

	return s.appToApp(ctx, *updatedAppInstance, nil), false, nil
}

func (s *Storage) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	oldApp, err := s.get(ctx, name)
	if err != nil {
		return nil, false, err
	}
	if deleteValidation != nil {
		if err := deleteValidation(ctx, oldApp); err != nil {
			return nil, false, err
		}
	}

	return s.appToApp(ctx, *oldApp, nil), true, s.client.Delete(ctx, oldApp)
}

func (s *Storage) Watch(ctx context.Context, options *internalversion.ListOptions) (watch.Interface, error) {
	opts := watcher.ListOptions("", options)
	opts.LabelSelector = namespace.Selector(ctx)
	w, err := s.client.Watch(ctx, &v1.AppInstanceList{}, opts)
	if err != nil {
		return nil, err
	}

	return watcher.Transform(w, func(obj runtime.Object) (result []runtime.Object) {
		app := obj.(*v1.AppInstance)

		newApp := s.appToApp(ctx, *app, nil)
		if options.FieldSelector != nil {
			if !options.FieldSelector.Matches(fields.Set{
				"metadata.name":      newApp.Name,
				"metadata.namespace": newApp.Namespace,
			}) {
				return nil
			}
		}
		result = append(result, newApp)
		return
	}), nil
}
