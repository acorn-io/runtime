package steve

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/acorn-io/acorn/pkg/server/registry"
	"github.com/rancher/apiserver/pkg/types"
	"github.com/rancher/steve/pkg/attributes"
	"github.com/rancher/wrangler/pkg/schemas"
	serverrest "k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/client-go/rest"
)

type subResources struct {
	defs       map[string]linkActions
	k8sHandler http.Handler
	schemas    *types.APISchemas
}

type linkActions struct {
	ActionHandlers  map[string]http.Handler
	LinkHandlers    map[string]http.Handler
	ResourceActions map[string]schemas.Action
}

func newSubResources(schemas *types.APISchemas, k8sHandler http.Handler, cfg *rest.Config) (*subResources, error) {
	s := &subResources{
		defs:       map[string]linkActions{},
		k8sHandler: k8sHandler,
		schemas:    schemas,
	}

	stores, err := registry.APIStores(nil, cfg)
	if err != nil {
		return nil, err
	}

	return s, s.build(stores)
}

func (s *subResources) build(stores map[string]serverrest.Storage) error {
	for k, v := range stores {
		resource, subResource, ok := strings.Cut(k, "/")
		if !ok {
			continue
		}

		def, ok := s.defs[resource]
		if !ok {
			def = linkActions{
				ActionHandlers:  map[string]http.Handler{},
				ResourceActions: map[string]schemas.Action{},
				LinkHandlers:    map[string]http.Handler{},
			}
			s.defs[resource] = def
		}

		input, err := s.schemas.Import(v.New())
		if err != nil {
			return err
		}

		if _, ok := v.(serverrest.Connecter); ok {
			def.LinkHandlers[subResource] = s.newHandler(subResource)
			def.ResourceActions[subResource] = schemas.Action{
				Input: input.ID,
			}
		} else if _, ok := v.(serverrest.Getter); ok {
			def.LinkHandlers[subResource] = s.newHandler(subResource)
		} else {
			def.ActionHandlers[subResource] = s.newHandler(subResource)
			def.ResourceActions[subResource] = schemas.Action{
				Input:  input.ID,
				Output: input.ID,
			}
		}
	}

	return nil
}

func (s *subResources) newHandler(name string) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		apiContext := types.GetAPIContext(req.Context())
		gvr := attributes.GVR(apiContext.Schema)
		req.URL.Path = fmt.Sprintf("/apis/api.acorn.io/v1/namespaces/%s/%s/%s/%s", apiContext.Namespace, gvr.Resource, apiContext.Name, name)
		s.k8sHandler.ServeHTTP(rw, req)
	})
}

func (s *subResources) Customize(apiSchema *types.APISchema) {
	gvr := attributes.GVR(apiSchema)
	def, ok := s.defs[gvr.Resource]
	if ok {
		apiSchema.ActionHandlers = def.ActionHandlers
		apiSchema.ResourceActions = def.ResourceActions
		apiSchema.LinkHandlers = def.LinkHandlers
	}
}
