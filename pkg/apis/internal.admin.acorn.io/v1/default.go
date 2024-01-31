package v1

import (
	"context"
	"fmt"
	"sort"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getCurrentClusterComputeClassDefault(ctx context.Context, c client.Client) (*ClusterComputeClassInstance, error) {
	clusterComputeClasses := ClusterComputeClassInstanceList{}
	if err := c.List(ctx, &clusterComputeClasses, &client.ListOptions{}); err != nil {
		return nil, err
	}

	sort.Slice(clusterComputeClasses.Items, func(i, j int) bool {
		return clusterComputeClasses.Items[i].Name < clusterComputeClasses.Items[j].Name
	})

	var defaultCCC *ClusterComputeClassInstance
	for _, clusterComputeClass := range clusterComputeClasses.Items {
		if clusterComputeClass.Default {
			if defaultCCC != nil {
				return nil, fmt.Errorf(
					"cannot establish defaults because two default computeclasses exist: %v and %v",
					defaultCCC.Name, clusterComputeClass.Name)
			}
			t := clusterComputeClass // Create a new variable that isn't being iterated on to get a pointer
			defaultCCC = &t
		}
	}

	return defaultCCC, nil
}

func getCurrentProjectComputeClassDefault(ctx context.Context, c client.Client, namespace string) (*ProjectComputeClassInstance, error) {
	projectComputeClasses := ProjectComputeClassInstanceList{}
	if err := c.List(ctx, &projectComputeClasses, &client.ListOptions{Namespace: namespace}); err != nil {
		return nil, err
	}

	sort.Slice(projectComputeClasses.Items, func(i, j int) bool {
		return projectComputeClasses.Items[i].Name < projectComputeClasses.Items[j].Name
	})

	var defaultPCC *ProjectComputeClassInstance
	for _, projectComputeClass := range projectComputeClasses.Items {
		if projectComputeClass.Default {
			if defaultPCC != nil {
				return nil, fmt.Errorf(
					"cannot establish defaults because two default computeclasses exist: %v and %v",
					defaultPCC.Name, projectComputeClass.Name)
			}
			t := projectComputeClass // Create a new variable that isn't being iterated on to get a pointer
			defaultPCC = &t
		}
	}

	return defaultPCC, nil
}

func GetDefaultComputeClass(ctx context.Context, c client.Client, namespace string) (string, error) {
	pcc, err := getCurrentProjectComputeClassDefault(ctx, c, namespace)
	if err != nil {
		return "", err
	} else if pcc != nil {
		return pcc.Name, nil
	}

	ccc, err := getCurrentClusterComputeClassDefault(ctx, c)
	if err != nil {
		return "", err
	} else if ccc != nil {
		return ccc.Name, nil
	}
	return "", nil
}
