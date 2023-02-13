package restconfig

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/acorn-io/baaah/pkg/ratelimit"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func Default() (*rest.Config, error) {
	return New(scheme.Scheme)
}

func ClientConfigFromFile(file, context string) clientcmd.ClientConfig {
	loader := clientcmd.NewDefaultClientConfigLoadingRules()
	loader.ExplicitPath = file
	cfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loader,
		&clientcmd.ConfigOverrides{
			CurrentContext: context,
		})
	fmt.Println(os.Environ())
	raw, err := cfg.RawConfig()
	if err != nil {
		fmt.Println("err!: ", err)
	} else {
		x, _ := json.Marshal(raw)
		fmt.Println("raw: " + string(x))
	}
	return cfg
}

func FromFile(file, context string) (*rest.Config, error) {
	return ClientConfigFromFile(file, context).ClientConfig()
}

func SetScheme(cfg *rest.Config, scheme *runtime.Scheme) *rest.Config {
	cfg.NegotiatedSerializer = serializer.NewCodecFactory(scheme)
	cfg.UserAgent = rest.DefaultKubernetesUserAgent()
	return cfg
}

func New(scheme *runtime.Scheme) (*rest.Config, error) {
	cfg, err := config.GetConfigWithContext(os.Getenv("CONTEXT"))
	if err != nil {
		return nil, err
	}
	cfg.RateLimiter = ratelimit.None
	return SetScheme(cfg, scheme), nil
}
