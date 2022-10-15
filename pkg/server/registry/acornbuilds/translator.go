package acornbuilds

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
	return &apiv1.AcornBuild{}
}

func (s *Translator) NewPublicList() mtypes.ObjectList {
	return &apiv1.AcornBuildList{}
}

func (s *Translator) FromPublic(ctx context.Context, obj runtime.Object) (mtypes.Object, error) {
	build := obj.(*apiv1.AcornBuild)
	result := &v1.AcornBuild{
		ObjectMeta: build.ObjectMeta,
		Spec:       build.Spec,
		Status:     build.Status,
	}
	result.UID = types.UID(strings.TrimSuffix(string(build.UID), "-ab"))
	return result, nil
}

func (s *Translator) ToPublic(objs ...runtime.Object) []mtypes.Object {
	return typed.MapSlice(objs, func(obj runtime.Object) mtypes.Object {
		build := obj.(*v1.AcornBuild)
		result := &apiv1.AcornBuild{
			ObjectMeta: build.ObjectMeta,
			Spec:       build.Spec,
			Status:     build.Status,
		}
		result.UID = result.UID + "-ab"
		return result
	})
}
