package images

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/types"
	api "github.com/acorn-io/runtime/pkg/apis/api.acorn.io"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	acornsign "github.com/acorn-io/runtime/pkg/cosign"
	"github.com/acorn-io/runtime/pkg/images"
	"github.com/acorn-io/runtime/pkg/imagesystem"
	"github.com/acorn-io/runtime/pkg/publicname"
	"github.com/acorn-io/runtime/pkg/tables"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Strategy struct {
	client    kclient.Client
	getter    strategy.Getter
	transport http.RoundTripper
}

func NewStrategy(getter strategy.Getter, c kclient.WithWatch, transport http.RoundTripper) *Strategy {
	return &Strategy{
		client:    c,
		getter:    getter,
		transport: transport,
	}
}

func (s *Strategy) validateDelete(ctx context.Context, obj types.Object) (types.Object, error) {
	img := obj.(*apiv1.Image)
	if img.Digest == "" {
		return nil, nil
	}

	apps := &v1.AppInstanceList{}
	err := s.client.List(ctx, apps, &kclient.ListOptions{
		Namespace: img.Namespace,
	})
	if err != nil {
		return nil, err
	}
	for _, app := range apps.Items {
		if app.Status.AppImage.Digest != "" && app.Status.AppImage.Digest == img.Digest {
			if len(img.Tags) > 0 {
				img.Tags = nil
				img.DeletionTimestamp = nil
				return img, s.client.Update(ctx, img)
			}

			name := publicname.Get(&app)
			if app.GetStopped() {
				name = name + " (stopped)"
			}
			return nil, apierrors.NewInvalid(schema.GroupKind{
				Group: api.Group,
				Kind:  "Image",
			}, img.Name, field.ErrorList{
				field.Forbidden(field.NewPath("digest"), fmt.Sprintf("image is in use by app %s", name)),
			})
		}
	}
	return nil, nil
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
	if obj, err := s.validateDelete(ctx, obj); err != nil {
		return nil, err
	} else if obj != nil {
		return obj, nil
	}
	image := obj.(*apiv1.Image)
	imageToDelete := &v1.ImageInstance{}
	err := s.client.Get(ctx, kclient.ObjectKey{Namespace: image.Namespace, Name: image.Name}, imageToDelete)
	if err != nil {
		return nil, err
	}

	// Prune signatures - from cluster (internal) registry only
	if image.Repo == "" { // image.Remote was deprecated
		remoteOpts := []remote.Option{remote.WithTransport(s.transport)}

		// make sure we're only searching in the internal registry
		repo, _, err := imagesystem.GetInternalRepoForNamespace(ctx, s.client, image.Namespace)
		if err != nil {
			return nil, err
		}

		sigTag, sigDigest, err := acornsign.FindSignature(repo.Digest(image.Digest), remoteOpts...)
		if err != nil {
			return nil, err
		}

		if sigDigest.Hex != "" {
			logrus.Debugf("Deleting signature artifact %s (digest %s) from registry", sigTag.Name(), sigDigest.String())
			if err := remote.Delete(sigTag.Context().Digest(sigDigest.String()), remoteOpts...); err != nil {
				return nil, err
			}
		}
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

func findImageMatch(imagelist apiv1.ImageList, imageName string) (*apiv1.Image, string, error) {
	img, match, err := images.FindImageMatch(imagelist, imageName)
	if err != nil {
		if errors.As(err, &images.ErrImageNotFound{}) {
			return nil, match, apierrors.NewNotFound(schema.GroupResource{Group: api.Group, Resource: "images"}, imageName)
		}
		if errors.As(err, &images.ErrImageIdentifierNotUnique{}) {
			return nil, match, apierrors.NewBadRequest(err.Error())
		}
		return nil, match, apierrors.NewInternalError(fmt.Errorf("error while finding image %s", imageName))
	}
	return img, match, err
}
