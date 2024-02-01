package quota

import (
	"fmt"
	"strconv"

	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	adminv1 "github.com/acorn-io/runtime/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/controller/appdefinition"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/z"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/acorn-io/runtime/pkg/condition"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/acorn-io/baaah/pkg/router"
)

// WaitForAllocation blocks the appInstance from being deployed until quota has been allocated on
// an associated QuotaRequest object.
func WaitForAllocation(req router.Request, resp router.Response) error {
	appInstance := req.Object.(*v1.AppInstance)

	// Create a condition setter for AppInstanceConditionQuota, which blocks the appInstance from being deployed
	// until quota has been allocated.
	status := condition.Setter(appInstance, resp, v1.AppInstanceConditionQuota)

	// Don't do anything if quota isn't enabled for this project.
	enforced, err := isEnforced(req, appInstance.Namespace)
	if err != nil {
		status.Error(err)
		return err
	} else if !enforced {
		status.Success()
		return nil
	}

	// Attempt to get the quotaRequest for this appInstance. It should exist with the name and namespace of the
	// appInstance being processed.
	quotaRequest := &adminv1.QuotaRequestInstance{}
	if err := req.Client.Get(req.Ctx, router.Key(appInstance.Namespace, appInstance.Name), quotaRequest); err != nil && !errors.IsNotFound(err) {
		return err
	}

	/*
		Determine how to proceed depending on if the quotaRequest exists and what it has written to its status. The three scenarios
		are the QuotaRequest:

		1. Exists and had an error while trying to allocate quota.
		2. Does not exist or has not yet had the requested resources marked as allocated.
		3. Exists and has successfully allocated the resources requested.
	*/
	if cond := quotaRequest.Status.Condition(adminv1.QuotaRequestCondition); cond.Error {
		status.Error(fmt.Errorf("quota allocation failed: %v", cond.Message))
	} else if waitingForAllocation(quotaRequest, appInstance) {
		status.Unknown("waiting for quota allocation")
	} else if quotaRequest.Status.Condition(adminv1.QuotaRequestCondition).Success {
		status.Success()
	}

	return nil
}

// waitingForAllocation determines if the quota request is waiting for allocation. This is determined
// by comparing the generation of the QuotaAppGeneration annotation of the quota request and the generation of
// the appInstance and by comparing the allocated resources of the quota request and the requested resources.
func waitingForAllocation(quotaRequest *adminv1.QuotaRequestInstance, appInstance *v1.AppInstance) bool {
	generation, err := strconv.ParseInt(quotaRequest.Annotations[labels.AcornAppGeneration], 10, 64)
	if err != nil {
		return true
	}
	return generation != appInstance.Generation ||
		!quotaRequest.Spec.Resources.Equals(quotaRequest.Status.AllocatedResources)
}

// EnsureQuotaRequest ensures that the quota request exists and is up to date.
func EnsureQuotaRequest(req router.Request, resp router.Response) error {
	appInstance := req.Object.(*v1.AppInstance)

	// Don't do anything if quota isn't enabled for this project
	if enforced, err := isEnforced(req, appInstance.Namespace); err != nil || !enforced {
		return err
	}

	// Create the quota request object and give calculate the standard numeric values
	name, namespace, app := appInstance.Name, appInstance.Namespace, appInstance.Status.AppSpec
	quotaRequest := &adminv1.QuotaRequestInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: name, Namespace: namespace,
			Annotations: map[string]string{labels.AcornAppGeneration: strconv.FormatInt(appInstance.Generation, 10)},
		},
		Spec: adminv1.QuotaRequestInstanceSpec{
			Resources: adminv1.QuotaRequestResources{
				BaseResources: adminv1.BaseResources{
					Jobs:    len(app.Jobs),
					Volumes: len(app.Volumes),
					Images:  len(app.Images),
				},
			},
		},
	}

	status := condition.Setter(appInstance, resp, v1.AppInstanceConditionQuota)

	// Add the more complex values to the quota request
	addContainers(app.Containers, quotaRequest)
	addCompute(app.Containers, appInstance, quotaRequest)
	// TODO: This is a stop-gap until we figure out how to handle the compute resources of
	//       jobs. The problem is that Jobs are not always running, so we can't just add
	//       their compute resources to the quota request permananetly. To some degree it'll
	//       have to be dynamic, but we can't do that until we have a better idea of how.
	// addCompute(app.Jobs, appInstance, quotaRequest)
	if err := addStorage(req, appInstance, quotaRequest); err != nil {
		status.Error(err)
		return err
	}

	resp.Objects(quotaRequest)
	return nil
}

// addContainers adds the number of containers and accounts for the scale of each container.
func addContainers(containers map[string]v1.Container, quotaRequest *adminv1.QuotaRequestInstance) {
	for _, container := range containers {
		quotaRequest.Spec.Resources.Containers += int(replicas(container.Scale))
	}
}

// addCompute adds the compute resources of the containers passed to the quota request.
func addCompute(containers map[string]v1.Container, appInstance *v1.AppInstance, quotaRequest *adminv1.QuotaRequestInstance) {
	// For each workload, add their memory/cpu requests to the quota request
	for name, container := range containers {
		var cpu, memory resource.Quantity
		if specific, ok := appInstance.Status.ResolvedOfferings.Containers[name]; ok {
			memory = *resource.NewQuantity(z.Dereference(specific.Memory), resource.BinarySI)
			cpu = *resource.NewMilliQuantity(z.Dereference(specific.CPU), resource.DecimalSI)
		} else if all, ok := appInstance.Status.ResolvedOfferings.Containers[""]; ok {
			cpu = *resource.NewMilliQuantity(z.Dereference(all.CPU), resource.DecimalSI)
			memory = *resource.NewQuantity(z.Dereference(all.Memory), resource.BinarySI)
		}

		// Multiply the memory/cpu requests by the scale of the container
		cpu.Mul(replicas(container.Scale))
		memory.Mul(replicas(container.Scale))

		// Add the compute resources to the quota request
		computeClass := appInstance.Status.ResolvedOfferings.Containers[name].Class
		quotaRequest.Spec.Resources.Add(adminv1.QuotaRequestResources{BaseResources: adminv1.BaseResources{ComputeClasses: adminv1.ComputeClassResources{
			computeClass: {
				Memory: memory,
				CPU:    cpu,
			},
		},
		}})

		// Recurse over any sidecars. Since sidecars can't have sidecars, this is safe.
		addCompute(container.Sidecars, appInstance, quotaRequest)
	}
}

// addStorage adds the storage resources of the volumes passed to the quota request.
func addStorage(req router.Request, appInstance *v1.AppInstance, quotaRequest *adminv1.QuotaRequestInstance) error {
	app := appInstance.Status.AppSpec

	// Add the volume storage needed to the quota request. We only parse net new volumes, not
	// existing ones that are then bound client-side.
	for name, volume := range app.Volumes {
		size := volume.Size
		if bound, boundSize := boundVolumeSize(name, appInstance.Spec.Volumes); bound {
			size = boundSize
		}

		// If the volume will be bound to an existing PV implicitly, then we don't need to count it. This will happen
		// when the incoming app has the same name as the app that created an existing released PV.
		if pvName, err := appdefinition.LookupExistingPV(req, appInstance, name); err != nil {
			return err
		} else if pvName != "" {
			continue
		}

		// Handle three cases:
		// 1. The volume's size is explicitly set to 0 or is implicitly set from boundVolumeSize. This means there is nothing to count.
		// 2. The volume's size is not set. This means we should assume the default size.
		// 3. The volume's size is set to a specific value. This means we should use that value.
		var sizeQuantity resource.Quantity
		switch size {
		case "0":
			continue
		case "":
			sizeQuantity = defaultVolumeSize(appInstance, name)
		default:
			parsedQuantity, err := resource.ParseQuantity(string(size))
			if err != nil {
				return err
			}
			sizeQuantity = parsedQuantity
		}

		volumeClass := appInstance.Status.ResolvedOfferings.Volumes[name].Class
		quotaRequest.Spec.Resources.Add(adminv1.QuotaRequestResources{
			BaseResources: adminv1.BaseResources{VolumeClasses: adminv1.VolumeClassResources{
				volumeClass: {VolumeStorage: sizeQuantity},
			}}})
	}

	// Add the secrets needed to the quota request. We only parse net new secrets, not
	// existing ones that are then bound client-side.
	for name := range app.Secrets {
		if boundSecret(name, appInstance.Spec.Secrets) {
			continue
		}
		quotaRequest.Spec.Resources.Secrets += 1
	}
	return nil
}

// defaultVolumeSize determines the default size of the specified volume. If the volume has a default size set
// on the status.Defaults.Volumes, it uses that. Otherwise, it uses the default size set on the status.Defaults.VolumeSize.
func defaultVolumeSize(appInstance *v1.AppInstance, name string) resource.Quantity {
	// Use the v1.DefaultSize if the appInstance doesn't have a default size set on the status.
	result := *v1.DefaultSize // Safe to dereference because it is statically set in the v1 package.

	// If the volume has a default size set on status.Defaults.Volumes, use that.
	if defaultVolume, set := appInstance.Status.Defaults.Volumes[name]; set {
		// We do not expect this to ever fail because VolumeClasses have their sizes validated. However,
		// if it does fail, we'll just use the default size instead.
		if parsedQuantity, err := resource.ParseQuantity(string(defaultVolume.Size)); err == nil {
			result = parsedQuantity
		}
	}

	return result
}

// boundVolumeSize determines if the specified volume will be bound and at what size. If it is bound to
// an existing volume, it returns "0" since that should not be double counted. If it would be bound to a
// volume in the Acornfile, it returns the new binding volume size. Otherwise, it returns false and a zero.
func boundVolumeSize(name string, bindings []v1.VolumeBinding) (bool, v1.Quantity) {
	for _, binding := range bindings {
		if binding.Target == name {
			if binding.Volume != "" {
				return true, "0"
			}
			return true, binding.Size
		}
	}
	return false, "0"
}

// boundSecret determines if the specified secret will be bound to an existing one.
func boundSecret(name string, bindings []v1.SecretBinding) bool {
	for _, binding := range bindings {
		if binding.Target == name && binding.Secret == "" {
			return true
		}
	}
	return false
}

// isEnforced determines if the project requires quota enforcement.
func isEnforced(req router.Request, namespace string) (bool, error) {
	project := v1.ProjectInstance{}
	if err := req.Client.Get(req.Ctx, router.Key("", namespace), &project); err != nil {
		return false, err
	}
	return project.Annotations[labels.ProjectEnforcedQuotaAnnotation] == "true", nil
}

// replicas returns the number of replicas based on an int32 pointer. If the
// pointer is nil, it is assumed to be 1.
func replicas(s *int32) int64 {
	if s != nil {
		return int64(*s)
	}
	return 1
}
