package quota

import (
	"fmt"

	apiv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	adminv1 "github.com/acorn-io/runtime/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/acorn-io/runtime/pkg/condition"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/acorn-io/baaah/pkg/router"
)

// WaitForAllocation blocks the appInstance from being deployed until quota has been allocated on
// an associated QuotaRequest object.
func WaitForAllocation(req router.Request, resp router.Response) error {
	appInstance := req.Object.(*apiv1.AppInstance)

	// Create a condition setter for AppInstanceConditionQuotaAllocated, which blocks the appInstance from being deployed
	// until quota has been allocated.
	status := condition.Setter(appInstance, resp, apiv1.AppInstanceConditionQuotaAllocated)

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
	err = req.Client.Get(req.Ctx, router.Key(appInstance.Namespace, appInstance.Name), quotaRequest)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	/*
		Determine how to proceed depending on if the quotaRequest exists and what it has written to its status. The four scenarios
		are QuotaRequest:

		1. Exists and has set FailedResources implying that the quota allocation failed.
		2. Exists and had an error while trying to allocate quota.
		3. Does not exist or has not yet been allocated the resources requested.
		4. Exists and has successfully allocated the resources requested.
	*/
	if quotaRequest.Status.FailedResources != nil {
		status.Error(fmt.Errorf("failed to provision the following resources: %v", quotaRequest.Status.FailedResources.NonEmptyString()))
	} else if cond := quotaRequest.Status.Condition(adminv1.QuotaRequestCondition); cond.Error {
		status.Error(fmt.Errorf("error occurred while trying to allocate quota: %v", cond.Message))
	} else if err != nil || !quotaRequest.Spec.Resources.Equals(quotaRequest.Status.AllocatedResources) {
		status.Unknown("waiting for quota allocation")
	} else if quotaRequest.Status.Condition(adminv1.QuotaRequestCondition).Success {
		status.Success()
	}

	return nil
}

// EnsureQuotaRequest ensures that the quota request exists and is up to date.
func EnsureQuotaRequest(req router.Request, resp router.Response) error {
	appInstance := req.Object.(*apiv1.AppInstance)

	// Don't do anything if quota isn't enabled for this project
	if enforced, err := isEnforced(req, appInstance.Namespace); err != nil || !enforced {
		return err
	}

	// Create the quota request object and calculate the standard numeric values
	name, namespace, app := appInstance.Name, appInstance.Namespace, appInstance.Status.AppSpec
	quotaRequest := &adminv1.QuotaRequestInstance{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec:       adminv1.QuotaRequestInstanceSpec{Resources: adminv1.Resources{}},
	}

	// Instatiate the quota request's maps to prevent nil pointer errors
	quotaRequest.Spec.Resources = adminv1.NewResources()

	if containers := len(app.Containers); containers > 0 {
		quotaRequest.Spec.Resources.Counts[adminv1.Containers] = containers
	}
	if jobs := len(app.Jobs); jobs > 0 {
		quotaRequest.Spec.Resources.Counts[adminv1.Jobs] = jobs
	}
	if images := len(app.Images); images > 0 {
		quotaRequest.Spec.Resources.Counts[adminv1.Images] = images
	}

	status := condition.Setter(appInstance, resp, apiv1.AppInstanceConditionQuotaAllocated)

	// Add the more complex values to the quota request
	addCompute(app.Containers, appInstance, quotaRequest)
	addCompute(app.Jobs, appInstance, quotaRequest)
	if err := addStorage(appInstance, quotaRequest); err != nil {
		status.Error(err)
		return err
	}

	resp.Objects(quotaRequest)
	return nil
}

// addCompute adds the compute resources of the containers passed to the quota request.
func addCompute(containers map[string]apiv1.Container, appInstance *apiv1.AppInstance, quotaRequest *adminv1.QuotaRequestInstance) {
	// For each workload, add their memory/cpu requests to the quota request
	for name, container := range containers {
		var requirements corev1.ResourceRequirements
		if specific, ok := appInstance.Status.Scheduling[name]; ok {
			requirements = specific.Requirements
		} else if all, ok := appInstance.Status.Scheduling[""]; ok {
			requirements = all.Requirements
		}

		cpu := quotaRequest.Spec.Resources.Quantities[adminv1.CPU]
		cpu.Add(requirements.Requests["cpu"])
		quotaRequest.Spec.Resources.Quantities[adminv1.CPU] = cpu

		memory := quotaRequest.Spec.Resources.Quantities[adminv1.Memory]
		memory.Add(requirements.Requests["memory"])
		quotaRequest.Spec.Resources.Quantities[adminv1.Memory] = memory

		// Recurse over any sidecars. Since sidecars can't have sidecars, this is safe.
		addCompute(container.Sidecars, appInstance, quotaRequest)
	}
}

// addStorage adds the storage resources of the volumes passed to the quota request.
func addStorage(appInstance *apiv1.AppInstance, quotaRequest *adminv1.QuotaRequestInstance) error {
	app := appInstance.Status.AppSpec

	// Add the volume storage needed to the quota request. We only parse net new volumes, not
	// existing ones that are then bound client-side.
	for name, volume := range app.Volumes {
		size := volume.Size
		if bound, boundSize := boundVolumeSize(name, appInstance.Spec.Volumes); bound {
			size = boundSize
		}

		// No need to proceed if the size is empty
		if size == "" || size == "0" {
			continue
		}

		parsedSize, err := resource.ParseQuantity(string(size))
		if err != nil {
			return err
		}
		volumeStorage := quotaRequest.Spec.Resources.Quantities[adminv1.VolumeStorage]
		volumeStorage.Add(parsedSize)
		quotaRequest.Spec.Resources.PersistentCounts[adminv1.Volumes] += 1
	}

	// Add the secrets needed to the quota request. We only parse net new secrets, not
	// existing ones that are then bound client-side.
	for name := range app.Secrets {
		if boundSecret(name, appInstance.Spec.Secrets) {
			continue
		}
		quotaRequest.Spec.Resources.PersistentCounts[adminv1.Secrets] += 1
	}
	return nil
}

// boundVolumeSize determines if the specified volume will be bound to an existing one. If
// it will not be bound, the size of the new volume is returned.
func boundVolumeSize(name string, bindings []apiv1.VolumeBinding) (bool, apiv1.Quantity) {
	for _, binding := range bindings {
		if binding.Target == name && binding.Volume == "" {
			return true, binding.Size
		}
	}
	return false, "0"
}

// boundSecret determines if the specified secret will be bound to an existing one.
func boundSecret(name string, bindings []apiv1.SecretBinding) bool {
	for _, binding := range bindings {
		if binding.Target == name && binding.Secret == "" {
			return true
		}
	}
	return false
}

// isEnforced determines if the project has quota enabled.
func isEnforced(req router.Request, namespace string) (bool, error) {
	// Use the underlying Namespace type that stores Projects
	project := corev1.Namespace{}
	if err := req.Client.Get(req.Ctx, router.Key("", namespace), &project); err != nil {
		return false, err
	}
	return project.Annotations[labels.ProjectEnforcedQuotaAnnotation] == "true", nil
}
