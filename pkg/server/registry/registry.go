package registry

import (
	api "github.com/acorn-io/acorn/pkg/apis/api.acorn.io"
	v1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/acorn/pkg/server/registry/apps"
	"github.com/acorn-io/acorn/pkg/server/registry/builders"
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

	buildersStorage := builders.NewStorage(c)
	imagesStorage, err := images.NewStorage(c, nil)
	if err != nil {
		return nil, err
	}
	containersStorage, err := containers.NewStorage(c)
	if err != nil {
		return nil, err
	}

	containerExec, err := containers.NewContainerExec(c, cfg)
	if err != nil {
		return nil, err
	}

	buildersPort, err := builders.NewBuildkitPort(c, cfg)
	if err != nil {
		return nil, err
	}

	registryPort, err := builders.NewRegistryPort(c, cfg)
	if err != nil {
		return nil, err
	}

	appsStorage, err := apps.NewStorage(c, clientFactory)
	if err != nil {
		return nil, err
	}

	logsStorage, err := apps.NewLogs(c, cfg)
	if err != nil {
		return nil, err
	}

	volumesStorage, err := volumes.NewStorage(c)
	if err != nil {
		return nil, err
	}

	stores := map[string]rest.Storage{
		"apps":                   appsStorage,
		"apps/log":               logsStorage,
		"apps/confirmupgrade":    apps.NewConfirmUpgrade(c),
		"apps/pullimage":         apps.NewPullAppImage(c),
		"builders":               buildersStorage,
		"builders/port":          buildersPort,
		"builders/registryport":  registryPort,
		"images":                 imagesStorage,
		"images/tag":             images.NewTagStorage(c),
		"images/push":            images.NewImagePush(c),
		"images/pull":            images.NewImagePull(c, clientFactory),
		"images/details":         images.NewImageDetails(c),
		"volumes":                volumesStorage,
		"containerreplicas":      containersStorage,
		"containerreplicas/exec": containerExec,
		"credentials":            credentials.NewStore(c),
		"credentials/expose":     credentials.NewExpose(c),
		"secrets":                secrets.NewStorage(c),
		"secrets/expose":         secrets.NewExpose(c),
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
