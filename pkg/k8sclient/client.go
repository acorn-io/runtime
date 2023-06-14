package k8sclient

import (
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/restconfig"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ObjectKey = client.ObjectKey
type ListOptions = client.ListOptions

func Default() (client.WithWatch, error) {
	cfg, err := restconfig.New(scheme.Scheme)
	if err != nil {
		return nil, err
	}

	return New(cfg)
}

func DefaultConfig() (*rest.Config, error) {
	return restconfig.New(scheme.Scheme)
}

func DefaultInterface() (kubernetes.Interface, error) {
	cfg, err := restconfig.New(scheme.Scheme)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(cfg)
}

func New(cfg *rest.Config) (client.WithWatch, error) {
	return client.NewWithWatch(cfg, client.Options{
		Scheme: scheme.Scheme,
	})
}
