package images

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	api "github.com/acorn-io/acorn/pkg/apis/api.acorn.io"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/pullsecret"
	"github.com/acorn-io/acorn/pkg/remoteopts"
	"github.com/google/go-containerregistry/pkg/name"
	ggcrv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DefaultRegistry = "NO_DEFAULT"
)

func NewImagePush(c client.WithWatch, images *Storage) *ImagePush {
	return &ImagePush{
		client: c,
		images: images,
	}
}

type ImagePush struct {
	images *Storage
	client client.WithWatch
}

func (i *ImagePush) NamespaceScoped() bool {
	return true
}

func (i *ImagePush) New() runtime.Object {
	return &apiv1.ImagePush{}
}

func (i *ImagePush) Connect(ctx context.Context, id string, options runtime.Object, r rest.Responder) (http.Handler, error) {
	id = strings.ReplaceAll(id, "+", "/")

	imageObj, err := i.images.Get(ctx, id, nil)
	if err != nil {
		return nil, err
	}

	image := imageObj.(*apiv1.Image)

	_, process, err := i.ImagePush(ctx, image, id)
	if err != nil {
		return nil, err
	}

	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		for update := range process {
			p := ImageProgress{
				Total:    update.Total,
				Complete: update.Complete,
			}
			if update.Error != nil {
				p.Error = update.Error.Error()
			}
			data, err := json.Marshal(p)
			if err != nil {
				panic("failed to marshal update: " + err.Error())
			}
			_, _ = rw.Write(append(data, '\n'))
		}
	}), nil
}

func (i *ImagePush) NewConnectOptions() (runtime.Object, bool, string) {
	return &apiv1.ImagePush{}, false, ""
}

func (i *ImagePush) ConnectMethods() []string {
	return []string{"POST"}
}

type ImageProgress struct {
	Total    int64  `json:"total,omitempty"`
	Complete int64  `json:"complete,omitempty"`
	Error    string `json:"error,omitempty"`
}

func (i *ImagePush) ImagePush(ctx context.Context, image *apiv1.Image, tagName string) (*apiv1.Image, <-chan ggcrv1.Update, error) {
	pushTag, err := name.NewTag(tagName, name.WithDefaultRegistry(DefaultRegistry))
	if err != nil {
		return nil, nil, err
	}

	if pushTag.Registry.RegistryStr() == DefaultRegistry {
		return nil, nil, apierrors.NewInvalid(schema.GroupKind{
			Group: api.Group,
			Kind:  "Image",
		}, image.Name, field.ErrorList{
			{
				Type:     field.ErrorTypeInvalid,
				Field:    "tags",
				BadValue: tagName,
				Detail:   "Missing registry host in the tag (ie ghcr.io)",
			},
		})
	}

	opts, err := remoteopts.GetRemoteOptions(ctx, i.client)
	if err != nil {
		return nil, nil, err
	}

	repo, err := getRepo(image.Namespace)
	if err != nil {
		return nil, nil, err
	}

	remoteImage, err := remote.Index(repo.Digest(image.Digest), opts...)
	if err != nil {
		return nil, nil, err
	}

	writeOpts, err := remoteopts.GetRemoteWriteOptions(ctx, i.client)
	if err != nil {
		return nil, nil, err
	}

	keyChain, err := pullsecret.Keychain(ctx, i.client, image.Namespace)
	if err != nil {
		return nil, nil, err
	}

	writeOpts = append(writeOpts, remote.WithAuthFromKeychain(keyChain))

	progress := make(chan ggcrv1.Update)
	writeOpts = append(writeOpts, remote.WithProgress(progress))
	go func() { _ = remote.WriteIndex(pushTag, remoteImage, writeOpts...) }()
	return image, progress, nil
}
