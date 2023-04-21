package images

import (
	"context"
	"fmt"
	"strings"

	api "github.com/acorn-io/acorn/pkg/apis/api.acorn.io"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/publicname"
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

func (s *Strategy) validateDelete(ctx context.Context, obj types.Object) error {
	img := obj.(*apiv1.Image)
	if img.Digest == "" {
		return nil
	}

	apps := &v1.AppInstanceList{}
	err := s.client.List(ctx, apps, &kclient.ListOptions{
		Namespace: img.Namespace,
	})
	if err != nil {
		return err
	}
	for _, app := range apps.Items {
		if app.Status.AppImage.Digest != "" && app.Status.AppImage.Digest == img.Digest {
			name := publicname.Get(&app)
			if app.Spec.GetStopped() {
				name = name + " (stopped)"
			}
			return apierrors.NewInvalid(schema.GroupKind{
				Group: api.Group,
				Kind:  "Image",
			}, img.Name, field.ErrorList{
				field.Forbidden(field.NewPath("digest"), fmt.Sprintf("image is in use by app %s", name)),
			})
		}
	}
	return nil
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
	if err := s.validateDelete(ctx, obj); err != nil {
		return nil, err
	}
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

// findImageMatch matches images by digest, digest prefix, or tag name:
//
// - digest (raw): sha256:<digest> or <digest> (exactly 64 chars)
// - digest (image): <registry>/<repo>@sha256:<digest> or <repo>@sha256:<digest>
// - digest prefix: sha256:<digest prefix> (min. 3 chars)
// - tag name: <registry>/<repo>:<tag> or <repo>:<tag>
// - tag name (with default): <registry>/<repo> or <repo> -> Will be matched against the default tag (:latest)
//   - Note: if we get some string here, that matches the SHAPermissivePrefixPattern, it could be both a digest or a name without a tag
//     so we will try to match it against the default tag (:latest) first and if that fails, we treat it as a digest(-prefix)
func findImageMatch(images apiv1.ImageList, search string) (*apiv1.Image, string, error) {
	var (
		repoDigest     name.Digest
		digest         string
		digestPrefix   string
		tagName        string
		tagNameDefault string
		canBeMultiple  bool // if true, we will not return on first match
	)
	if strings.HasPrefix(search, "sha256:") {
		digest = search
	} else if tags2.SHAPattern.MatchString(search) {
		digest = "sha256:" + search
		tagNameDefault = search // this could as well be some name without registry/repo path and tag
	} else if tags2.SHAPermissivePrefixPattern.MatchString(search) {
		digestPrefix = "sha256:" + search
		tagNameDefault = search // this could as well be some name without registry/repo path and tag
	} else {
		ref, err := name.ParseReference(search, name.WithDefaultRegistry(""), name.WithDefaultTag(""))
		if err != nil {
			return nil, "", err
		}
		if ref.Identifier() == "" {
			tagNameDefault = ref.Name() // some name without a tag, so we will try to match it against the default tag (:latest)
			canBeMultiple = true
		} else if dig, ok := ref.(name.Digest); ok {
			repoDigest = dig
		} else {
			tagName = ref.Name()
		}
	}

	if tagNameDefault != "" {
		// add default tag (:latest)
		t, err := name.ParseReference(tagNameDefault, name.WithDefaultRegistry(""))
		if err != nil {
			return nil, "", err
		}
		tagNameDefault = t.Name()
	}

	var matchedImage apiv1.Image
	var matchedTag string
	for _, image := range images.Items {
		// >>> match by tag name with default tag (:latest)
		if tagNameDefault != "" {
			for _, tag := range image.Tags {
				if tag == tagNameDefault {
					return &image, tag, nil
				}
			}
		}

		// >>> match by digest or digest prefix
		if image.Digest == digest {
			return &image, "", nil
		} else if digestPrefix != "" && strings.HasPrefix(image.Digest, digestPrefix) {
			if matchedImage.Digest != "" && matchedImage.Digest != image.Digest {
				return nil, "", apierrors.NewBadRequest(fmt.Sprintf("Image identifier %v is not unique", search))
			}
			matchedImage = image
		}

		// >>> match by repo digest
		// this returns an image which matches the digest and has at least one tag
		// which matches the repo part of the repo digest.
		if repoDigest.Name() != "" && image.Digest == repoDigest.DigestStr() {
			for _, tag := range image.Tags {
				imageParsedTag, err := name.NewTag(tag, name.WithDefaultRegistry(""))
				if err != nil {
					continue
				}
				if imageParsedTag.Context().Name() == repoDigest.Context().Name() {
					return &image, tag, nil
				}
			}
		}

		// >>> match by tag name
		for _, tag := range image.Tags {
			if tag == search {
				if !canBeMultiple {
					return &image, tag, nil
				}
				if matchedImage.Digest != "" && matchedImage.Digest != image.Digest {
					return nil, "", apierrors.NewBadRequest(fmt.Sprintf("Image identifier %v is not unique", search))
				}
				matchedImage = image
				matchedTag = tag
			} else if tag != "" {
				imageParsedTag, err := name.NewTag(tag, name.WithDefaultRegistry(""), name.WithDefaultTag("")) // no default here, as we also have repo-only tag items
				if err != nil {
					continue
				}
				if imageParsedTag.Name() == tagName {
					if !canBeMultiple {
						return &image, tag, nil
					}
					if matchedImage.Digest != "" && matchedImage.Digest != image.Digest {
						return nil, "", apierrors.NewBadRequest(fmt.Sprintf("Image identifier %v is not unique", search))
					}
					matchedImage = image
					matchedTag = tag
				}
			}
		}
	}

	if matchedImage.Digest != "" {
		return &matchedImage, matchedTag, nil
	}

	return nil, "", apierrors.NewNotFound(schema.GroupResource{
		Group:    api.Group,
		Resource: "images",
	}, search)
}
