package images

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/tags"
	"github.com/google/go-containerregistry/pkg/name"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewTagStorage(c client.WithWatch, images *Storage) *TagStorage {
	return &TagStorage{
		client: c,
		images: images,
	}
}

type TagStorage struct {
	images *Storage
	client client.WithWatch
}

func (s *TagStorage) NamespaceScoped() bool {
	return true
}

func (s *TagStorage) New() runtime.Object {
	return &apiv1.ImageTag{}
}

func (s *TagStorage) Create(ctx context.Context, name string, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	if createValidation != nil {
		if err := createValidation(ctx, obj); err != nil {
			return nil, err
		}
	}

	opts := obj.(*apiv1.ImageTag)
	image, err := s.ImageTag(ctx, name, opts.TagName)
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

func (s *TagStorage) ImageTag(ctx context.Context, imageName, tagName string) (*apiv1.Image, error) {
	imageObj, err := s.images.Get(ctx, imageName, nil)
	if err != nil {
		return nil, err
	}

	image := imageObj.(*apiv1.Image)

	_, err = name.NewTag(tagName)
	if err != nil {
		return nil, err
	}

	return image, tags.Write(ctx, s.client, image.Namespace, image.Digest, tagName)
}
