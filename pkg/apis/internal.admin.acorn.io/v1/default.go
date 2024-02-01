package v1

import (
	"context"
	"fmt"
	"sort"

	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/z"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getCurrentClusterComputeClassDefault(ctx context.Context, c client.Client, projectDefaultComputeClass string) (*ClusterComputeClassInstance, error) {
	clusterComputeClasses := ClusterComputeClassInstanceList{}
	if err := c.List(ctx, &clusterComputeClasses, &client.ListOptions{}); err != nil {
		return nil, err
	}

	sort.Slice(clusterComputeClasses.Items, func(i, j int) bool {
		return clusterComputeClasses.Items[i].Name < clusterComputeClasses.Items[j].Name
	})

	var defaultCCC, projectDefaultCCC *ClusterComputeClassInstance
	for _, clusterComputeClass := range clusterComputeClasses.Items {
		if clusterComputeClass.Default {
			if defaultCCC != nil {
				return nil, fmt.Errorf(
					"cannot establish defaults because two default computeclasses exist: %v and %v",
					defaultCCC.Name, clusterComputeClass.Name)
			}

			// Create a new variable that isn't being iterated on to get a pointer
			if projectDefaultComputeClass != "" {
				defaultCCC = z.Pointer(clusterComputeClass)
			}
		}

		if clusterComputeClass.Name == projectDefaultComputeClass {
			projectDefaultCCC = z.Pointer(clusterComputeClass)
		}
	}

	if projectDefaultCCC != nil {
		return projectDefaultCCC, nil
	}

	return defaultCCC, nil
}

func getCurrentProjectComputeClassDefault(ctx context.Context, c client.Client, projectDefaultComputeClass, namespace string) (*ProjectComputeClassInstance, error) {
	projectComputeClasses := ProjectComputeClassInstanceList{}
	if err := c.List(ctx, &projectComputeClasses, &client.ListOptions{Namespace: namespace}); err != nil {
		return nil, err
	}

	sort.Slice(projectComputeClasses.Items, func(i, j int) bool {
		return projectComputeClasses.Items[i].Name < projectComputeClasses.Items[j].Name
	})

	var defaultPCC, projectDefaultPCC *ProjectComputeClassInstance
	for _, projectComputeClass := range projectComputeClasses.Items {
		if projectComputeClass.Default {
			if defaultPCC != nil {
				return nil, fmt.Errorf(
					"cannot establish defaults because two default computeclasses exist: %v and %v",
					defaultPCC.Name, projectComputeClass.Name)
			}

			// Create a new variable that isn't being iterated on to get a pointer
			if projectDefaultComputeClass != "" {
				defaultPCC = z.Pointer(projectComputeClass)
			}
		}

		if projectComputeClass.Name == projectDefaultComputeClass {
			projectDefaultPCC = z.Pointer(projectComputeClass)
		}
	}

	if projectDefaultPCC != nil {
		return projectDefaultPCC, nil
	}

	return defaultPCC, nil
}

func GetDefaultComputeClass(ctx context.Context, c client.Client, namespace string) (string, error) {
	var project internalv1.ProjectInstance
	if err := c.Get(ctx, client.ObjectKey{Name: namespace}, &project); err != nil {
		return "", fmt.Errorf("failed to get projectinstance to determine default compute class: %w", err)
	}
	projectDefault := project.Status.DefaultComputeClass

	pcc, err := getCurrentProjectComputeClassDefault(ctx, c, projectDefault, namespace)
	if err != nil {
		return "", err
	} else if pcc != nil {
		return pcc.Name, nil
	}

	ccc, err := getCurrentClusterComputeClassDefault(ctx, c, projectDefault)
	if err != nil {
		return "", err
	} else if ccc != nil {
		return ccc.Name, nil
	}
	return "", nil
}
