package cluster

import (
	"context"
	"net/http"
	"time"

	uiv1 "github.com/acorn-io/acorn/pkg/apis/ui.acorn.io/v1"
	"github.com/rancher/apiserver/pkg/store/empty"
	"github.com/rancher/apiserver/pkg/types"
	wranglerschemas "github.com/rancher/wrangler/pkg/schemas"
)

func Register(ctx context.Context, schemas *types.APISchemas) {
	if _, err := schemas.Import(&uiv1.Install{}); err != nil {
		panic(err)
	}

	go startPolling(ctx)

	schemas.MustImportAndCustomize(&uiv1.Cluster{}, func(schema *types.APISchema) {
		schema.Store = &ClusterStore{}
		schema.ActionHandlers = map[string]http.Handler{
			"init": http.HandlerFunc(Install),
		}
		schema.CollectionMethods = []string{http.MethodGet}
		schema.ResourceMethods = []string{http.MethodGet}
		schema.ResourceActions = map[string]wranglerschemas.Action{
			"init": {
				Input: "install",
			},
		}
		schema.Formatter = func(request *types.APIRequest, resource *types.RawResource) {
			if resource.APIObject.Object.(*uiv1.Cluster).Status.Installed {
				resource.Links["api"] = resource.Links["self"] + "/v1"
			}
		}
	})
}

type ClusterStore struct {
	empty.Store
}

func (e *ClusterStore) ByID(apiOp *types.APIRequest, schema *types.APISchema, id string) (types.APIObject, error) {
	cluster, err := GetCluster(apiOp.Context(), id)
	if err != nil {
		return types.APIObject{}, err
	}
	return types.APIObject{
		ID:     cluster.Name,
		Object: cluster,
	}, nil
}

func (e *ClusterStore) List(apiOp *types.APIRequest, schema *types.APISchema) (result types.APIObjectList, _ error) {
	clusters, err := ListClusters(apiOp.Context())
	if err != nil {
		return result, err
	}

	for _, cluster := range clusters {
		c := cluster
		result.Objects = append(result.Objects, types.APIObject{
			ID:     cluster.Name,
			Object: &c,
		})
	}

	return
}

func (e *ClusterStore) Watch(apiOp *types.APIRequest, schema *types.APISchema, wr types.WatchRequest) (chan types.APIEvent, error) {
	result := make(chan types.APIEvent)
	go func() {
		defer close(result)
		for {
			clusters, err := ListClusters(apiOp.Context())
			if err == nil {
				for _, cluster := range clusters {
					cluster := cluster
					result <- types.APIEvent{
						Name:         cluster.Name,
						Namespace:    "",
						ResourceType: schema.ID,
						ID:           cluster.Name,
						Object: types.APIObject{
							Type:   schema.ID,
							ID:     cluster.Name,
							Object: &cluster,
						},
					}
				}
			}

			select {
			case <-apiOp.Context().Done():
				return
			case <-time.After(2 * time.Second):
			}
		}
	}()
	return result, nil
}
