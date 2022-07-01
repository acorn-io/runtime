package cluster

import (
	"context"
	"net/http"
	"time"

	uiv1 "github.com/acorn-io/acorn/pkg/apis/ui.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/install"
	"github.com/rancher/apiserver/pkg/store/empty"
	"github.com/rancher/apiserver/pkg/types"
	wranglerschemas "github.com/rancher/wrangler/pkg/schemas"
	"k8s.io/apimachinery/pkg/api/equality"
)

func Register(ctx context.Context, schemas *types.APISchemas) {
	schemas.MustImportAndCustomize(&uiv1.Install{}, func(schema *types.APISchema) {
		delete(schema.ResourceFields, "mode")
		f := schema.ResourceFields["image"]
		f.Default = install.DefaultImage()
		schema.ResourceFields["image"] = f
	})

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

func sendErr(result chan<- types.APIEvent, err error) {
	result <- types.APIEvent{
		Error: err,
	}
}

func sendEvent(result chan<- types.APIEvent, eventType string, schema *types.APISchema, cluster uiv1.Cluster) {
	result <- types.APIEvent{
		Name:         eventType,
		ResourceType: schema.ID,
		ID:           cluster.Name,
		Object: types.APIObject{
			Type:   schema.ID,
			ID:     cluster.Name,
			Object: &cluster,
		},
	}
}

func (e *ClusterStore) Watch(apiOp *types.APIRequest, schema *types.APISchema, wr types.WatchRequest) (chan types.APIEvent, error) {
	result := make(chan types.APIEvent)
	go func() {
		defer close(result)
		oldClusters := map[string]uiv1.Cluster{}

		for {
			newClusters := map[string]uiv1.Cluster{}
			clusters, err := ListClusters(apiOp.Context())
			if err != nil {
				sendErr(result, err)
			} else {
				for _, cluster := range clusters {
					if oldCluster, ok := oldClusters[cluster.Name]; !ok && len(oldClusters) > 0 {
						sendEvent(result, types.CreateAPIEvent, schema, cluster)
					} else if !equality.Semantic.DeepEqual(oldCluster, cluster) {
						sendEvent(result, types.ChangeAPIEvent, schema, cluster)
					}
					newClusters[cluster.Name] = cluster
				}

				for k, oldCluster := range oldClusters {
					if _, ok := newClusters[k]; !ok {
						sendEvent(result, types.RemoveAPIEvent, schema, oldCluster)
					}
				}

				oldClusters = newClusters
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
