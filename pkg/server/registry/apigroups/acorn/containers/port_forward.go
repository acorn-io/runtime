package containers

import (
	"context"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/acorn-io/baaah/pkg/restconfig"
	"github.com/acorn-io/mink/pkg/strategy"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/k8sclient"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	registryrest "k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type PortForward struct {
	*strategy.DestroyAdapter
	client     kclient.WithWatch
	t          *Translator
	proxy      httputil.ReverseProxy
	RESTClient rest.Interface
}

func NewPortForward(client kclient.WithWatch, cfg *rest.Config) (*PortForward, error) {
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

	return &PortForward{
		t: &Translator{
			client: client,
		},
		client: client,
		proxy: httputil.ReverseProxy{
			FlushInterval: 200 * time.Millisecond,
			Transport:     transport,
			Director:      func(request *http.Request) {},
		},
		RESTClient: k8s.CoreV1().RESTClient(),
	}, nil
}

func (c *PortForward) New() runtime.Object {
	return &apiv1.ContainerReplicaExecOptions{}
}

func (c *PortForward) connect(podName, podNamespace string, execOpt *apiv1.ContainerReplicaPortForwardOptions) (http.Handler, error) {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		req := c.RESTClient.Get().
			Namespace(podNamespace).
			Resource("pods").
			Name(podName).
			SubResource("portforward").
			VersionedParams(&corev1.PodPortForwardOptions{
				Ports: []int32{int32(execOpt.Port)},
			}, scheme.ParameterCodec)
		request.URL = req.URL()
		c.proxy.ServeHTTP(writer, request)
	}), nil
}

func (c *PortForward) Connect(ctx context.Context, id string, options runtime.Object, _ registryrest.Responder) (http.Handler, error) {
	forwardOpts := options.(*apiv1.ContainerReplicaPortForwardOptions)

	container := &apiv1.ContainerReplica{}
	ns, _ := request.NamespaceFrom(ctx)

	err := c.client.Get(ctx, k8sclient.ObjectKey{Namespace: ns, Name: id}, container)
	if err != nil {
		return nil, err
	}

	return c.connect(container.Status.PodName, container.Status.PodNamespace, forwardOpts)
}

func (c *PortForward) NewConnectOptions() (runtime.Object, bool, string) {
	return &apiv1.ContainerReplicaPortForwardOptions{}, false, ""
}

func (c *PortForward) ConnectMethods() []string {
	return []string{"GET"}
}
