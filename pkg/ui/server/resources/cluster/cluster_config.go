package cluster

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/rancher/wrangler/pkg/ratelimit"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type ClusterConfig struct {
	Name   string
	Server string
	Config *rest.Config
	Error  error
}

func toClusterConfig(rawConfig clientcmdapi.Config, contextName string, context *clientcmdapi.Context) ClusterConfig {
	cluster := ClusterConfig{
		Name: contextName,
	}

	rawCluster := rawConfig.Clusters[context.Cluster]
	if rawCluster != nil {
		cluster.Server = rawCluster.Server
	}

	cfg, err := clientcmd.NewDefaultClientConfig(rawConfig, &clientcmd.ConfigOverrides{
		CurrentContext: contextName,
	}).ClientConfig()
	if err == nil {
		cfg.RateLimiter = ratelimit.None
		cluster.Config = cfg
	} else {
		cluster.Error = err
	}

	return cluster
}

func clusters(kubeconfig, name string) (result []ClusterConfig, _ error) {
	rawConfig, err := rawConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	for k, v := range rawConfig.Contexts {
		if name == "" || k == name || (name == "_" && rawConfig.CurrentContext == k) {
			result = append(result, toClusterConfig(rawConfig, k, v))
		}
	}

	return
}

func rawConfig(kubeconfig string) (clientcmdapi.Config, error) {
	if len(kubeconfig) > 0 {
		return loadConfigWithContext(&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig}, "").RawConfig()
	}

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if _, ok := os.LookupEnv("HOME"); !ok {
		u, err := user.Current()
		if err != nil {
			return clientcmdapi.Config{}, fmt.Errorf("could not get current user: %v", err)
		}
		loadingRules.Precedence = append(loadingRules.Precedence, filepath.Join(u.HomeDir, clientcmd.RecommendedHomeDir, clientcmd.RecommendedFileName))
	}

	return loadConfigWithContext(loadingRules, "").RawConfig()
}

func loadConfigWithContext(loader clientcmd.ClientConfigLoader, context string) clientcmd.ClientConfig {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loader,
		&clientcmd.ConfigOverrides{
			CurrentContext: context,
		})
}
