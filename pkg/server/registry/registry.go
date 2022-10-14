package registry

import (
	api "github.com/acorn-io/acorn/pkg/apis/api.acorn.io"
	v1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/acorn/pkg/server/registry/apps"
	"github.com/acorn-io/acorn/pkg/server/registry/builders"
	"github.com/acorn-io/acorn/pkg/server/registry/containers"
	"github.com/acorn-io/acorn/pkg/server/registry/credentials"
	"github.com/acorn-io/acorn/pkg/server/registry/images"
	"github.com/acorn-io/acorn/pkg/server/registry/info"
	"github.com/acorn-io/acorn/pkg/server/registry/secrets"
	"github.com/acorn-io/acorn/pkg/server/registry/volumes"
	"github.com/acorn-io/mink/pkg/db"
	"github.com/acorn-io/mink/pkg/serializer"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	clientgo "k8s.io/client-go/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func APIStoresForInspection(cfg *clientgo.Config) (map[string]rest.Storage, error) {
	c, err := k8sclient.New(cfg)
	if err != nil {
		return nil, err
	}
	return APIStores(c, cfg, cfg, nil)
}

func APIStores(c kclient.WithWatch, cfg, localCfg *clientgo.Config, db *db.Factory) (map[string]rest.Storage, error) {
	clientFactory, err := client.NewClientFactory(localCfg)
	if err != nil {
		return nil, err
	}

	buildersStorage, buildersStatus, err := builders.NewStorage(c, db)
	if err != nil {
		return nil, err
	}

	imagesStorage, err := images.NewStorage(c, db)
	if err != nil {
		return nil, err
	}

	containersStorage, containersStatus, err := containers.NewStorage(c, db)
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

	appsStorage, appStatusStorage, err := apps.NewStorage(c, clientFactory, db)
	if err != nil {
		return nil, err
	}

	logsStorage, err := apps.NewLogs(c, cfg)
	if err != nil {
		return nil, err
	}

	volumesStorage, volumesStatus, err := volumes.NewStorage(c, db)
	if err != nil {
		return nil, err
	}

	stores := map[string]rest.Storage{
		"apps":                   appsStorage,
		"apps/status":            appStatusStorage,
		"apps/log":               logsStorage,
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

	if db != nil {
		stores["builders/status"] = buildersStatus
		stores["containerreplicas/status"] = containersStatus
		stores["volumes/status"] = volumesStatus
	}

	return stores, nil
}

func APIGroups(c kclient.WithWatch, cfg, localCfg *clientgo.Config, dsn string) (*genericapiserver.APIGroupInfo, error) {
	var dbFactory *db.Factory
	if dsn != "" {
		dbFactory = db.NewFactory(scheme.Scheme, dsn, nil)
	}
	stores, err := APIStores(c, cfg, localCfg, dbFactory)
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
