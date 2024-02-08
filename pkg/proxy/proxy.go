package proxy

import (
	"net/http"
	"net/url"
	"strings"

	"k8s.io/apimachinery/pkg/util/proxy"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport"
)

var (
	er = &errorResponder{}
)

type errorResponder struct {
}

func (e *errorResponder) Error(w http.ResponseWriter, _ *http.Request, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = w.Write([]byte(err.Error()))
}

// Handler is mostly copied from "kubectl proxy" code
func Handler(cfg *rest.Config) (http.Handler, error) {
	host := cfg.Host
	if !strings.HasSuffix(host, "/") {
		host = host + "/"
	}
	target, err := url.Parse(host)
	if err != nil {
		return nil, err
	}

	transport, err := rest.TransportFor(cfg)
	if err != nil {
		return nil, err
	}
	upgradeTransport, err := makeUpgradeTransport(cfg, transport)
	if err != nil {
		return nil, err
	}

	proxy := proxy.NewUpgradeAwareHandler(target, transport, false, false, er)
	proxy.UpgradeTransport = upgradeTransport
	proxy.UseRequestLocation = true
	proxy.UseLocationHost = true

	handler := http.Handler(proxy)

	if len(target.Path) > 1 {
		handler = prependPath(target.Path[:len(target.Path)-1], handler)
	}

	return proxyHeaders(handler), nil
}

func proxyHeaders(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		req.Header.Del("Authorization")
		if req.Header.Get("X-Forwarded-Proto") == "" && req.TLS != nil {
			req.Header.Set("X-Forwarded-Proto", "https")
		}
		handler.ServeHTTP(rw, req)
	})
}

func prependPath(prefix string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if len(req.URL.Path) > 1 {
			req.URL.Path = prefix + req.URL.Path
		} else {
			req.URL.Path = prefix
		}
		h.ServeHTTP(w, req)
	})
}

func makeUpgradeTransport(config *rest.Config, rt http.RoundTripper) (proxy.UpgradeRequestRoundTripper, error) {
	transportConfig, err := config.TransportConfig()
	if err != nil {
		return nil, err
	}

	upgrader, err := transport.HTTPWrappersForConfig(transportConfig, proxy.MirrorRequest)
	if err != nil {
		return nil, err
	}

	return proxy.NewUpgradeRequestRoundTripper(rt, upgrader), nil
}
