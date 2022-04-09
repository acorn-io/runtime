package portforwarder

import (
	"context"
	"net"
	"net/url"
	"strconv"

	"github.com/ibuildthecloud/herd/pkg/k8schannel"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
)

type WebSocketDialer struct {
	dialer *k8schannel.Dialer
	url    string
}

func (w *WebSocketDialer) DialContext(ctx context.Context, address string) (net.Conn, error) {
	conn, err := w.dialer.DialContext(ctx, w.url, nil)
	if err != nil {
		return nil, err
	}
	return conn.ForStream(0), nil
}

func NewWebSocketDialer(cfg *rest.Config, pod *corev1.Pod, port uint32) (*WebSocketDialer, error) {
	url, err := urlForPodAndPort(cfg, pod, port)
	if err != nil {
		return nil, err
	}

	dialer, err := k8schannel.NewDialer(cfg, true)
	if err != nil {
		return nil, err
	}

	newURL := *url
	if newURL.Scheme == "http" {
		newURL.Scheme = "ws"
	} else if newURL.Scheme == "https" {
		newURL.Scheme = "wss"
	}

	return &WebSocketDialer{
		dialer: dialer,
		url:    newURL.String(),
	}, nil
}

func urlForPodAndPort(cfg *rest.Config, pod *corev1.Pod, port uint32) (*url.URL, error) {
	cfg.APIPath = "/api"
	cfg.GroupVersion = &schema.GroupVersion{
		Group:   "",
		Version: "v1",
	}
	restClient, err := rest.RESTClientFor(cfg)
	if err != nil {
		return nil, err
	}

	url := restClient.Get().
		Resource("pods").
		Namespace(pod.Namespace).
		Name(pod.Name).
		SubResource("portforward").
		URL()
	q := url.Query()
	q.Set("ports", strconv.Itoa(int(port)))
	url.RawQuery = q.Encode()
	return url, nil
}
