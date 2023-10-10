package appdefinition

import (
	"context"

	"github.com/acorn-io/baaah/pkg/typed"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/imagerules"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/ref"
	"github.com/sirupsen/logrus"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func checkConsumerPermsAuthz(ctx context.Context, c kclient.Client, appInstance *v1.AppInstance, containerName string, consumedPerms v1.Permissions) (deniedConsumerPerms v1.Permissions, _ error) {
	imageName := appInstance.Status.AppImage.Name

	deniedConsumerPerms = v1.Permissions{
		ServiceName: containerName,
	}

	// I wonder if this is OK here, since at this point, there could already be a new image staged
	// and thus the annotation would already be updated and may not match the image that we use here.
	// That should be a very rare case though and has to have a very good timing.
	// Anyway, this couldn't be used for privilege escalation, since we're still using the current digest.
	if oi, ok := appInstance.Annotations[labels.AcornOriginalImage]; ok {
		imageName = oi
	}

	authzPerms, err := imagerules.GetAuthorizedPermissions(ctx, c, appInstance.Status.Namespace, imageName, appInstance.Status.AppImage.Digest)
	if err != nil {
		return deniedConsumerPerms, err
	}

	// Need to deepcopy here since otherwise we'd override the name in the original object which we still need
	copyWithName := func(perms []v1.Permissions, name string) []v1.Permissions {
		nperms := make([]v1.Permissions, len(perms))
		for i := range perms {
			nperms[i] = perms[i].DeepCopy().Get()
			nperms[i].ServiceName = name
		}
		return nperms
	}

	tmp := v1.Permissions{
		ServiceName: containerName,
		Rules:       consumedPerms.GetRules(),
	}

	if d, granted := v1.GrantsAll(appInstance.Status.Namespace, []v1.Permissions{tmp}, copyWithName(authzPerms, containerName)); !granted {
		for _, p := range d { // should really only be one item anyway
			deniedConsumerPerms.Rules = append(deniedConsumerPerms.Rules, p.GetRules()...)
		}
	}

	return
}

// getConsumerPermissions returns the permissions for a given container augmented with permissions from
// any services it depends on that expose consumer permissions
func getConsumerPermissions(ctx context.Context, c kclient.Client, appInstance *v1.AppInstance, containerName string, container v1.Container) (result v1.Permissions, _ error) {
	result = v1.Permissions{
		ServiceName: containerName,
	}

	for _, dep := range container.Dependencies {
		// This shouldn't happen, but okay?
		if dep.TargetName == "" {
			continue
		}

		svc := &v1.ServiceInstance{}
		if err := ref.Lookup(ctx, c, svc, appInstance.Status.Namespace, dep.TargetName); apierror.IsNotFound(err) {
			// We can ignore missing deps because the normal dep ordering will ensure that this container
			// can't be created/updated until its deps are
			continue
		} else if err != nil {
			return result, err
		}

		if svc.Spec.Consumer == nil {
			continue
		}

		if svc.Spec.Consumer.Permissions != nil {
			result.Rules = append(result.Rules, svc.Spec.Consumer.Permissions.GetRules()...)
		}
	}

	return
}

func getPermissions(ctx context.Context, c kclient.Client, appInstance *v1.AppInstance, serviceName string, container v1.Container, checkConsumerPerms bool) (result v1.Permissions, _ error) {
	result = v1.FindPermission(serviceName, appInstance.Status.Permissions)
	consumedPerms, err := getConsumerPermissions(ctx, c, appInstance, serviceName, container)
	if err != nil {
		return result, err
	}
	authorized := true
	if consumedPerms.HasRules() && checkConsumerPerms {
		logrus.Infof("@@@ CHECKING CONSUMER PERMS FOR %s", serviceName)
		unauthorized, err := checkConsumerPermsAuthz(ctx, c, appInstance, serviceName, consumedPerms)
		if err != nil {
			return result, err
		}
		if unauthorized.HasRules() {
			authorized = false
			appInstance.Status.DeniedConsumerPermissions = append(appInstance.Status.DeniedConsumerPermissions, unauthorized)
		}
	}
	if authorized {
		result.Rules = append(result.Rules, consumedPerms.GetRules()...)
	}
	return
}

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
