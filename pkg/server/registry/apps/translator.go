package apps

import (
	"context"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/baaah/pkg/typed"
	mtypes "github.com/acorn-io/mink/pkg/types"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apiserver/pkg/storage"
)

type Translator struct {
}

func (s *Translator) FromPublicName(ctx context.Context, namespace, name string) (string, string, error) {
	return namespace, name, nil
}

func (s *Translator) ListOpts(namespace string, opts storage.ListOptions) (string, storage.ListOptions) {
	return namespace, opts
}

func (s *Translator) NewPublic() mtypes.Object {
	return &apiv1.App{}
}

func (s *Translator) NewPublicList() mtypes.ObjectList {
	return &apiv1.AppList{}
}

func (s *Translator) FromPublic(ctx context.Context, obj runtime.Object) (result mtypes.Object, _ error) {
	return s.fromPublicApp(*obj.(*apiv1.App)), nil
}

func (s *Translator) ToPublic(objs ...runtime.Object) (result []mtypes.Object) {
	return typed.MapSlice(objs, func(obj runtime.Object) mtypes.Object {
		return s.toPublicApp(*obj.(*v1.AppInstance))
	})
}

func (s *Translator) fromPublicApp(app apiv1.App) *v1.AppInstance {
	app.OwnerReferences = nil
	app.ManagedFields = nil
	app.UID = types.UID(strings.TrimSuffix(string(app.UID), "-a"))
	return &v1.AppInstance{
		ObjectMeta: app.ObjectMeta,
		Spec:       app.Spec,
		Status:     app.Status,
	}
}

func (s *Translator) toPublicApp(app v1.AppInstance) *apiv1.App {
	app.OwnerReferences = nil
	app.ManagedFields = nil
	app.UID = app.UID + "-a"
	return &apiv1.App{
		ObjectMeta: app.ObjectMeta,
		Spec:       app.Spec,
		Status:     app.Status,
	}
}
