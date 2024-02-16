package v1

import (
	"context"
	"fmt"
	"sort"

	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/z"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// GetDefaultComputeClassName gets the name of the effective default ComputeClass for a given project namespace.
// The precedence for picking the default ComputeClass is as follows:
//  1. Any ProjectComputeClassInstance (in the project namespace) or ClusterComputeClassInstance that is specified by the
//     ProjectInstance's DefaultComputeClass field
//  2. The ProjectComputeClassInstance (in the project namespace) with a Default field set to true
//  3. The ClusterComputeClassInstance with a Default field set to true
//
// If no default ComputeClass is found, an empty string is returned.
// If the project specifies a default compute class that doesn't exist, an error is returned.
func GetDefaultComputeClassName(ctx context.Context, c kclient.Client, namespace string) (string, error) {
	var project internalv1.ProjectInstance
	if err := c.Get(ctx, kclient.ObjectKey{Name: namespace}, &project); err != nil {
		return "", fmt.Errorf("failed to get project instance to determine default compute class: %w", err)
	}

	if projectSpecified := project.Status.DefaultComputeClass; projectSpecified != "" {
		if err := lookupComputeClass(ctx, c, namespace, projectSpecified); err != nil {
			if apierrors.IsNotFound(err) {
				return "", fmt.Errorf("project specified default compute class [%v] does not exist: %w", projectSpecified, err)
			}
			return "", fmt.Errorf("failed to get project specified default compute class [%v]: %w", projectSpecified, err)
		}

		return projectSpecified, nil
	}

	if pcc, err := getCurrentProjectComputeClassDefault(ctx, c, namespace); err != nil {
		return "", err
	} else if pcc != nil && pcc.Name != "" {
		return pcc.Name, nil
	}

	if ccc, err := getCurrentClusterComputeClassDefault(ctx, c); err != nil {
		return "", err
	} else if ccc != nil && ccc.Name != "" {
		return ccc.Name, nil
	}

	return "", nil
}

func getCurrentClusterComputeClassDefault(ctx context.Context, c kclient.Client) (*ClusterComputeClassInstance, error) {
	var clusterComputeClasses ClusterComputeClassInstanceList
	if err := c.List(ctx, &clusterComputeClasses, &kclient.ListOptions{}); err != nil {
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

			defaultCCC = z.Pointer(clusterComputeClass)
		}
	}

	return defaultCCC, nil
}

func getCurrentProjectComputeClassDefault(ctx context.Context, c kclient.Client, namespace string) (*ProjectComputeClassInstance, error) {
	var projectComputeClasses ProjectComputeClassInstanceList
	if err := c.List(ctx, &projectComputeClasses, &kclient.ListOptions{Namespace: namespace}); err != nil {
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

			defaultPCC = z.Pointer(projectComputeClass)
		}
	}

	return defaultPCC, nil
}

func lookupComputeClass(ctx context.Context, c kclient.Client, namespace, name string) error {
	if err := c.Get(ctx, kclient.ObjectKey{Namespace: namespace, Name: name},
		new(ProjectComputeClassInstance)); kclient.IgnoreNotFound(err) != nil {
		return err
	}

	return c.Get(ctx, kclient.ObjectKey{Namespace: "", Name: name}, new(ClusterComputeClassInstance))
}
