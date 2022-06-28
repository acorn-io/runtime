package registry

import (
	api "github.com/acorn-io/acorn/pkg/apis/api.acorn.io"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/acorn/pkg/server/registry/apps"
	"github.com/acorn-io/acorn/pkg/server/registry/builders"
	"github.com/acorn-io/acorn/pkg/server/registry/containers"
	"github.com/acorn-io/acorn/pkg/server/registry/credentials"
	"github.com/acorn-io/acorn/pkg/server/registry/images"
	"github.com/acorn-io/acorn/pkg/server/registry/info"
	"github.com/acorn-io/acorn/pkg/server/registry/secrets"
	"github.com/acorn-io/acorn/pkg/server/registry/volumes"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	clientgo "k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func APIStores(c client.WithWatch, cfg *clientgo.Config) (map[string]rest.Storage, error) {
	buildersStorage := builders.NewStorage(c)
	imagesStorage := images.NewStorage(c)
	imagesDetails := images.NewImageDetails(c, imagesStorage)
	containerStorage := containers.NewStorage(c)
	secretsStorage := secrets.NewStorage(c)
	tagsStorage := images.NewTagStorage(c, imagesStorage)
	containerExec, err := containers.NewContainerExec(c, containerStorage, cfg)
	if err != nil {
		return nil, err
	}
	buildersPort, err := builders.NewBuildkitPort(c, buildersStorage, cfg)
	if err != nil {
		return nil, err
	}

	registryPort, err := builders.NewRegistryPort(c, buildersStorage, cfg)
	if err != nil {
		return nil, err
	}

	appsStorage := apps.NewStorage(c, imagesStorage, imagesDetails)
	logsStorage, err := apps.NewLogs(c, appsStorage, cfg)
	if err != nil {
		return nil, err
	}

	return map[string]rest.Storage{
		"apps":                   appsStorage,
		"apps/log":               logsStorage,
		"builders":               builders.NewStorage(c),
		"builders/port":          buildersPort,
		"builders/registryport":  registryPort,
		"images":                 imagesStorage,
		"images/tag":             tagsStorage,
		"images/push":            images.NewImagePush(c, imagesStorage),
		"images/pull":            images.NewImagePull(c, imagesStorage, tagsStorage),
		"images/details":         imagesDetails,
		"volumes":                volumes.NewStorage(c),
		"containerreplicas":      containerStorage,
		"containerreplicas/exec": containerExec,
		"credentials":            credentials.NewStorage(c),
		"secrets":                secretsStorage,
		"secrets/expose":         secrets.NewExpose(c, secretsStorage),
		"infos":                  info.NewStorage(c),
	}, nil
}

func APIGroups(c client.WithWatch, cfg *clientgo.Config) (*genericapiserver.APIGroupInfo, error) {
	stores, err := APIStores(c, cfg)
	if err != nil {
		return nil, err
	}

	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(api.Group, scheme.Scheme, scheme.ParameterCodec, scheme.Codecs)
	apiGroupInfo.VersionedResourcesStorageMap["v1"] = stores
	apiGroupInfo.NegotiatedSerializer = &noProtobufSerializer{r: apiGroupInfo.NegotiatedSerializer}
	return &apiGroupInfo, nil
}

type noProtobufSerializer struct {
	r runtime.NegotiatedSerializer
}

func (n *noProtobufSerializer) SupportedMediaTypes() []runtime.SerializerInfo {
	si := n.r.SupportedMediaTypes()
	result := make([]runtime.SerializerInfo, 0, len(si))
	for _, s := range si {
		if s.MediaType == runtime.ContentTypeProtobuf {
			continue
		}
		result = append(result, s)
	}
	return result
}

func (n *noProtobufSerializer) EncoderForVersion(serializer runtime.Encoder, gv runtime.GroupVersioner) runtime.Encoder {
	return n.r.EncoderForVersion(serializer, gv)
}

func (n *noProtobufSerializer) DecoderToVersion(serializer runtime.Decoder, gv runtime.GroupVersioner) runtime.Decoder {
	return n.r.DecoderToVersion(serializer, gv)
}
