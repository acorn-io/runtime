package lasso

import (
	"time"

	controllerruntime "github.com/rancher/lasso/controller-runtime"
	lcache "github.com/rancher/lasso/pkg/cache"
	lclient "github.com/rancher/lasso/pkg/client"
	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/lasso/pkg/dynamic"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Runtime struct {
	Backend *Backend
}

func NewRuntime(cfg *rest.Config, scheme *runtime.Scheme) (*Runtime, error) {
	return NewRuntimeForNamespace(cfg, "", scheme)
}

func NewRuntimeForNamespace(cfg *rest.Config, namespace string, scheme *runtime.Scheme) (*Runtime, error) {
	cf, err := lclient.NewSharedClientFactory(cfg, &lclient.SharedClientFactoryOptions{
		Scheme: scheme,
	})
	if err != nil {
		return nil, err
	}

	cacheFactory := lcache.NewSharedCachedFactory(cf, &lcache.SharedCacheFactoryOptions{
		DefaultNamespace: namespace,
	})

	factory := controller.NewSharedControllerFactory(cacheFactory, &controller.SharedControllerFactoryOptions{
		DefaultRateLimiter: workqueue.NewItemFastSlowRateLimiter(500*time.Millisecond, time.Second, 2),
	})
	if err != nil {
		return nil, err
	}

	restClient, err := rest.UnversionedRESTClientFor(cfg)
	if err != nil {
		return nil, err
	}

	dc := discovery.NewDiscoveryClient(restClient)
	cache, err := controllerruntime.NewNewCacheFunc(factory.SharedCacheFactory(), dynamic.New(dc))(cfg, cache.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, err
	}

	uncachedClient, err := client.New(cfg, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, err
	}

	cachedClient, err := client.NewDelegatingClient(client.NewDelegatingClientInput{
		CacheReader: cache,
		Client:      uncachedClient,
	})
	if err != nil {
		return nil, err
	}

	return &Runtime{
		Backend: newBackend(factory, newCacheClient(uncachedClient, cachedClient), cache),
	}, nil
}
