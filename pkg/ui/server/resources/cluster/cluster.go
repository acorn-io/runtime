package cluster

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	ui_acorn_io "github.com/acorn-io/acorn/pkg/apis/ui.acorn.io"
	uiv1 "github.com/acorn-io/acorn/pkg/apis/ui.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/labels"
	detector "github.com/rancher/kubernetes-provider-detector"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
)

var (
	clusterList     []uiv1.Cluster
	ignoreNamespace = map[string]bool{
		"kube-node-lease": true,
		"kube-public":     true,
	}
)

func GetCluster(ctx context.Context, name string) (*uiv1.Cluster, error) {
	clusters, err := ListClusters(ctx)
	if err != nil {
		return nil, err
	}
	for _, cluster := range clusters {
		if name == "_" && cluster.Spec.Default {
			return &cluster, nil
		}
		if cluster.Name == name {
			return &cluster, nil
		}
	}
	return nil, apierror.NewNotFound(schema.GroupResource{
		Group:    ui_acorn_io.Group,
		Resource: "clusters",
	}, name)
}

func GetConfig(name string) (*ClusterConfig, error) {
	configs, err := clusters(os.Getenv("KUBECONFIG"))
	if err != nil {
		return nil, err
	}
	for _, config := range configs {
		if name == "_" && config.Default {
			return &config, nil
		}
		if config.Name == name {
			return &config, nil
		}
	}
	return nil, apierror.NewNotFound(schema.GroupResource{
		Group:    ui_acorn_io.Group,
		Resource: "clusters",
	}, name)
}

func ListClusters(ctx context.Context) (result []uiv1.Cluster, _ error) {
	if len(clusterList) == 0 {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		return listClusters(ctx)
	}
	return clusterList, nil
}

func startPolling(ctx context.Context) {
	for {
		clusters, err := listClusters(ctx)
		if err == nil {
			clusterList = clusters
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}
}

func listClusters(ctx context.Context) (result []uiv1.Cluster, _ error) {
	var (
		resultLock  sync.Mutex
		contextName = os.Getenv("CONTEXT")
	)

	configs, err := clusters(os.Getenv("KUBECONFIG"))
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
		version := make(chan error, 1)
		go func() {
			_, err := k8s.Discovery().ServerVersion()
			version <- err
		}()
		select {
		case <-ctx.Done():
			result.Status.Error = fmt.Sprint(ctx.Err())
		case err := <-version:
			if err != nil && result.Status.Error == "" {
				result.Status.Error = err.Error()
			} else if err == nil {
				result.Status.Available = true
			}
		}
	}

	if result.Status.Available {
		provider, err := detector.DetectProvider(ctx, k8s)
		if err == nil {
			result.Status.Provider = provider
		}

		nses, err := k8s.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
		if err == nil {
			for _, ns := range nses.Items {
				if ns.Labels[labels.AcornManaged] == "true" ||
					strings.HasSuffix(ns.Name, "-system") ||
					ignoreNamespace[ns.Name] {
					continue
				}
				result.Status.Namespaces = append(result.Status.Namespaces, ns.Name)
			}
			sort.Strings(result.Status.Namespaces)
		}
	}

	return result
}
