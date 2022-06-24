package server

import (
	"context"
	"net"
	"net/http"

	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/acorn/pkg/ui/server/resources/cluster"
	"github.com/acorn-io/acorn/pkg/ui/server/steve"
	"github.com/acorn-io/acorn/pkg/version"
	"github.com/gorilla/mux"
	"github.com/rancher/apiserver/pkg/server"
	"github.com/rancher/apiserver/pkg/store/apiroot"
	"github.com/rancher/apiserver/pkg/subscribe"
	"github.com/rancher/apiserver/pkg/types"
	"github.com/sirupsen/logrus"
)

func New(ctx context.Context, addr string) (string, error) {
	// Create the default server
	s := server.DefaultAPIServer()

	// Register root handler to list api versions
	apiroot.Register(s.Schemas, []string{"v1"})

	// Watches
	subscribe.Register(s.Schemas, subscribe.DefaultGetter, version.Get().String())

	// Cluster Type
	cluster.Register(ctx, s.Schemas)

	// Setup mux router to assign variables the server will look for (refer to MuxURLParser for all variable names)
	router := mux.NewRouter()
	router.StrictSlash(true)

	apiRoot := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		s.Handle(&types.APIRequest{
			Request:   r,
			Response:  rw,
			Type:      "apiRoot",
			URLPrefix: "v1",
		})
	})

	// When a route is found construct a custom API request to serves up the API root content

	steves := steve.New(http.NotFoundHandler())
	router.PathPrefix("/v1/clusters/{name}/v1").Handler(steves)
	router.PathPrefix("/v1/clusters/{name}/schemas").Handler(steves)
	router.Handle("/", apiRoot).Queries("api", "")
	router.Handle("/", apiRoot).HeadersRegexp("Accepts", ".*json.*")
	router.Handle("/v1", apiRoot)
	router.Handle("/{prefix:v1}/{type}", s)
	router.Path("/{prefix:v1}/{type}/{name}").Queries("action", "{action}").Handler(s)
	router.Handle("/{prefix:v1}/{type}/{name}", s)
	router.NotFoundHandler = NotFound(system.IndexURL)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return "", err
	}

	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	server := &http.Server{Handler: router}
	go func() {
		logrus.Fatal(server.Serve(ln))
	}()
	return ln.Addr().String(), nil
}
