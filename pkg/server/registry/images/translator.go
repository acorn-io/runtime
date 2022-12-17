package images

import (
	"context"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/baaah/pkg/typed"
	mtypes "github.com/acorn-io/mink/pkg/types"
	"k8s.io/apimachinery/pkg/runtime"
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
	return &apiv1.Image{}
}

func (s *Translator) NewPublicList() mtypes.ObjectList {
	return &apiv1.ImageList{}
}

func (s *Translator) FromPublic(ctx context.Context, obj runtime.Object) (result mtypes.Object, _ error) {
	return s.fromPublicImage(*obj.(*apiv1.Image)), nil
}

func (s *Translator) ToPublic(objs ...runtime.Object) (result []mtypes.Object) {
	return typed.MapSlice(objs, func(obj runtime.Object) mtypes.Object {
		return s.toPublicImage(*obj.(*v1.ImageInstance))
	})
}

func (s *Translator) fromPublicImage(image apiv1.Image) *v1.ImageInstance {
	return &v1.ImageInstance{
		TypeMeta:   image.TypeMeta,
		ObjectMeta: image.ObjectMeta,
		Digest:     image.Digest,
		Tags:       image.Tags,
	}
}

func (s *Translator) toPublicImage(image v1.ImageInstance) *apiv1.Image {

	return &apiv1.Image{
		TypeMeta:   image.TypeMeta,
		ObjectMeta: image.ObjectMeta,
		Digest:     image.Digest,
		Tags:       image.Tags,
	}
}
