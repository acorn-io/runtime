package images

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"

	api "github.com/acorn-io/acorn/pkg/apis/api.acorn.io"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/build/buildkit"
	"github.com/acorn-io/acorn/pkg/remoteopts"
	tags2 "github.com/acorn-io/acorn/pkg/tags"
	"github.com/acorn-io/mink/pkg/types"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/storage"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Strategy struct {
	client kclient.WithWatch
}

func NewStrategy(c kclient.WithWatch) *Strategy {
	return &Strategy{
		client: c,
	}
}

func (s *Strategy) Get(ctx context.Context, namespace, name string) (types.Object, error) {
	return s.ImageGet(ctx, namespace, name)
}

func (s *Strategy) List(ctx context.Context, namespace string, opts storage.ListOptions) (types.ObjectList, error) {
	images, err := s.ImageList(ctx, namespace)
	if err != nil {
		return nil, err
	}
	return &images, nil
}

func (s *Strategy) New() types.Object {
	return &apiv1.Image{}
}

func (s *Strategy) NewList() types.ObjectList {
	return &apiv1.ImageList{}
}

func (s *Strategy) ImageList(ctx context.Context, namespace string) (apiv1.ImageList, error) {
	if namespace != "" {
		return s.forNamespace(ctx, namespace)
	}

	var (
		result     apiv1.ImageList
		namespaces = &corev1.NamespaceList{}
	)

	err := s.client.List(ctx, namespaces)
	if err != nil {
		return result, err
	}

	sort.Slice(namespaces.Items, func(i, j int) bool {
		return namespaces.Items[i].Name < namespaces.Items[j].Name
	})

	for _, ns := range namespaces.Items {
		list, err := s.forNamespace(ctx, ns.Name)
		if err != nil {
			return result, err
		}
		result.Items = append(result.Items, list.Items...)
	}

	return result, nil
}

func (s *Strategy) forNamespace(ctx context.Context, ns string) (apiv1.ImageList, error) {
	if ok, err := buildkit.Exists(ctx, s.client); err != nil {
		return apiv1.ImageList{}, err
	} else if !ok {
		return apiv1.ImageList{}, nil
	}

	opts, err := remoteopts.WithServerDialer(ctx, s.client)
	if err != nil {
		return apiv1.ImageList{}, err
	}

	repo, err := getRepo(ns)
	if err != nil {
		return apiv1.ImageList{}, err
	}

	names, err := remote.List(repo, opts...)
	if tErr, ok := err.(*transport.Error); ok && tErr.StatusCode == http.StatusNotFound {
		return apiv1.ImageList{}, nil
	}
	if err != nil {
		return apiv1.ImageList{}, err
	}

	tags, err := tags2.Get(ctx, s.client, ns)
	if err != nil {
		return apiv1.ImageList{}, err
	}

	result := apiv1.ImageList{}
	for _, imageName := range names {
		if !tags2.SHAPattern.MatchString(imageName) {
			continue
		}
		tags := tags[imageName]
		if len(tags) == 0 {
			tags = append(tags, "")
		}
		for _, tag := range tags {
			image := apiv1.Image{
				ObjectMeta: metav1.ObjectMeta{
					Name:      imageName,
					Namespace: ns,
				},
				Digest:    "sha256:" + imageName,
				Reference: tag,
			}
			if tag != "" {
				parsedTag, err := name.NewTag(tag)
				if err == nil {
					image.Repository = strings.TrimSuffix(tag, ":"+parsedTag.TagStr())
					image.Tag = parsedTag.TagStr()
				}
			}
			result.Items = append(result.Items, image)
		}
	}

	return result, nil
}

func (s *Strategy) Delete(ctx context.Context, obj types.Object) (types.Object, error) {
	image, matchedReference, err := s.imageGet(ctx, obj.GetNamespace(), obj.GetName())
	if err != nil {
		return nil, err
	}

	if matchedReference != "" {
		tagCount, err := tags2.Remove(ctx, s.client, image.Namespace, image.Digest, matchedReference)
		if tagCount > 0 || err != nil {
			return image, nil
		}
	}

	repo, err := getRepo(image.Namespace)
	if err != nil {
		return nil, err
	}

	opts, err := remoteopts.WithServerDialer(ctx, s.client)
	if err != nil {
		return nil, err
	}

	return image, remote.Delete(repo.Digest(image.Digest), opts...)
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
	images, err := s.ImageList(ctx, namespace)
	if err != nil {
		return nil, "", err
	}

	return findImageMatch(images, imageName)
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
		tag, err := name.ParseReference(imageName)
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
		} else if image.Reference == imageName {
			return &image, image.Tag, nil
		} else if image.Reference != "" {
			imageParsedTag, err := name.NewTag(image.Reference)
			if err != nil {
				continue
			}
			if imageParsedTag.Name() == tagName {
				return &image, image.Reference, nil
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

func getRepo(namespace string) (name.Repository, error) {
	return name.NewRepository("127.0.0.1:5000/acorn/" + namespace)
}
