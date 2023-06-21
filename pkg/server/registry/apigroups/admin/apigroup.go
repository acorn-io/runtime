package admin

import (
	"github.com/acorn-io/mink/pkg/serializer"
	adminapi "github.com/acorn-io/runtime/pkg/apis/admin.acorn.io"
	v1 "github.com/acorn-io/runtime/pkg/apis/admin.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/scheme"
	"github.com/acorn-io/runtime/pkg/server/registry/apigroups/admin/computeclass"
	"github.com/acorn-io/runtime/pkg/server/registry/apigroups/admin/volumeclass"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	clientgo "k8s.io/client-go/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func Stores(c kclient.WithWatch) (map[string]rest.Storage, error) {
	return map[string]rest.Storage{
		"clustercomputeclasses": computeclass.NewClusterStorage(c),
		"projectcomputeclasses": computeclass.NewProjectStorage(c),
		"clustervolumeclasses":  volumeclass.NewClusterStorage(c),
		"projectvolumeclasses":  volumeclass.NewProjectStorage(c),
	}, nil
}

func APIGroup(c kclient.WithWatch, _, _ *clientgo.Config) (*genericapiserver.APIGroupInfo, error) {
	stores, err := Stores(c)
	if err != nil {
		return nil, err
	}

	newScheme := runtime.NewScheme()
	err = scheme.AddToScheme(newScheme)
	if err != nil {
		return nil, err
	}

	err = v1.AddToSchemeWithGV(newScheme, schema.GroupVersion{
		Group:   adminapi.Group,
		Version: runtime.APIVersionInternal,
	})
	if err != nil {
		return nil, err
	}

	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(adminapi.Group, newScheme, scheme.ParameterCodec, scheme.Codecs)
	apiGroupInfo.VersionedResourcesStorageMap["v1"] = stores
	apiGroupInfo.NegotiatedSerializer = serializer.NewNoProtobufSerializer(apiGroupInfo.NegotiatedSerializer)
	return &apiGroupInfo, nil
}
