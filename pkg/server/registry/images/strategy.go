package images

import (
	"context"
	"fmt"
	"strings"

	api "github.com/acorn-io/acorn/pkg/apis/api.acorn.io"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/tables"
	tags2 "github.com/acorn-io/acorn/pkg/tags"
	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/types"
	"github.com/google/go-containerregistry/pkg/name"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Strategy struct {
	client kclient.Client
	getter strategy.Getter
}

func NewStrategy(getter strategy.Getter, c kclient.WithWatch) *Strategy {
	return &Strategy{
		client: c,
		getter: getter,
	}
}

func (s *Strategy) validateObject(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
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
		result = append(result, field.Forbidden(field.NewPath("digest"), fmt.Sprintf("unable to updates image %s as image digests do not match", newImage.Name[:12])))
		return result
	}
	return s.validateObject(ctx, obj)
}

func (s *Strategy) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return tables.ImageConverter.ConvertToTable(ctx, object, tableOptions)
}

func (s *Strategy) Get(ctx context.Context, namespace, name string) (types.Object, error) {
	obj, err := s.getter.Get(ctx, namespace, name)
	if !apierrors.IsNotFound(err) && err != nil {
		return nil, err
	} else if err == nil {
		return obj, nil
	}
	return s.ImageGet(ctx, namespace, name)
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

	image, _, err := s.findImage(ctx, namespace, name)
	return image, err
}

func (s *Strategy) findImage(ctx context.Context, namespace, imageName string) (*apiv1.Image, string, error) {
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
