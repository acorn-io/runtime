package k8schannel

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"k8s.io/client-go/rest"
)

type Dialer struct {
	dialer    *websocket.Dialer
	headers   http.Header
	needsInit bool
}

func (d *Dialer) DialContext(ctx context.Context, url string, headers http.Header) (*Connection, error) {
	newHeaders := http.Header{}
	for k, v := range d.headers {
		newHeaders[k] = v
	}
	for k, v := range headers {
		newHeaders[k] = v
	}

	if strings.HasPrefix(url, "http") {
		url = strings.Replace(url, "http", "ws", 1)
	}

	conn, resp, err := d.dialer.DialContext(ctx, url, newHeaders)
	if err != nil {
		if resp != nil && resp.Body != nil {
			data, readErr := ioutil.ReadAll(resp.Body)
			if readErr == nil && len(data) > 0 {
				return nil, fmt.Errorf("%w: %s", err, data)
			}
		}
		return nil, err
	}
	return NewConnection(conn, d.needsInit), nil
}

type headerCapture struct {
	headers http.Header
}

func GetHeadersFor(cfg *rest.Config) (http.Header, error) {
	headerCapture := &headerCapture{}
	rt, err := rest.HTTPWrappersForConfig(cfg, headerCapture)
	if err != nil {
		return nil, err
	}
	_, err = rt.RoundTrip(&http.Request{})
	return headerCapture.headers, err
}

func (h *headerCapture) RoundTrip(request *http.Request) (*http.Response, error) {
	h.headers = request.Header
	return &http.Response{}, nil
}

func NewDialer(cfg *rest.Config, needsInit bool) (*Dialer, error) {
	tlsConfig, err := rest.TLSConfigFor(cfg)
	if err != nil {
		return nil, err
	}

	headers, err := GetHeadersFor(cfg)
	if err != nil {
		return nil, err
	}

	return &Dialer{
		needsInit: needsInit,
		dialer: &websocket.Dialer{
			Subprotocols:     []string{"v4.channel.k8s.io"},
			Proxy:            http.ProxyFromEnvironment,
			HandshakeTimeout: 45 * time.Second,
			TLSClientConfig:  tlsConfig,
		},
		headers: headers,
	}, nil
}
