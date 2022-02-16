package portforwarder

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

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

func Forward(ctx context.Context, cfg *rest.Config, pod *corev1.Pod, port uint32) (string, error) {
	return ForwardWithSPDY(ctx, cfg, pod, port)
}

func ForwardWithSPDY(ctx context.Context, cfg *rest.Config, pod *corev1.Pod, port uint32) (string, error) {
	url, err := urlForPodAndPort(cfg, pod, port)
	if err != nil {
		return "", err
	}

	ready := make(chan struct{}, 1)
	transport, upgrader, err := spdy.RoundTripperFor(cfg)
	if err != nil {
		return "", err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, url)
	fw, err := portforward.NewOnAddresses(dialer, []string{"127.0.0.1"}, []string{fmt.Sprintf(":%d", port)},
		ctx.Done(), ready, nil, nil)
	if err != nil {
		return "", err
	}

	go fw.ForwardPorts()
	<-ready

	ports, err := fw.GetPorts()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("tcp://127.0.0.1:%d", ports[0].Local), nil
}
