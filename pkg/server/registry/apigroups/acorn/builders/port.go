package builders

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/mink/pkg/strategy"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/buildclient"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/system"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	registryrest "k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	_ registryrest.Connecter = (*BuilderPort)(nil)
)

type BuilderPort struct {
	*strategy.DestroyAdapter
	client     kclient.WithWatch
	proxy      httputil.ReverseProxy
	httpClient *http.Client
}

func NewBuilderPort(client kclient.WithWatch, transport http.RoundTripper) (*BuilderPort, error) {
	return &BuilderPort{
		client: client,
		proxy: httputil.ReverseProxy{
			Transport:     transport,
			FlushInterval: 200 * time.Millisecond,
			Director:      func(request *http.Request) {},
		},
		httpClient: &http.Client{
			Transport: transport,
		},
	}, nil
}

func (c *BuilderPort) New() runtime.Object {
	return &apiv1.Builder{}
}

func (c *BuilderPort) NewConnectOptions() (runtime.Object, bool, string) {
	return nil, false, ""
}

func (c *BuilderPort) Connect(ctx context.Context, id string, options runtime.Object, r registryrest.Responder) (http.Handler, error) {
	ns, _ := request.NamespaceFrom(ctx)

	cfg, err := config.Get(ctx, c.client)
	if err != nil {
		return nil, err
	}

	builder := &apiv1.Builder{}
	err = c.client.Get(ctx, router.Key(ns, id), builder)
	if err != nil {
		return nil, err
	}

	builderHost := fmt.Sprintf("%s.%s.%s:8080", builder.Status.ServiceName, system.ImagesNamespace, cfg.InternalClusterDomain)
	// if it's not ready no point in waiting, the caller should have at least waited until it was ready
	if builder.Status.Ready {
		buildclient.PingBuilder(ctx, "http://"+builderHost)
	}

	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		request.URL = &url.URL{
			Scheme: "http",
			Host:   builderHost,
		}
		c.proxy.ServeHTTP(writer, request)
	}), nil
}

func (c *BuilderPort) ConnectMethods() []string {
	return []string{"GET"}
}
