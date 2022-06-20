package apps

import (
	"context"

	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/run"
	"github.com/acorn-io/acorn/pkg/server/registry/images"
	"github.com/acorn-io/acorn/pkg/tables"
	tags2 "github.com/acorn-io/acorn/pkg/tags"
	"github.com/acorn-io/acorn/pkg/watcher"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c client.WithWatch, images *images.Storage) *Storage {
	return &Storage{
		TableConvertor: tables.AppConverter,
		client:         c,
		images:         images,
	}
}

type Storage struct {
	rest.TableConvertor

	client client.WithWatch
	images *images.Storage
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
	ns, _ := request.NamespaceFrom(ctx)
	apps := &v1.AppInstanceList{}
	err := s.client.List(ctx, apps, &client.ListOptions{
		Namespace: ns,
	})
	if err != nil {
		return nil, err
	}

	tags, _ := tags2.Get(ctx, s.client, ns)

	result := &apiv1.AppList{
		ListMeta: metav1.ListMeta{
			ResourceVersion: apps.ResourceVersion,
		},
	}

	for _, app := range apps.Items {
		result.Items = append(result.Items, *appToApp(app, tags))
	}

	return result, nil
}

func appToApp(app v1.AppInstance, tags map[string][]string) *apiv1.App {
	possibleTags := tags[app.Spec.Image]
	if len(possibleTags) > 0 {
		app.Spec.Image = possibleTags[0]
	}
	return &apiv1.App{
		ObjectMeta: app.ObjectMeta,
		Spec:       app.Spec,
		Status:     app.Status,
	}
}

func (s *Storage) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	app := &v1.AppInstance{}
	ns, _ := request.NamespaceFrom(ctx)
	err := s.client.Get(ctx, client.ObjectKey{
		Name:      name,
		Namespace: ns,
	}, app)
	if err != nil {
		return nil, err
	}

	tags, _ := tags2.Get(ctx, s.client, ns)
	return appToApp(*app, tags), nil
}

func (s *Storage) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	if createValidation != nil {
		if err := createValidation(ctx, obj); err != nil {
			return nil, err
		}
	}

	params := obj.(*apiv1.App)

	var (
		app     *v1.AppInstance
		runOpts = run.Options{
			Name:             params.Name,
			GenerateName:     params.GenerateName,
			Namespace:        params.Namespace,
			Annotations:      params.Annotations,
			Labels:           params.Labels,
			Endpoints:        params.Spec.Endpoints,
			Client:           s.client,
			DeployArgs:       params.Spec.DeployArgs,
			Volumes:          params.Spec.Volumes,
			Secrets:          params.Spec.Secrets,
			Services:         params.Spec.Services,
			Ports:            params.Spec.Ports,
			DevMode:          params.Spec.DevMode,
			PublishProtocols: params.Spec.PublishProtocols,
			Profiles:         params.Spec.Profiles,
		}
	)

	image, err := s.resolveTag(ctx, params.Namespace, params.Spec.Image)
	if err != nil {
		return nil, err
	}

	app, err = run.Run(ctx, image, &runOpts)
	if err != nil {
		return nil, err
	}

	tags, _ := tags2.Get(ctx, s.client, app.Namespace)
	return appToApp(*app, tags), err
}

func (s *Storage) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	oldApp := &v1.AppInstance{}
	ns, _ := request.NamespaceFrom(ctx)
	err := s.client.Get(ctx, client.ObjectKey{
		Name:      name,
		Namespace: ns,
	}, oldApp)
	if err != nil {
		return nil, false, err
	}

	tags, _ := tags2.Get(ctx, s.client, ns)

	oldObj := appToApp(*oldApp, tags)
	newObj, err := objInfo.UpdatedObject(ctx, oldObj)
	if err != nil {
		return nil, false, err
	}

	if updateValidation != nil {
		err := updateValidation(ctx, newObj, oldObj)
		if err != nil {
			return nil, false, err
		}
	}

	newApp := newObj.(*apiv1.App)

	if newApp.Spec.Image != oldApp.Spec.Image {
		image, err := s.resolveTag(ctx, ns, newApp.Spec.Image)
		if err != nil {
			return nil, false, err
		}
		newApp.Spec.Image = image
	}

	oldApp.ObjectMeta = newApp.ObjectMeta
	oldApp.Spec = newApp.Spec

	if err := s.client.Update(ctx, oldApp); err != nil {
		return nil, false, err
	}

	return appToApp(*oldApp, tags), false, nil
}

func (s *Storage) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	obj, err := s.Get(ctx, name, nil)
	if err != nil {
		return nil, false, err
	}
	if deleteValidation != nil {
		if err := deleteValidation(ctx, obj); err != nil {
			return nil, false, err
		}
	}
	return obj, true, s.client.Delete(ctx, &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: obj.(*apiv1.App).Namespace,
		},
	})
}

func (s *Storage) Watch(ctx context.Context, options *internalversion.ListOptions) (watch.Interface, error) {
	ns, _ := request.NamespaceFrom(ctx)
	w, err := s.client.Watch(ctx, &v1.AppInstanceList{}, watcher.ListOptions(ns, options))
	if err != nil {
		return nil, err
	}

	return watcher.Transform(w, func(obj runtime.Object) []runtime.Object {
		tags, _ := tags2.Get(ctx, s.client, ns)
		return []runtime.Object{
			appToApp(*obj.(*v1.AppInstance), tags),
		}
	}), nil
}
