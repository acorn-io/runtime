package helper

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/controller"
	"github.com/acorn-io/acorn/pkg/crds"
	hclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/acorn/pkg/server"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/restconfig"
	"github.com/google/go-containerregistry/pkg/registry"
	uuid2 "github.com/google/uuid"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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

const (
	APIServerLocalCertPath = "apiserver.local.config"
)

func EnsureCRDs(t *testing.T) {
	t.Helper()

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

func ClientAndNamespace(t *testing.T) (client.Client, *corev1.Namespace) {
	t.Helper()

	StartController(t)
	kclient := MustReturn(hclient.Default)
	ns := TempNamespace(t, kclient)
	return BuilderClient(t, ns.Name), ns
}

func BuilderClient(t *testing.T, namespace string) client.Client {
	t.Helper()

	StartController(t)

	c, err := client.New(StartAPI(t), namespace)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func ensureNamespace(t *testing.T) {
	t.Helper()

	kclient := MustReturn(hclient.Default)
	_ = kclient.Create(context.Background(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: system.Namespace,
		},
	})
	_ = kclient.Create(context.Background(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: system.ImagesNamespace,
		},
	})
}

func StartAPI(t *testing.T) *rest.Config {
	t.Helper()

	apiStartLock.Lock()
	defer apiStartLock.Unlock()

	if os.Getenv("TEST_ACORN_API_SERVER") == "external" {
		r, e := hclient.DefaultConfig()
		if e != nil {
			t.Fatal(e)
		}
		return r
	}

	if apiStarted {
		return apiRESTConfig
	}

	ensureNamespace(t)

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	srv := server.New()
	srv.Options.SecureServing.Listener = l
	srv.Options.Authentication.TolerateInClusterLookupFailure = true
	cfg, err := srv.NewConfig("dev")
	if err != nil {
		t.Fatal(err)
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

	if err == nil {
		go func() {
			cfg.LocalRestConfig = restConfig
			err := srv.Run(context.Background(), cfg)
			t.Log("failed to start api", err)
		}()
	}

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
	t.Helper()

	os.Setenv("ACORN_TEST_ALLOW_LOCALHOST_REGISTRY", "true")

	if os.Getenv("TEST_ACORN_API_SERVER") != "external" {
		srv := httptest.NewServer(registry.New())
		return srv.Listener.Addr().String(), srv.Close
	}

	ensureNamespace(t)

	return startTestRegistry(t, system.Namespace)
}

func StartController(t *testing.T) {
	t.Helper()

	if os.Getenv("TEST_ACORN_CONTROLLER") == "external" {
		return
	}
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

func int32Ptr(i int32) *int32 { return &i }
func startTestRegistry(t *testing.T, namespace string) (string, func()) {
	t.Helper()

	restConfig := StartAPI(t)
	k8sclient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		t.Fatal(err)
	}

	depSpec := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-registry",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "test-registry"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "test-registry"}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "test-registry",
						Image: "registry:2",
						Ports: []corev1.ContainerPort{{
							Name:          "http",
							Protocol:      corev1.ProtocolTCP,
							ContainerPort: 5000,
						}}}}}}}}

	dep, err := k8sclient.AppsV1().Deployments(namespace).Create(context.Background(), depSpec, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// Wait for the registry to become available before progressing
	var retries int
	for *dep.Spec.Replicas != dep.Status.ReadyReplicas {
		dep, err = k8sclient.AppsV1().Deployments(namespace).Get(context.Background(), dep.GetName(), metav1.GetOptions{})
		if err != nil {
			t.Fatal(err)
		}

		if retries > 6 {
			t.Fatal(errors.New("test registry failed to come up after 6 retries"))
		}
		time.Sleep(5 * time.Second)
		retries++
	}

	svcSpec := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-registry",
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "test-registry"},
			Type:     "ClusterIP",
			Ports: []corev1.ServicePort{{
				Port: 5000,
				TargetPort: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 5000,
				}}}}}

	svc, err := k8sclient.CoreV1().Services(namespace).Create(context.Background(), svcSpec, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}

	return fmt.Sprintf("%v.%v.svc.cluster.local:5000", svc.GetName(), namespace), func() {
		_ = k8sclient.CoreV1().Services(namespace).Delete(context.Background(), svc.GetName(), metav1.DeleteOptions{})
		_ = k8sclient.AppsV1().Deployments(namespace).Delete(context.Background(), dep.GetName(), metav1.DeleteOptions{})
	}
}
