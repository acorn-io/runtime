package portforwarder

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/acorn-io/acorn/pkg/k8schannel"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
)

type WebSocketDialer struct {
	dialer *k8schannel.Dialer
	url    string
}

func (w *WebSocketDialer) DialContext(ctx context.Context, address string) (net.Conn, error) {
	for i := 0; ; i++ {
		// It seems to be that port forwards are very unreliable on connect
		conn, err := w.dialer.DialContext(ctx, w.url, nil)
		if err != nil {
			if i < 5 {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			return nil, fmt.Errorf("failed to connect: %w", err)
		}
		return conn.ForStream(0), nil
	}
}

func NewWebSocketDialerForURL(cfg *rest.Config, url *url.URL) (*WebSocketDialer, error) {
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

func NewWebSocketDialer(cfg *rest.Config, pod *corev1.Pod, port uint32) (*WebSocketDialer, error) {
	cfg.APIPath = "/api"
	cfg.GroupVersion = &schema.GroupVersion{
		Group:   "",
		Version: "v1",
	}
	restClient, err := rest.RESTClientFor(cfg)
	if err != nil {
		return nil, err
	}

	url := URLForPortAndPod(restClient, pod, port)
	return NewWebSocketDialerForURL(cfg, url)
}

func URLForPortAndPod(restClient rest.Interface, pod *corev1.Pod, port uint32) *url.URL {
	url := restClient.Get().
		Resource("pods").
		Namespace(pod.Namespace).
		Name(pod.Name).
		SubResource("portforward").
		URL()
	q := url.Query()
	q.Set("ports", strconv.Itoa(int(port)))
	url.RawQuery = q.Encode()
	return url
}
