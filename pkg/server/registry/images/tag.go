package images

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/tags"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/types"
	"github.com/google/go-containerregistry/pkg/name"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewTagStorage(c client.WithWatch) rest.Storage {
	return stores.NewBuilder(c.Scheme(), &apiv1.ImageTag{}).
		WithCreate(&TagStrategy{
			client: c,
		}).Build()
}

type TagStrategy struct {
	client client.WithWatch
}

func (t *TagStrategy) Create(ctx context.Context, obj types.Object) (types.Object, error) {
	opts := obj.(*apiv1.ImageTag)
	image, err := t.ImageTag(ctx, obj.GetNamespace(), obj.GetName(), opts.TagName)
	if err != nil {
		return nil, err
	}
	return &apiv1.ImageTag{
		ObjectMeta: metav1.ObjectMeta{
			Name:      image.Name,
			Namespace: image.Namespace,
		},
		TagName: opts.TagName,
	}, nil
}

func (t *TagStrategy) New() types.Object {
	return &apiv1.ImageTag{}
}

func (t *TagStrategy) ImageTag(ctx context.Context, namespace, imageName, tagName string) (*apiv1.Image, error) {
	image := &apiv1.Image{}
	err := t.client.Get(ctx, router.Key(namespace, imageName), image)
	if err != nil {
		return nil, err
	}

	_, err = name.NewTag(tagName)
	if err != nil {
		return nil, err
	}

	return image, tags.Write(ctx, t.client, image.Namespace, image.Digest, tagName)
}
