package imagesystem

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/k8schannel"
	"github.com/acorn-io/runtime/pkg/system"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewAPIBasedTransport(client kclient.Client, cfg *rest.Config) (http.RoundTripper, error) {
	if system.IsRunningAsPod() {
		return http.DefaultTransport, nil
	}

	cfg = rest.CopyConfig(cfg)
	cfg.APIPath = "/api"
	cfg.GroupVersion = &schema.GroupVersion{
		Group:   "",
		Version: "v1",
	}

	restClient, err := rest.RESTClientFor(cfg)
	if err != nil {
		return nil, err
	}

	k8sdialer, err := k8schannel.NewDialer(cfg, true)
	if err != nil {
		return nil, err
	}

	dialer := &dialer{
		client:    client,
		rest:      restClient,
		dialer:    &net.Dialer{},
		k8sdialer: k8sdialer,
	}

	newTransport := http.DefaultTransport.(*http.Transport).Clone()
	newTransport.DialContext = dialer.dial

	return newTransport, nil
}

type dialer struct {
	client    kclient.Client
	rest      *rest.RESTClient
	dialer    *net.Dialer
	k8sdialer *k8schannel.Dialer
}

func (d *dialer) dial(ctx context.Context, network, addr string) (net.Conn, error) {
	// quick check
	parts := strings.Split(addr, ".")
	if len(parts) < 3 || parts[1] != system.ImagesNamespace {
		return d.dialer.DialContext(ctx, network, addr)
	}

	cfg, err := config.Get(ctx, d.client)
	if err != nil {
		return nil, err
	}

	host, port, _ := net.SplitHostPort(addr)
	if !strings.HasSuffix(host, cfg.InternalClusterDomain) {
		return d.dialer.DialContext(ctx, network, addr)
	}

	serviceName, namespace := parts[0], parts[1]

	service := &corev1.Service{}
	if err := d.client.Get(ctx, router.Key(namespace, serviceName), service); err != nil {
		return nil, err
	}

	var targetPortRef intstr.IntOrString
	for _, servicePort := range service.Spec.Ports {
		if servicePort.Name == port || strconv.FormatInt(int64(servicePort.Port), 10) == port {
			targetPortRef = servicePort.TargetPort
			break
		}
	}

	endpoints := &corev1.Endpoints{}
	if err := d.client.Get(ctx, router.Key(namespace, serviceName), endpoints); err != nil {
		return nil, err
	}

	var podName string
	for _, endpoint := range endpoints.Subsets {
		for _, addr := range endpoint.Addresses {
			if addr.TargetRef != nil && addr.TargetRef.Kind == "Pod" {
				podName = addr.TargetRef.Name
				break
			}
		}
	}

	var portNum int32
	if targetPortRef.IntVal != 0 {
		portNum = targetPortRef.IntVal
	}

	if podName == "" || portNum == 0 {
		return nil, fmt.Errorf("failed to find target for %s", addr)
	}

	url := URLForPortAndPod(d.rest, namespace, podName, portNum).String()
	conn, err := d.k8sdialer.DialContext(ctx, url, nil)
	if err != nil {
		return nil, err
	}

	return conn.ForStream(0), nil
}

func URLForPortAndPod(restClient rest.Interface, namespace, name string, port int32) *url.URL {
	url := restClient.Get().
		Resource("pods").
		Namespace(namespace).
		Name(name).
		SubResource("portforward").
		URL()
	q := url.Query()
	q.Set("ports", strconv.Itoa(int(port)))
	url.RawQuery = q.Encode()
	return url
}
