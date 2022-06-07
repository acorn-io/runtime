package cluster

import (
	"context"
	"os"
	"sort"
	"sync"
	"time"

	ui_acorn_io "github.com/acorn-io/acorn/pkg/apis/ui.acorn.io"
	uiv1 "github.com/acorn-io/acorn/pkg/apis/ui.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	detector "github.com/rancher/kubernetes-provider-detector"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
)

func GetCluster(ctx context.Context, name string) (*uiv1.Cluster, error) {
	clusters, err := listClusters(ctx, name)
	if err != nil {
		return nil, err
	}
	if len(clusters) == 0 {
		return nil, apierror.NewNotFound(schema.GroupResource{
			Group:    ui_acorn_io.Group,
			Resource: "clusters",
		}, name)
	}
	return &clusters[0], nil
}

func GetConfig(name string) (*ClusterConfig, error) {
	configs, err := clusters(os.Getenv("KUBECONFIG"), name)
	if err != nil {
		return nil, err
	}
	if len(configs) == 0 {
		return nil, apierror.NewNotFound(schema.GroupResource{
			Group:    ui_acorn_io.Group,
			Resource: "clusters",
		}, name)
	}
	return &configs[0], nil
}

func ListClusters(ctx context.Context) (result []uiv1.Cluster, _ error) {
	return listClusters(ctx, "")
}

func listClusters(ctx context.Context, name string) (result []uiv1.Cluster, _ error) {
	var (
		resultLock  sync.Mutex
		contextName = os.Getenv("CONTEXT")
	)

	configs, err := clusters(os.Getenv("KUBECONFIG"), name)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	wg := sync.WaitGroup{}
	for _, config := range configs {
		wg.Add(1)
		go func(config ClusterConfig) {
			defer wg.Done()
			cluster := toCluster(ctx, config)
			resultLock.Lock()
			defer resultLock.Unlock()
			result = append(result, cluster)
		}(config)
	}

	wg.Wait()

	sort.Slice(result, func(i, j int) bool {
		if result[i].Name == contextName {
			return true
		}
		if result[j].Name == contextName {
			return false
		}
		return result[i].Name < result[j].Name
	})

	return result, nil
}

func toCluster(ctx context.Context, config ClusterConfig) uiv1.Cluster {
	result := uiv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: config.Name,
		},
		Spec: uiv1.ClusterSpec{
			Address: config.Server,
		},
	}

	if config.Error != nil {
		result.Status.Error = config.Error.Error()
		return result
	}

	client, err := client.New(config.Config, "")
	if err != nil {
		result.Status.Error = err.Error()
		return result
	}

	info, err := client.Info(ctx)
	if err == nil {
		result.Status.Available = true
		result.Status.Installed = true
		result.Status.Info = &info.Spec
	}

	k8s, err := kubernetes.NewForConfig(config.Config)
	if err != nil {
		result.Status.Error = err.Error()
		return result
	}

	if !result.Status.Available {
		_, err := k8s.Discovery().ServerVersion()
		if err != nil && result.Status.Error == "" {
			result.Status.Error = err.Error()
		} else if err == nil {
			result.Status.Available = true
		}
	}

	if result.Status.Available {
		provider, err := detector.DetectProvider(ctx, k8s)
		if err == nil {
			result.Status.Provider = provider
		}
	}

	return result
}
