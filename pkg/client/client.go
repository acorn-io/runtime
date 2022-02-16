package client

import (
	"github.com/ibuildthecloud/baaah/pkg/restconfig"
	"github.com/ibuildthecloud/herd/pkg/scheme"
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

func New(cfg *rest.Config) (client.WithWatch, error) {
	return client.NewWithWatch(cfg, client.Options{
		Scheme: scheme.Scheme,
	})
}
