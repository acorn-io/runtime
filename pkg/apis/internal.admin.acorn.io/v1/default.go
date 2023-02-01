package v1

import (
	"context"
	"fmt"
	"sort"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getCurrentClusterWorkloadClassDefault(ctx context.Context, c client.Client) (*ClusterWorkloadClassInstance, error) {
	clusterWorkloadClasses := ClusterWorkloadClassInstanceList{}
	if err := c.List(ctx, &clusterWorkloadClasses, &client.ListOptions{}); err != nil {
		return nil, err
	}

	sort.Slice(clusterWorkloadClasses.Items, func(i, j int) bool {
		return clusterWorkloadClasses.Items[i].Name < clusterWorkloadClasses.Items[j].Name
	})

	var defaultCWC *ClusterWorkloadClassInstance
	for _, clusterWorkloadClass := range clusterWorkloadClasses.Items {
		if clusterWorkloadClass.Default {
			if defaultCWC != nil {
				return nil, fmt.Errorf(
					"cannot establish defaults because two default workloadclasses exist: %v and %v",
					defaultCWC.Name, clusterWorkloadClass.Name)
			}
			t := clusterWorkloadClass // Create a new variable that isn't being iterated on to get a pointer
			defaultCWC = &t
		}
	}

	return defaultCWC, nil
}

func getCurrentProjectWorkloadClassDefault(ctx context.Context, c client.Client, namespace string) (*ProjectWorkloadClassInstance, error) {
	projectWorkloadClasses := ProjectWorkloadClassInstanceList{}
	if err := c.List(ctx, &projectWorkloadClasses, &client.ListOptions{Namespace: namespace}); err != nil {
		return nil, err
	}

	sort.Slice(projectWorkloadClasses.Items, func(i, j int) bool {
		return projectWorkloadClasses.Items[i].Name < projectWorkloadClasses.Items[j].Name
	})

	var defaultPWC *ProjectWorkloadClassInstance
	for _, projectWorkloadClass := range projectWorkloadClasses.Items {
		if projectWorkloadClass.Default {
			if defaultPWC != nil {
				return nil, fmt.Errorf(
					"cannot establish defaults because two default workloadclasses exist: %v and %v",
					defaultPWC.Name, projectWorkloadClass.Name)
			}
			t := projectWorkloadClass // Create a new variable that isn't being iterated on to get a pointer
			defaultPWC = &t
		}
	}

	return defaultPWC, nil
}

func GetDefaultWorkloadClass(ctx context.Context, c client.Client, namespace string) (string, error) {
	pwc, err := getCurrentProjectWorkloadClassDefault(ctx, c, namespace)
	if err != nil {
		return "", err
	} else if pwc != nil {
		return pwc.Name, nil
	}

	cwc, err := getCurrentClusterWorkloadClassDefault(ctx, c)
	if err != nil {
		return "", err
	} else if cwc != nil {
		return cwc.Name, nil
	}
	return "", nil
}
