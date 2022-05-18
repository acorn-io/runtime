package registry

import (
	api "github.com/acorn-io/acorn/pkg/apis/api.acorn.io"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/acorn/pkg/server/registry/apps"
	"github.com/acorn-io/acorn/pkg/server/registry/containers"
	"github.com/acorn-io/acorn/pkg/server/registry/images"
	"github.com/acorn-io/acorn/pkg/server/registry/volumes"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	clientgo "k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func APIGroups(c client.WithWatch, cfg *clientgo.Config) (*genericapiserver.APIGroupInfo, error) {
	imagesStorage := images.NewStorage(c)
	containerStorage := containers.NewStorage(c)
	tagsStorage := images.NewTagStorage(c, imagesStorage)
	containerExec, err := containers.NewContainerExec(c, containerStorage, cfg)
	if err != nil {
		return nil, err
	}

	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(api.Group, scheme.Scheme, scheme.ParameterCodec, scheme.Codecs)
	apiGroupInfo.VersionedResourcesStorageMap["v1"] = map[string]rest.Storage{
		"apps":                   apps.NewStorage(c, imagesStorage),
		"images":                 imagesStorage,
		"images/tag":             tagsStorage,
		"images/push":            images.NewImagePush(c, imagesStorage),
		"images/pull":            images.NewImagePull(c, imagesStorage, tagsStorage),
		"images/details":         images.NewImageDetails(c, imagesStorage),
		"volumes":                volumes.NewStorage(c),
		"containerreplicas":      containerStorage,
		"containerreplicas/exec": containerExec,
	}

	return &apiGroupInfo, nil
}
