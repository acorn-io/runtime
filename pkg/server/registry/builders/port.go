package builders

import (
	"context"
	"net/http"
	"net/http/httputil"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/build/buildkit"
	"github.com/acorn-io/acorn/pkg/portforwarder"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/restconfig"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	clientgo "k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type BuildkitPort struct {
	builders   *Storage
	client     client.WithWatch
	proxy      httputil.ReverseProxy
	RESTClient clientgo.Interface
	k8s        kubernetes.Interface
}

func NewBuildkitPort(client client.WithWatch, builders *Storage, cfg *clientgo.Config) (*BuildkitPort, error) {
	cfg = clientgo.CopyConfig(cfg)
	restconfig.SetScheme(cfg, scheme.Scheme)

	k8s, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	transport, err := clientgo.TransportFor(cfg)
	if err != nil {
		return nil, err
	}

	return &BuildkitPort{
		k8s:      k8s,
		client:   client,
		builders: builders,
		proxy: httputil.ReverseProxy{
			Transport: transport,
			Director:  func(request *http.Request) {},
		},
		RESTClient: k8s.CoreV1().RESTClient(),
	}, nil
}

func (c *BuildkitPort) New() runtime.Object {
	return &apiv1.ContainerReplicaExecOptions{}
}

func (c *BuildkitPort) connect(pod *corev1.Pod, port int) (http.Handler, error) {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		request.URL = portforwarder.URLForPortAndPod(c.RESTClient, pod, uint32(port))
		c.proxy.ServeHTTP(writer, request)
	}), nil
}

func (c *BuildkitPort) Connect(ctx context.Context, id string, options runtime.Object, r rest.Responder) (http.Handler, error) {
	_, err := c.builders.Get(ctx, id, nil)
	if err != nil {
		return nil, err
	}

	_, pod, err := buildkit.GetBuildkitPod(ctx, c.client)
	if err != nil {
		return nil, err
	}

	return c.connect(pod, system.BuildkitPort)
}

func (c *BuildkitPort) NewConnectOptions() (runtime.Object, bool, string) {
	return &apiv1.ContainerReplicaExecOptions{}, false, ""
}

func (c *BuildkitPort) ConnectMethods() []string {
	return []string{"GET"}
}
