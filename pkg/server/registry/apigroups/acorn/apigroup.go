package acorn

import (
	"github.com/acorn-io/mink/pkg/serializer"
	api "github.com/acorn-io/runtime/pkg/apis/api.acorn.io"
	v1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/event"
	"github.com/acorn-io/runtime/pkg/imagesystem"
	"github.com/acorn-io/runtime/pkg/scheme"
	"github.com/acorn-io/runtime/pkg/server/registry/apigroups/acorn/apps"
	"github.com/acorn-io/runtime/pkg/server/registry/apigroups/acorn/builders"
	"github.com/acorn-io/runtime/pkg/server/registry/apigroups/acorn/builds"
	"github.com/acorn-io/runtime/pkg/server/registry/apigroups/acorn/containers"
	"github.com/acorn-io/runtime/pkg/server/registry/apigroups/acorn/credentials"
	"github.com/acorn-io/runtime/pkg/server/registry/apigroups/acorn/devsessions"
	"github.com/acorn-io/runtime/pkg/server/registry/apigroups/acorn/events"
	"github.com/acorn-io/runtime/pkg/server/registry/apigroups/acorn/imageallowrules"
	"github.com/acorn-io/runtime/pkg/server/registry/apigroups/acorn/images"
	"github.com/acorn-io/runtime/pkg/server/registry/apigroups/acorn/info"
	"github.com/acorn-io/runtime/pkg/server/registry/apigroups/acorn/keys"
	"github.com/acorn-io/runtime/pkg/server/registry/apigroups/acorn/projects"
	"github.com/acorn-io/runtime/pkg/server/registry/apigroups/acorn/regions"
	"github.com/acorn-io/runtime/pkg/server/registry/apigroups/acorn/secrets"
	"github.com/acorn-io/runtime/pkg/server/registry/apigroups/acorn/volumes"
	"github.com/acorn-io/runtime/pkg/server/registry/apigroups/acorn/volumes/class"
	"github.com/acorn-io/runtime/pkg/server/registry/apigroups/admin/computeclass"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	clientgo "k8s.io/client-go/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func Stores(c kclient.WithWatch, cfg, localCfg *clientgo.Config) (map[string]rest.Storage, error) {
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

	portForward, err := containers.NewPortForward(c, cfg)
	if err != nil {
		return nil, err
	}

	appsStorage := apps.NewStorage(c, clientFactory, event.NewRecorder(c))

	logsStorage, err := apps.NewLogs(c, cfg)
	if err != nil {
		return nil, err
	}

	volumesStorage := volumes.NewStorage(c)

	stores := map[string]rest.Storage{
		"acornimagebuilds":              buildsStorage,
		"apps":                          appsStorage,
		"apps/log":                      logsStorage,
		"apps/confirmupgrade":           apps.NewConfirmUpgrade(c),
		"apps/pullimage":                apps.NewPullAppImage(c),
		"apps/ignorecleanup":            apps.NewIgnoreCleanup(c),
		"devsessions":                   devsessions.NewStorage(c, clientFactory),
		"builders":                      buildersStorage,
		"builders/port":                 buildersPort,
		"images":                        imagesStorage,
		"images/tag":                    images.NewTagStorage(c),
		"images/push":                   images.NewImagePush(c, transport),
		"images/pull":                   images.NewImagePull(c, clientFactory, transport),
		"images/details":                images.NewImageDetails(c, transport),
		"images/copy":                   images.NewImageCopy(c, transport),
		"projects":                      projects.NewStorage(c, true),
		"volumes":                       volumesStorage,
		"volumeclasses":                 class.NewClassStorage(c),
		"containerreplicas":             containersStorage,
		"containerreplicas/exec":        containerExec,
		"containerreplicas/portforward": portForward,
		"credentials":                   credentials.NewStore(c),
		"secrets":                       secrets.NewStorage(c),
		"secrets/reveal":                secrets.NewReveal(c),
		"infos":                         info.NewStorage(c),
		"computeclasses":                computeclass.NewAggregateStorage(c),
		"regions":                       regions.NewStorage(c),
		"imageallowrules":               imageallowrules.NewStorage(c),
		"events":                        events.NewStorage(c),
		"publickeys":                    keys.NewStorage(c),
	}

	return stores, nil
}

func APIGroup(c kclient.WithWatch, cfg, localCfg *clientgo.Config) (*genericapiserver.APIGroupInfo, error) {
	stores, err := Stores(c, cfg, localCfg)
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
