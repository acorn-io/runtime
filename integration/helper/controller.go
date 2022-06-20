package helper

import (
	"context"
	"net"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/controller"
	hclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/acorn/pkg/server"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/crds"
	"github.com/acorn-io/baaah/pkg/restconfig"
	"github.com/google/go-containerregistry/pkg/registry"
	uuid2 "github.com/google/uuid"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
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

func BuilderClient(t *testing.T, namespace string) client.Client {
	c, err := client.New(StartAPI(t), namespace)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func ensureNamespace(t *testing.T) {
	kclient := MustReturn(hclient.Default)
	_ = kclient.Create(context.Background(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: system.Namespace,
		},
	})
}
func StartAPI(t *testing.T) *rest.Config {
	apiStartLock.Lock()
	defer apiStartLock.Unlock()

	if apiStarted {
		return apiRESTConfig
	}

	ensureNamespace(t)

	l, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := server.New()
	srv.Options.SecureServing.Listener = l
	srv.Options.Authentication.TolerateInClusterLookupFailure = true
	cfg, err := srv.NewConfig("dev")
	if err == nil {
		go func() {
			err := srv.Run(context.Background(), cfg)
			t.Log("failed to start api", err)
		}()
	}

	kubeconfig := clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			"default": {
				Server:                "https://" + l.Addr().String(),
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
		EnsureCRDs(t)
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

	ensureNamespace(t)

	k8s, err := hclient.DefaultInterface()
	if err != nil {
		t.Fatal(err)
	}

	lock(context.Background(), k8s, func(ctx context.Context) {
		c, err := controller.New()
		if err != nil {
			t.Fatal(err)
		}

		if err := c.Start(context.Background()); err != nil {
			t.Fatal(err)
		}
	})

	EnsureCRDs(t)
	controllerStarted = true
}

func lock(ctx context.Context, client kubernetes.Interface, cb func(ctx context.Context)) {
	id := uuid2.New().String()
	rl, err := resourcelock.New(resourcelock.LeasesResourceLock,
		system.Namespace,
		"acorn-controller",
		client.CoreV1(),
		client.CoordinationV1(),
		resourcelock.ResourceLockConfig{
			Identity: id,
		})
	if err != nil {
		logrus.Fatalf("error creating leader lost for %s/%s id: %s", system.Namespace, "acorn-controller", id)
	}

	go leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:          rl,
		LeaseDuration: 3 * time.Second,
		RenewDeadline: 2 * time.Second,
		RetryPeriod:   time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				go cb(ctx)
			},
			OnStoppedLeading: func() {
			},
		},
		WatchDog:        nil,
		ReleaseOnCancel: true,
		Name:            "",
	})
}
