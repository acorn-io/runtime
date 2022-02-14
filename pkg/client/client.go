package client

import (
	"github.com/ibuildthecloud/herd/pkg/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func MustDefault() client.WithWatch {
	c, err := Default()
	if err != nil {
		panic(err)
	}
	return c
}

func Default() (client.WithWatch, error) {
	cfg, err := config.GetConfig()
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
