package images

import (
	"context"
	"fmt"
	"strings"

	api "github.com/acorn-io/acorn/pkg/apis/api.acorn.io"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/build/buildkit"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/tables"
	tags2 "github.com/acorn-io/acorn/pkg/tags"
	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"github.com/acorn-io/mink/pkg/strategy/translation"
	"github.com/acorn-io/mink/pkg/types"
	"github.com/google/go-containerregistry/pkg/name"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Strategy struct {
	strategy.CompleteStrategy
	rest.TableConvertor

	client        kclient.Client
	clientFactory *client.Factory
}

func NewStrategy(c kclient.WithWatch, clientFactory *client.Factory) (strategy.CompleteStrategy, error) {
	storageStrategy, err := newStorageStrategy(c)
	if err != nil {
		return nil, err
	}
	return NewStrategyWithStorage(c, clientFactory, storageStrategy), nil
}

func NewStrategyWithStorage(c kclient.WithWatch, clientFactory *client.Factory, storageStrategy strategy.CompleteStrategy) strategy.CompleteStrategy {
	return &Strategy{
		TableConvertor:   tables.ImageConverter,
		CompleteStrategy: storageStrategy,
		client:           c,
		clientFactory:    clientFactory,
	}
}

func newStorageStrategy(kclient kclient.WithWatch) (strategy.CompleteStrategy, error) {
	return translation.NewTranslationStrategy(
		&Translator{},
		remote.NewRemote(&v1.ImageInstance{}, &v1.ImageInstanceList{}, kclient)), nil
}

// TODO migrate the logic to validateUpdate when create is removed
func (s *Strategy) Validate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	image := obj.(*apiv1.Image)
	duplicateTag := make(map[string]bool)

	for _, tag := range image.Tags {
		imageParsedTag, err := name.NewTag(tag, name.WithDefaultRegistry(""))
		if err != nil {
			continue
		}
		duplicateTag[imageParsedTag.Name()] = true
	}
	imageList := &apiv1.ImageList{}

	err := s.client.List(ctx, imageList, &kclient.ListOptions{
		Namespace: image.Namespace,
	})
	if err != nil {
		result = append(result, field.InternalError(field.NewPath("namespace"), err))
	}

	for _, imageItem := range imageList.Items {
		if imageItem.Digest == image.Digest {
			continue
		}
		for i, tag := range imageItem.Tags {
			if duplicateTag[imageItem.Tags[i]] {
				result = append(result, field.Duplicate(field.NewPath("tag name"), fmt.Errorf("unable to tag image %s with tag %s as it is already in use by %s", image.Name[:12], tag, imageItem.Name[:12])))
			}
		}
	}
	return result
}

func (s *Strategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) (result field.ErrorList) {
	newImage := obj.(*apiv1.Image)
	oldImage := old.(*apiv1.Image)
	if newImage.Digest != oldImage.Digest {
		result = append(result, field.Duplicate(field.NewPath("digest"), fmt.Errorf("unable to updates image %s as image digests do not match", newImage.Name[:12])))
		return result
	}
	return s.Validate(ctx, obj)
}

func (s *Strategy) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return tables.ImageConverter.ConvertToTable(ctx, object, tableOptions)
}

func getRepo(namespace string) (name.Repository, error) {
	return name.NewRepository("127.0.0.1:5000/acorn/" + namespace)
}

func (s *Strategy) Get(ctx context.Context, namespace, name string) (types.Object, error) {
	return s.ImageGet(ctx, namespace, name)
}

// TODO THIS create method go away, since users wont be allowed to Create images, just ImageInstances should be created by the backend
func (s *Strategy) Create(ctx context.Context, obj types.Object) (types.Object, error) {
	image := obj.(*apiv1.Image)

	for i, tag := range image.Tags {
		imageParsedTag, err := name.NewTag(tag, name.WithDefaultRegistry(""))
		if err != nil {
			return nil, err
		}
		if tag != "" {
			image.Tags[i] = imageParsedTag.Name()
		}
	}

	imageInstance := &v1.ImageInstance{
		TypeMeta:   image.TypeMeta,
		ObjectMeta: image.ObjectMeta,
		Digest:     image.Digest,
		Tags:       image.Tags,
	}
	return image, s.client.Create(ctx, imageInstance)
}

func (s *Strategy) Update(ctx context.Context, obj types.Object) (types.Object, error) {
	image := obj.(*apiv1.Image)
	duplicateTag := make(map[string]bool)

	for i, tag := range image.Tags {
		imageParsedTag, err := name.NewTag(tag, name.WithDefaultRegistry(""))
		if err != nil {
			return nil, err
		}
		if tag != "" {
			image.Tags[i] = imageParsedTag.Name()
		}
		currentTag := image.Tags[i]
		if duplicateTag[image.Tags[i]] {
			image.Tags = append(image.Tags[:i], image.Tags[i+1:]...)
		}
		duplicateTag[currentTag] = true
	}
	oldImage := &v1.ImageInstance{}
	err := s.client.Get(ctx, kclient.ObjectKey{Namespace: image.Namespace, Name: image.Name}, oldImage)
	if apierrors.IsNotFound(err) {
		return image, err
	}
	oldImage.Tags = image.Tags

	return image, s.client.Update(ctx, oldImage)
}

func (s *Strategy) Delete(ctx context.Context, obj types.Object) (types.Object, error) {
	image := obj.(*apiv1.Image)
	imageToDelete := &v1.ImageInstance{}
	err := s.client.Get(ctx, kclient.ObjectKey{Namespace: image.Namespace, Name: image.Name}, imageToDelete)
	if err != nil {
		return nil, err
	}
	return image, s.client.Delete(ctx, imageToDelete)

}

func (s *Strategy) ImageGet(ctx context.Context, namespace, name string) (*apiv1.Image, error) {
	name = strings.ReplaceAll(name, "+", "/")

	if ok, err := buildkit.Exists(ctx, s.client); err != nil {
		return nil, err
	} else if !ok {
		return nil, apierrors.NewNotFound(schema.GroupResource{
			Group:    api.Group,
			Resource: "images",
		}, name)
	}

	image, _, err := s.imageGet(ctx, namespace, name)
	return image, err
}

func (s *Strategy) imageGet(ctx context.Context, namespace, imageName string) (*apiv1.Image, string, error) {
	result := &apiv1.ImageList{}

	err := s.client.List(ctx, result, &kclient.ListOptions{
		Namespace: namespace,
	})
	if err != nil {
		return nil, "", err
	}

	return findImageMatch(*result, imageName)
}

func findImageMatch(images apiv1.ImageList, imageName string) (*apiv1.Image, string, error) {
	var (
		digest       string
		digestPrefix string
		tagName      string
	)
	if strings.HasPrefix(imageName, "sha256:") {
		digest = imageName
	} else if tags2.SHAPattern.MatchString(imageName) {
		digest = "sha256:" + imageName
	} else if tags2.SHAPermissivePrefixPattern.MatchString(imageName) {
		digestPrefix = "sha256:" + imageName
	} else {
		_, err := name.NewTag(imageName, name.WithDefaultRegistry(""))
		if err != nil {
			return nil, "", err
		}
		tag, err := name.ParseReference(imageName, name.WithDefaultRegistry(""))
		if err != nil {
			return nil, "", err
		}
		tagName = tag.Name()
	}

	var matchedImage apiv1.Image
	for _, image := range images.Items {
		if image.Digest == digest {
			return &image, "", nil
		} else if digestPrefix != "" && strings.HasPrefix(image.Digest, digestPrefix) {
			if matchedImage.Digest != "" && matchedImage.Digest != image.Digest {
				reason := fmt.Sprintf("Image identifier %v is not unique", imageName)
				return nil, "", apierrors.NewBadRequest(reason)
			}
			matchedImage = image
		}

		for i, tag := range image.Tags {
			if tag == imageName {
				return &image, image.Tags[i], nil
			} else if tag != "" {
				imageParsedTag, err := name.NewTag(tag, name.WithDefaultRegistry(""))
				if err != nil {
					continue
				}
				if imageParsedTag.Name() == tagName {
					return &image, tag, nil
				}
			}
		}
	}

	if matchedImage.Digest != "" {
		return &matchedImage, "", nil
	}

	return nil, "", apierrors.NewNotFound(schema.GroupResource{
		Group:    api.Group,
		Resource: "images",
	}, imageName)
}
