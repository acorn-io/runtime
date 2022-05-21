package images

import (
	"context"
	"net/http"
	"strings"

	api "github.com/acorn-io/acorn/pkg/apis/api.acorn.io"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/build/buildkit"
	"github.com/acorn-io/acorn/pkg/remoteopts"
	"github.com/acorn-io/acorn/pkg/tables"
	tags2 "github.com/acorn-io/acorn/pkg/tags"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c client.WithWatch) *Storage {
	return &Storage{
		TableConvertor: tables.ImageConverter,
		client:         c,
	}
}

type Storage struct {
	rest.TableConvertor

	client client.WithWatch
}

func (s *Storage) NewList() runtime.Object {
	return &apiv1.ImageList{}
}

func (s *Storage) NamespaceScoped() bool {
	return true
}

func (s *Storage) New() runtime.Object {
	return &apiv1.Image{}
}

func (s *Storage) List(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {
	images, err := s.ImageList(ctx)
	if err != nil {
		return nil, err
	}
	return &images, nil
}

func (s *Storage) ImageList(ctx context.Context) (apiv1.ImageList, error) {
	ns, _ := request.NamespaceFrom(ctx)
	if ns == "" {
		return apiv1.ImageList{}, nil
	}

	if ok, err := buildkit.Exists(ctx, s.client); err != nil {
		return apiv1.ImageList{}, err
	} else if !ok {
		return apiv1.ImageList{}, nil
	}

	opts, err := remoteopts.GetRemoteOptions(ctx, s.client)
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

func (s *Storage) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return s.ImageGet(ctx, name)
}

func (s *Storage) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	image, matchedReference, err := s.imageGet(ctx, name)
	if apierrors.IsNotFound(err) {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}

	if deleteValidation != nil {
		if err := deleteValidation(ctx, image); err != nil {
			return nil, false, err
		}
	}

	if matchedReference != "" {
		tagCount, err := tags2.Remove(ctx, s.client, image.Namespace, image.Digest, matchedReference)
		if tagCount > 0 || err != nil {
			return nil, true, err
		}
	}

	repo, err := getRepo(image.Namespace)
	if err != nil {
		return nil, false, err
	}

	opts, err := remoteopts.GetRemoteOptions(ctx, s.client)
	if err != nil {
		return nil, false, err
	}

	return image, true, remote.Delete(repo.Digest(image.Digest), opts...)
}

func (s *Storage) ImageGet(ctx context.Context, name string) (*apiv1.Image, error) {
	name = strings.ReplaceAll(name, "+", "/")

	if ok, err := buildkit.Exists(ctx, s.client); err != nil {
		return nil, err
	} else if !ok {
		return nil, apierrors.NewNotFound(schema.GroupResource{
			Group:    api.Group,
			Resource: "images",
		}, name)
	}

	image, _, err := s.imageGet(ctx, name)
	return image, err
}

func (s *Storage) imageGet(ctx context.Context, imageName string) (*apiv1.Image, string, error) {
	images, err := s.ImageList(ctx)
	if err != nil {
		return nil, "", err
	}

	var (
		digest       string
		digestPrefix string
		tagName      string
	)

	if strings.HasPrefix(imageName, "sha256:") {
		digest = imageName
	} else if tags2.SHAPattern.MatchString(imageName) {
		digest = "sha256:" + imageName
	} else if tags2.SHAShortPattern.MatchString(imageName) {
		digestPrefix = "sha256:" + imageName
	} else {
		tag, err := name.ParseReference(imageName)
		if err != nil {
			return nil, "", err
		}
		tagName = tag.Name()
	}

	for _, image := range images.Items {
		if image.Digest == digest {
			return &image, "", nil
		} else if digestPrefix != "" && strings.HasPrefix(image.Digest, digestPrefix) {
			return &image, "", nil
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

	return nil, "", apierrors.NewNotFound(schema.GroupResource{
		Group:    api.Group,
		Resource: "images",
	}, imageName)
}

func getRepo(namespace string) (name.Repository, error) {
	return name.NewRepository("127.0.0.1:5000/acorn/" + namespace)
}
