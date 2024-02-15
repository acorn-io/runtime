package v1

import (
	"context"
	"fmt"
	"sort"

	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/z"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getCurrentClusterComputeClassDefault(ctx context.Context, c client.Client, projectSpecified string) (*ClusterComputeClassInstance, error) {
	var clusterComputeClasses ClusterComputeClassInstanceList
	if err := c.List(ctx, &clusterComputeClasses, &client.ListOptions{}); err != nil {
		return nil, err
	}

	sort.Slice(clusterComputeClasses.Items, func(i, j int) bool {
		return clusterComputeClasses.Items[i].Name < clusterComputeClasses.Items[j].Name
	})

	var defaultCCC, projectSpecifiedCCC *ClusterComputeClassInstance
	for _, clusterComputeClass := range clusterComputeClasses.Items {
		if clusterComputeClass.Default {
			if defaultCCC != nil {
				return nil, fmt.Errorf(
					"cannot establish defaults because two default computeclasses exist: %v and %v",
					defaultCCC.Name, clusterComputeClass.Name)
			}

			// Create a new variable that isn't being iterated on to get a pointer
			defaultCCC = z.Pointer(clusterComputeClass)
		}

		if projectSpecified != "" && clusterComputeClass.Name == projectSpecified {
			projectSpecifiedCCC = z.Pointer(clusterComputeClass)
		}
	}

	if projectSpecifiedCCC != nil {
		return projectSpecifiedCCC, nil
	}

	return defaultCCC, nil
}

func getCurrentProjectComputeClassDefault(ctx context.Context, c client.Client, projectSpecified, namespace string) (*ProjectComputeClassInstance, error) {
	var projectComputeClasses ProjectComputeClassInstanceList
	if err := c.List(ctx, &projectComputeClasses, &client.ListOptions{Namespace: namespace}); err != nil {
		return nil, err
	}

	sort.Slice(projectComputeClasses.Items, func(i, j int) bool {
		return projectComputeClasses.Items[i].Name < projectComputeClasses.Items[j].Name
	})

	var defaultPCC, projectSpecifiedPCC *ProjectComputeClassInstance
	for _, projectComputeClass := range projectComputeClasses.Items {
		if projectComputeClass.Default {
			if defaultPCC != nil {
				return nil, fmt.Errorf(
					"cannot establish defaults because two default computeclasses exist: %v and %v",
					defaultPCC.Name, projectComputeClass.Name)
			}

			// Create a new variable that isn't being iterated on to get a pointer
			defaultPCC = z.Pointer(projectComputeClass)
		}

		if projectSpecified != "" && projectComputeClass.Name == projectSpecified {
			projectSpecifiedPCC = z.Pointer(projectComputeClass)
		}
	}

	if projectSpecifiedPCC != nil {
		return projectSpecifiedPCC, nil
	}

	return defaultPCC, nil
}

// GetDefaultComputeClassName gets the name of the effective default ComputeClass for a given project namespace.
// The precedence for picking the default ComputeClass is as follows:
//  1. Any ProjectComputeClassInstance (in the project namespace) or ClusterComputeClassInstance that is specified by the
//     ProjectInstance's DefaultComputeClass field
//  2. The ProjectComputeClassInstance (in the project namespace) with a Default field set to true
//  3. The ClusterComputeClassInstance with a Default field set to true
//
// If no default ComputeClass is found, an empty string is returned.
func GetDefaultComputeClassName(ctx context.Context, c client.Client, namespace string) (string, error) {
	var project internalv1.ProjectInstance
	if err := c.Get(ctx, client.ObjectKey{Name: namespace}, &project); err != nil {
		return "", fmt.Errorf("failed to get projectinstance to determine default compute class: %w", err)
	}
	projectSpecified := project.Status.DefaultComputeClass

	var defaultComputeClasses []string
	pcc, err := getCurrentProjectComputeClassDefault(ctx, c, projectSpecified, namespace)
	if err != nil {
		return "", err
	}
	if pcc != nil {
		defaultComputeClasses = append(defaultComputeClasses, pcc.Name)
	}

	ccc, err := getCurrentClusterComputeClassDefault(ctx, c, projectSpecified)
	if err != nil {
		return "", err
	}
	if ccc != nil {
		defaultComputeClasses = append(defaultComputeClasses, ccc.Name)
	}

	if sets.New(defaultComputeClasses...).Has(projectSpecified) {
		return projectSpecified, nil
	}

	if len(defaultComputeClasses) > 0 {
		return defaultComputeClasses[0], nil
	}

	return "", nil
}
