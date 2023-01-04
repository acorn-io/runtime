package images

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/types"
	"github.com/google/go-containerregistry/pkg/name"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
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

	image, err := t.ImageTag(ctx, obj.GetNamespace(), obj.GetName(), opts.Tags)
	if err != nil {
		return nil, err
	}
	return &apiv1.ImageTag{
		ObjectMeta: metav1.ObjectMeta{
			Name:      image.Name,
			Namespace: image.Namespace,
		},
		Tags: image.Tags,
	}, nil
}

func (t *TagStrategy) New() types.Object {
	return &apiv1.ImageTag{}
}

func (t *TagStrategy) ImageTag(ctx context.Context, namespace, imageName string, tags []string) (*apiv1.Image, error) {
	image := &apiv1.Image{}
	err := t.client.Get(ctx, router.Key(namespace, imageName), image)
	if err != nil {
		return nil, err
	}

	imageList := &apiv1.ImageList{}
	err = t.client.List(ctx, imageList, &client.ListOptions{
		Namespace: namespace,
	})
	if err != nil {
		return nil, err
	}
	set := sets.NewString()
	for _, tag := range append(image.Tags, normalizeTags(tags)...) {
		set.Insert(tag)
	}

	duplicateTag := make(map[string]bool)
	for _, tag := range normalizeTags(tags) {
		duplicateTag[tag] = true
	}
	for _, img := range imageList.Items {
		for i, tag := range normalizeTags(img.Tags) {
			if duplicateTag[tag] && img.Name != image.Name {
				img.Tags = append(img.Tags[:i], img.Tags[i+1:]...)
				err = t.client.Update(ctx, &img)
				if err != nil {
					return image, err
				}
			}
		}
	}

	image.Tags = set.List()
	return image, t.client.Update(ctx, image)
}

func normalizeTags(tags []string) []string {
	var result []string
	for _, tag := range tags {
		imageParsedTag, err := name.NewTag(tag, name.WithDefaultRegistry(""))
		if err != nil {
			return nil
		}
		result = append(result, imageParsedTag.Name())
	}
	return result
}
