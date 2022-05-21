package helper

import (
	"context"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/controller"
	hclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/acorn/pkg/server"
	"github.com/acorn-io/baaah/pkg/crds"
	"github.com/acorn-io/baaah/pkg/restconfig"
	"github.com/google/go-containerregistry/pkg/registry"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

var (
	controllerStarted   = false
	controllerStartLock sync.Mutex
	apiStarted          = false
	apiStartLock        sync.Mutex
	apiRESTConfig       *rest.Config
)

func EnsureCRDs(t *testing.T) {
	ctx := GetCTX(t)
	if err := crds.Create(ctx, scheme.Scheme, v1.SchemeGroupVersion); err != nil {
		t.Fatal(err)
	}
	c, err := hclient.Default()
	if err != nil {
		t.Fatal(err)
	}

	var apps v1.AppInstanceList
	for {
		if err := c.List(ctx, &apps); err != nil {
			time.Sleep(time.Second)
		} else {
			break
		}
	}
}

func StartAPI(t *testing.T) *rest.Config {
	apiStartLock.Lock()
	defer apiStartLock.Unlock()

	if apiStarted {
		return apiRESTConfig
	}

	srv := server.New()
	srv.Options.SecureServing.BindPort = 37443
	srv.Options.Authentication.TolerateInClusterLookupFailure = true
	cfg, err := srv.NewConfig("dev")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		err := srv.Run(context.Background(), cfg)
		t.Log("failed to start api", err)
	}()

	kubeconfig := clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			"default": {
				Server:                "https://localhost:37443",
				InsecureSkipTLSVerify: true,
			},
		},
		AuthInfos: nil,
		Contexts: map[string]*clientcmdapi.Context{
			"default": {
				Cluster: "default",
			},
		},
		CurrentContext: "default",
	}
	restConfig, err := clientcmd.NewDefaultClientConfig(kubeconfig, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		t.Fatal(err)
	}

	restConfig = restconfig.SetScheme(restConfig, scheme.Scheme)
	restConfig.GroupVersion = &apiv1.SchemeGroupVersion
	restClient, err := rest.RESTClientFor(restConfig)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 100; i++ {
		_, err := restClient.Get().AbsPath("/readyz/ping").DoRaw(context.Background())
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		apiRESTConfig = restConfig
		apiStarted = true
		return restConfig
	}

	t.Fatal("failed to start API")
	return nil
}

func StartRegistry(t *testing.T) (string, func()) {
	srv := httptest.NewServer(registry.New())
	return srv.Listener.Addr().String(), srv.Close
}

func StartController(t *testing.T) {
	controllerStartLock.Lock()
	defer controllerStartLock.Unlock()

	if controllerStarted {
		return
	}

	c, err := controller.New()
	if err != nil {
		t.Fatal(err)
	}

	if err := c.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	controllerStarted = true
}
