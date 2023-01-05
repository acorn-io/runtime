package registry

import (
	api "github.com/acorn-io/acorn/pkg/apis/api.acorn.io"
	v1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/imagesystem"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/acorn/pkg/server/registry/apps"
	"github.com/acorn-io/acorn/pkg/server/registry/builders"
	"github.com/acorn-io/acorn/pkg/server/registry/builds"
	"github.com/acorn-io/acorn/pkg/server/registry/containers"
	"github.com/acorn-io/acorn/pkg/server/registry/credentials"
	"github.com/acorn-io/acorn/pkg/server/registry/images"
	"github.com/acorn-io/acorn/pkg/server/registry/info"
	"github.com/acorn-io/acorn/pkg/server/registry/secrets"
	"github.com/acorn-io/acorn/pkg/server/registry/volumes"
	"github.com/acorn-io/mink/pkg/serializer"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	clientgo "k8s.io/client-go/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func APIStores(c kclient.WithWatch, cfg, localCfg *clientgo.Config) (map[string]rest.Storage, error) {
	clientFactory, err := client.NewClientFactory(localCfg)
	if err != nil {
		return nil, err
	}

	transport, err := imagesystem.NewAPIBasedTransport(c, cfg)
	if err != nil {
		return nil, err
	}

	buildersStorage := builders.NewStorage(c)
	buildersPort, err := builders.NewBuilderPort(c, transport)
	if err != nil {
		return nil, err
	}

	buildsStorage := builds.NewStorage(c)
	imagesStorage := images.NewStorage(c)

	containersStorage := containers.NewStorage(c)

	containerExec, err := containers.NewContainerExec(c, cfg)
	if err != nil {
		return nil, err
	}

	appsStorage := apps.NewStorage(c, clientFactory)

	logsStorage, err := apps.NewLogs(c, cfg)
	if err != nil {
		return nil, err
	}

	volumesStorage := volumes.NewStorage(c)

	stores := map[string]rest.Storage{
		"acornimagebuilds":       buildsStorage,
		"apps":                   appsStorage,
		"apps/log":               logsStorage,
		"apps/confirmupgrade":    apps.NewConfirmUpgrade(c),
		"apps/pullimage":         apps.NewPullAppImage(c),
		"builders":               buildersStorage,
		"builders/port":          buildersPort,
		"images":                 imagesStorage,
		"images/tag":             images.NewTagStorage(c),
		"images/push":            images.NewImagePush(c, transport),
		"images/pull":            images.NewImagePull(c, clientFactory, transport),
		"images/details":         images.NewImageDetails(c, transport),
		"volumes":                volumesStorage,
		"containerreplicas":      containersStorage,
		"containerreplicas/exec": containerExec,
		"credentials":            credentials.NewStore(c),
		"secrets":                secrets.NewStorage(c),
		"secrets/reveal":         secrets.NewReveal(c),
		"infos":                  info.NewStorage(c),
	}

	return stores, nil
}

func APIGroups(c kclient.WithWatch, cfg, localCfg *clientgo.Config) (*genericapiserver.APIGroupInfo, error) {
	stores, err := APIStores(c, cfg, localCfg)
	if err != nil {
		return nil, err
	}

	newScheme := runtime.NewScheme()
	err = scheme.AddToScheme(newScheme)
	if err != nil {
		return nil, err
	}

	err = v1.AddToSchemeWithGV(newScheme, schema.GroupVersion{
		Group:   api.Group,
		Version: runtime.APIVersionInternal,
	})
	if err != nil {
		return nil, err
	}

	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(api.Group, newScheme, scheme.ParameterCodec, scheme.Codecs)
	apiGroupInfo.VersionedResourcesStorageMap["v1"] = stores
	apiGroupInfo.NegotiatedSerializer = serializer.NewNoProtobufSerializer(apiGroupInfo.NegotiatedSerializer)
	return &apiGroupInfo, nil
}
