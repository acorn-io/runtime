package builders

import (
	"context"
	"net/http"
	"net/http/httputil"
	"time"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/build/buildkit"
	"github.com/acorn-io/acorn/pkg/portforwarder"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/restconfig"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/mink/pkg/strategy"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	registryrest "k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type RegistryPort struct {
	*strategy.DestroyAdapter
	client     kclient.WithWatch
	proxy      httputil.ReverseProxy
	RESTClient rest.Interface
	k8s        kubernetes.Interface
}

func NewRegistryPort(client kclient.WithWatch, cfg *rest.Config) (*RegistryPort, error) {
	cfg = rest.CopyConfig(cfg)
	restconfig.SetScheme(cfg, scheme.Scheme)

	k8s, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	transport, err := rest.TransportFor(cfg)
	if err != nil {
		return nil, err
	}

	return &RegistryPort{
		k8s:    k8s,
		client: client,
		proxy: httputil.ReverseProxy{
			FlushInterval: 200 * time.Millisecond,
			Transport:     transport,
			Director:      func(request *http.Request) {},
		},
		RESTClient: k8s.CoreV1().RESTClient(),
	}, nil
}

func (c *RegistryPort) New() runtime.Object {
	return &apiv1.ContainerReplicaExecOptions{}
}

func (c *RegistryPort) connect(pod *corev1.Pod, port int) (http.Handler, error) {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		request.URL = portforwarder.URLForPortAndPod(c.RESTClient, pod, uint32(port))
		c.proxy.ServeHTTP(writer, request)
	}), nil
}

func (c *RegistryPort) Connect(ctx context.Context, id string, options runtime.Object, r registryrest.Responder) (http.Handler, error) {
	ns, _ := request.NamespaceFrom(ctx)

	builder := &apiv1.Builder{}
	err := c.client.Get(ctx, router.Key(ns, id), builder)
	if err != nil {
		return nil, err
	}

	_, pod, err := buildkit.GetBuildkitPod(ctx, c.client)
	if err != nil {
		return nil, err
	}

	return c.connect(pod, system.RegistryPort)
}

func (c *RegistryPort) NewConnectOptions() (runtime.Object, bool, string) {
	return &apiv1.ContainerReplicaExecOptions{}, false, ""
}

func (c *RegistryPort) ConnectMethods() []string {
	return []string{"GET"}
}
