package permissions

import (
	"context"

	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/imagerules"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/profiles"
	"github.com/acorn-io/runtime/pkg/ref"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// GetConsumerPermissions returns the permissions for a given container augmented with permissions from
// any services it depends on that expose consumer permissions
func GetConsumerPermissions(ctx context.Context, c kclient.Client, appInstance *v1.AppInstance, containerName string, container v1.Container) (result v1.Permissions, _ error) {
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

func collectConsumerPermissions(ctx context.Context, client kclient.Client, app *v1.AppInstance) ([]v1.Permissions, error) {
	var result []v1.Permissions

	for containerName, containerDef := range app.Status.AppSpec.Containers {
		consumerPerms, err := GetConsumerPermissions(ctx, client, app, containerName, containerDef)
		if err != nil {
			return nil, err
		}
		result = append(result, consumerPerms)
	}

	for functionName, functionDef := range app.Status.AppSpec.Functions {
		consumerPerms, err := GetConsumerPermissions(ctx, client, app, functionName, functionDef)
		if err != nil {
			return nil, err
		}
		result = append(result, consumerPerms)
	}

	for jobName, jobDef := range app.Status.AppSpec.Jobs {
		consumerPerms, err := GetConsumerPermissions(ctx, client, app, jobName, jobDef)
		if err != nil {
			return nil, err
		}
		result = append(result, consumerPerms)
	}

	return result, nil
}

func CheckConsumerPermsAuthorized(ctx context.Context, c kclient.Client, appInstance *v1.AppInstance, consumedPerms []v1.Permissions) ([]v1.Permissions, error) {
	imageName := appInstance.Status.AppImage.Name

	// I wonder if this is OK here, since at this point, there could already be a new image staged
	// and thus the annotation would already be updated and may not match the image that we use here.
	// That should be a very rare case though and has to have a very good timing.
	// Anyway, this couldn't be used for privilege escalation, since we're still using the current digest.
	if oi, ok := appInstance.Annotations[labels.AcornOriginalImage]; ok {
		imageName = oi
	}

	authzPerms, err := imagerules.GetAuthorizedPermissions(ctx, c, appInstance.Namespace, imageName, appInstance.Status.AppImage.Digest)
	if err != nil {
		return nil, err
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

	denied := []v1.Permissions{}

	for _, tmp := range consumedPerms {
		if d, granted := v1.GrantsAll(appInstance.Status.Namespace, []v1.Permissions{tmp}, copyWithName(authzPerms, tmp.ServiceName)); !granted {
			denied = append(denied, d...)
		}
	}

	return denied, nil
}

func ConsumerPermissions(req router.Request, _ router.Response) error {
	app := req.Object.(*v1.AppInstance)

	iraEnabled, err := config.GetFeature(req.Ctx, req.Client, profiles.FeatureImageRoleAuthorizations)
	if err != nil {
		return err
	}

	// Register for re-triggering when any service instance changes
	svclist := &v1.ServiceInstanceList{}
	if err := req.List(svclist, &kclient.ListOptions{
		Namespace: app.Status.Namespace,
	}); err != nil {
		return err
	}

	consumerPerms, err := collectConsumerPermissions(req.Ctx, req.Client, app)
	if err != nil {
		return err
	}

	newPerms, ok := v1.GrantsAll(app.Namespace, consumerPerms, app.Status.Permissions)
	if ok {
		// nothing new -> nothing to do
		app.Status.DeniedConsumerPermissions = nil
		return nil
	}

	if iraEnabled {
		denied, err := CheckConsumerPermsAuthorized(req.Ctx, req.Client, app, newPerms)
		if err != nil {
			return err
		}

		if len(denied) > 0 {
			app.Status.DeniedConsumerPermissions = denied
			return nil
		}
	}

	app.Status.DeniedConsumerPermissions = nil
	app.Status.Permissions = v1.SimplifySet(append(app.Status.Permissions, newPerms...))
	return nil
}
