package steve

import (
	"context"
	"fmt"
	"net/http"
	"time"

	api "github.com/acorn-io/acorn/pkg/apis/api.acorn.io"
	cluster2 "github.com/acorn-io/acorn/pkg/ui/server/resources/cluster"
	"github.com/acorn-io/acorn/pkg/version"
	"github.com/gorilla/mux"
	"github.com/moby/locker"
	"github.com/rancher/apiserver/pkg/types"
	"github.com/rancher/apiserver/pkg/urlbuilder"
	"github.com/rancher/steve/pkg/attributes"
	"github.com/rancher/steve/pkg/schema"
	"github.com/rancher/steve/pkg/server"
	"github.com/rancher/steve/pkg/server/router"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	schema2 "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
)

type Steve struct {
	locker  locker.Locker
	servers map[string]http.Handler
	next    http.Handler
}

func New(next http.Handler) *Steve {
	return &Steve{
		servers: map[string]http.Handler{},
		next:    next,
	}
}

func (s *Steve) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	handler, err := s.getHandler(req.Context(), mux.Vars(req)["name"])
	if apierrors.IsNotFound(err) {
		s.next.ServeHTTP(rw, req)
		return
	} else if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	handler.ServeHTTP(rw, req)
}

func (s *Steve) getHandler(ctx context.Context, clusterName string) (http.Handler, error) {
	s.locker.Lock(clusterName)
	defer func() {
		_ = s.locker.Unlock(clusterName)
	}()

	handler, ok := s.servers[clusterName]
	if !ok {
		cluster, err := cluster2.GetConfig(clusterName)
		if err != nil {
			return nil, err
		}
		handler, err = newSteve(ctx, clusterName, cluster.Config, s.next)
		if err != nil {
			return nil, err
		}
		s.servers[clusterName] = handler
	}

	return handler, nil
}

func newSteve(ctx context.Context, name string, restConfig *rest.Config, next http.Handler) (http.Handler, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	controllers, err := server.NewController(restConfig, nil)
	if err != nil {
		return nil, err
	}

	var k8sHandler http.Handler
	prefix := fmt.Sprintf("/v1/clusters/%s", name)
	s, err := server.New(context.Background(), restConfig, &server.Options{
		Controllers: controllers,
		Next:        next,
		Router: func(h router.Handlers) http.Handler {
			k8sHandler = h.K8sProxy

			m := mux.NewRouter().PathPrefix(prefix).Subrouter()
			m.Use(func(handler http.Handler) http.Handler {
				return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
					req.Header.Set(urlbuilder.PrefixHeader, prefix)
					handler.ServeHTTP(rw, req)
				})
			})

			m.UseEncodedPath()
			m.StrictSlash(true)
			m.Use(urlbuilder.RedirectRewrite)

			m.Path("/").Handler(h.APIRoot).HeadersRegexp("Accepts", ".*json.*")
			m.Path("/{name:v1}").Handler(h.APIRoot)

			m.Path("/v1/{type}").Handler(h.K8sResource)
			m.Path("/v1/{type}/{nameorns}").Queries("link", "{link}").Handler(h.K8sResource)
			m.Path("/v1/{type}/{nameorns}").Queries("action", "{action}").Handler(h.K8sResource)
			m.Path("/v1/{type}/{nameorns}").Handler(h.K8sResource)
			m.Path("/v1/{type}/{namespace}/{name}").Queries("action", "{action}").Handler(h.K8sResource)
			m.Path("/v1/{type}/{namespace}/{name}").Queries("link", "{link}").Handler(h.K8sResource)
			m.Path("/v1/{type}/{namespace}/{name}").Handler(h.K8sResource)
			m.Path("/v1/{type}/{namespace}/{name}/{link}").Handler(h.K8sResource)
			m.NotFoundHandler = h.Next

			return m
		},
		ServerVersion: version.Get().String(),
	})
	if err != nil {
		return nil, err
	}

	subResources, err := newSubResources(s.BaseSchemas, k8sHandler, restConfig)
	if err != nil {
		return nil, err
	}

	s.SchemaFactory.AddTemplate(schema.Template{
		Customize: func(apiSchema *types.APISchema) {
			gvr := attributes.GVR(apiSchema)
			if gvr.Resource == "" || gvr.Resource == "namespaces" {
				return
			}
			if gvr.Group == "api.acorn.io" {
				subResources.Customize(apiSchema)
				apiSchema.Formatter = func(request *types.APIRequest, resource *types.RawResource) {
					delete(resource.Links, "view")
				}
			} else {
				attributes.SetVerbs(apiSchema, nil)
			}
		},
	})

	_ = controllers.Start(context.Background())

	for {
		if s.ClusterCache.List(schema2.GroupVersionKind{
			Group:   api.Group,
			Version: "v1",
			Kind:    "App",
		}) != nil {
			break
		}
		select {
		case <-ctx.Done():
			break
		case <-time.After(100 * time.Millisecond):
		}
	}

	return s, nil
}
