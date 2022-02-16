package portforwarder

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
)

type WebSocketDialer struct {
	dialer  *websocket.Dialer
	headers http.Header
	url     string
}

func (w *WebSocketDialer) DialContext(ctx context.Context, address string) (net.Conn, error) {
	conn, _, err := w.dialer.DialContext(ctx, w.url, w.headers)
	if err != nil {
		return nil, err
	}
	return &connection{conn: conn}, nil
}

type connection struct {
	conn           *websocket.Conn
	initialized    bool
	errInitialized bool
	buf            []byte
}

func (c *connection) Read(b []byte) (n int, err error) {
	if len(c.buf) > 0 {
		n := copy(b, c.buf)
		c.buf = c.buf[n:]
		return n, nil
	}

	_, data, err := c.conn.ReadMessage()
	if err != nil {
		return 0, err
	}

	if data[0] == 1 {
		if !c.errInitialized {
			c.errInitialized = true
			return c.Read(b)
		}
		return 0, fmt.Errorf("ERROR CHANNEL: %s", string(data[1:]))
	}

	if data[0] != 0 {
		return c.Read(b)
	}

	if !c.initialized {
		c.initialized = true
		return c.Read(b)
	}

	data = data[1:]
	n = copy(b, data)
	c.buf = data[n:]
	return n, nil
}

func (c *connection) Write(b []byte) (n int, err error) {
	if len(b) == 0 {
		return 0, nil
	}

	// k8s doesn't seem like frames more that 1k, which seems
	// inefficient, but who knows, I just work here.
	if len(b) > 1024 {
		n, err := c.Write(b[:1024])
		if err != nil {
			return n, err
		}
		n2, err := c.Write(b[1024:])
		return n + n2, err
	}
	m, err := c.conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return 0, err
	}
	if _, err := m.Write([]byte{0}); err != nil {
		return 0, err
	}
	n, err = m.Write(b)
	if err != nil {
		return 0, err
	}

	return n, m.Close()
}

func (c *connection) Close() (err error) {
	return c.conn.Close()
}

func (c *connection) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *connection) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *connection) SetDeadline(t time.Time) error {
	c.SetWriteDeadline(t)
	return c.SetReadDeadline(t)
}

func (c *connection) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *connection) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
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

func NewWebSocketDialer(cfg *rest.Config, pod *corev1.Pod, port uint32) (*WebSocketDialer, error) {
	url, err := urlForPodAndPort(cfg, pod, port)
	if err != nil {
		return nil, err
	}

	tlsConfig, err := rest.TLSConfigFor(cfg)
	if err != nil {
		return nil, err
	}

	headers, err := GetHeadersFor(cfg)
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
		dialer: &websocket.Dialer{
			Subprotocols:     []string{"v4.channel.k8s.io"},
			Proxy:            http.ProxyFromEnvironment,
			HandshakeTimeout: 45 * time.Second,
			TLSClientConfig:  tlsConfig,
		},
		headers: headers,
		url:     newURL.String(),
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
