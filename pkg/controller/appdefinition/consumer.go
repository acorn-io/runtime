package appdefinition

import (
	"context"

	"github.com/acorn-io/baaah/pkg/typed"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/ref"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// augmentContainerWithConsumerInfo adds files and environment variables from any services this container depends on
// that expose consumer files and environment variables
func augmentContainerWithConsumerInfo(ctx context.Context, c kclient.Client, namespace string, container v1.Container) (v1.Container, error) {
	result := *container.DeepCopy()
	for _, dep := range container.Dependencies {
		// This shouldn't happen, but okay?
		if dep.TargetName == "" {
			continue
		}

		svc := &v1.ServiceInstance{}
		if err := ref.Lookup(ctx, c, svc, namespace, dep.TargetName); apierror.IsNotFound(err) {
			// We can ignore missing deps because the normal dep ordering will ensure that this container
			// can't be created/update until it's dependency is
			continue
		} else if err != nil {
			return result, err
		}

		if svc.Spec.Consumer == nil {
			continue
		}

		for _, fileName := range typed.SortedKeys(svc.Spec.Consumer.Files) {
			if _, ok := result.Files[fileName]; ok {
				continue
			}
			file := svc.Spec.Consumer.Files[fileName]
			if file.Secret.Name != "" {
				file.Secret.Name = dep.TargetName + "." + file.Secret.Name
			}

			if result.Files == nil {
				result.Files = map[string]v1.File{}
			}

			result.Files[fileName] = file
		}

	envLoop:
		for _, envVar := range svc.Spec.Consumer.Environment {
			if envVar.Name != "" {
				for _, existingEnvVar := range result.Environment {
					if existingEnvVar.Name == envVar.Name {
						continue envLoop
					}
				}
			}
			if envVar.Secret.Name != "" {
				envVar.Secret.Name = dep.TargetName + "." + envVar.Secret.Name
			}

			result.Environment = append(result.Environment, envVar)
		}
	}

	return result, nil
}
