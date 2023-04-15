package scheduling

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/condition"
	tl "github.com/acorn-io/acorn/pkg/tolerations"

	adminv1 "github.com/acorn-io/acorn/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/baaah/pkg/router"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Calculate is a handler that sets the scheduling rules for an AppInstance to its
// status if and only if its generation is different from its observedGeneration.
//
// This is necessary because querying for scheduling rules will result in all running
// AppInstances using the backing resources (the Acorn Config or a ComputeClass, for example)
// to be redeployed when the resources change. By calculating scheduling rules only when the
// generation changes, we can ensure that updated backing resources are only applied when an
// AppInstance is updated directly.
func Calculate(req router.Request, resp router.Response) error {
	appInstance := req.Object.(*v1.AppInstance)
	status := condition.Setter(appInstance, resp, v1.AppInstanceConditionScheduling)

	// Only recalculate scheduling when a change is detected in generation or image digest.
	if appInstance.Generation != appInstance.Status.ObservedGeneration || appInstance.Status.AppImage.Digest != appInstance.Status.ObservedImageDigest {
		if err := calculate(req, appInstance); err != nil {
			status.Error(err)
			resp.DisablePrune()
			return nil
		}
	}
	status.Success()
	return nil
}

func calculate(req router.Request, appInstance *v1.AppInstance) error {
	if appInstance.Status.Scheduling == nil {
		appInstance.Status.Scheduling = map[string]v1.Scheduling{}
	}
	if err := addScheduling(req, appInstance, appInstance.Status.AppSpec.Containers); err != nil {
		return err
	}
	if err := addScheduling(req, appInstance, appInstance.Status.AppSpec.Jobs); err != nil {
		return err
	}
	return nil
}

func addScheduling(req router.Request, appInstance *v1.AppInstance, workloads map[string]v1.Container) error {
	for name, container := range workloads {
		var (
			affinity    *corev1.Affinity
			tolerations []corev1.Toleration
		)

		computeClass, err := adminv1.GetClassForWorkload(req.Ctx, req.Client, appInstance.Spec.ComputeClasses, container, name, appInstance.Namespace)
		if err != nil {
			return err
		}

		requirements, err := ResourceRequirements(req, appInstance, name, container, computeClass)
		if err != nil {
			return err
		}

		for sidecarName, sidecarContainer := range container.Sidecars {
			sidecarRequirements, err := ResourceRequirements(req, appInstance, sidecarName, sidecarContainer, computeClass)
			if err != nil {
				return err
			}
			appInstance.Status.Scheduling[sidecarName] = v1.Scheduling{Requirements: *sidecarRequirements}
		}

		affinity, tolerations, err = Nodes(req, name, container, appInstance, computeClass)
		if err != nil {
			return err
		}

		// Add default toleration to taints.acorn.io/workload. This is so that when worker nodes are tainted
		// with taints.acorn.io/workload, user app can still tolerate. Only add default toleration when toleration is not set
		if len(tolerations) == 0 {
			tolerations = append(tolerations, corev1.Toleration{
				Key:      tl.WorkloadTolerationKey,
				Operator: corev1.TolerationOpExists,
			})
		}

		appInstance.Status.Scheduling[name] = v1.Scheduling{
			Requirements: *requirements,
			Affinity:     affinity,
			Tolerations:  tolerations,
		}
	}
	return nil
}

// Add edits the provided PodTemplateSpec to have the applied configuration for the ComputeClass and Memory values
func Nodes(req router.Request, name string, container v1.Container, app *v1.AppInstance, computeClass *adminv1.ProjectComputeClassInstance) (*corev1.Affinity, []corev1.Toleration, error) {
	if computeClass != nil {
		// Return any custom affinities and tolerations from the ComputeClass
		return computeClass.Affinity, computeClass.Tolerations, nil
	}

	return nil, nil, nil
}

// ResourceRequirements determines the cpu and memory amount to be set for the limits/requests of the Pod
func ResourceRequirements(req router.Request, app *v1.AppInstance, containerName string, container v1.Container, computeClass *adminv1.ProjectComputeClassInstance) (*corev1.ResourceRequirements, error) {
	cfg, err := config.Get(req.Ctx, req.Client)
	if err != nil {
		return nil, err
	}

	requirements := &corev1.ResourceRequirements{Limits: corev1.ResourceList{}, Requests: corev1.ResourceList{}}

	var memDefault *int64
	if val, ok := app.Status.Defaults.Memory[containerName]; ok && val != nil {
		memDefault = val
	} else if val, ok := app.Status.Defaults.Memory[""]; ok && val != nil {
		memDefault = val
	}

	memMax := cfg.WorkloadMemoryMaximum
	if computeClass != nil {
		memMax = new(int64)
		if computeClass.Memory.Max != "" && computeClass.Memory.Max != "0" {
			maxQuantity, err := resource.ParseQuantity(computeClass.Memory.Max)
			if err != nil {
				return nil, err
			}
			memMax = &[]int64{maxQuantity.Value()}[0]
		}
	}

	memoryQuantity, err := v1.ValidateMemory(app.Spec.Memory, containerName, container, memDefault, memMax)
	if err != nil {
		return nil, err
	}

	if memoryQuantity.Value() != 0 {
		requirements.Requests[corev1.ResourceMemory] = memoryQuantity
		requirements.Limits[corev1.ResourceMemory] = memoryQuantity
	}

	if computeClass != nil {
		cpuQuantity, err := adminv1.CalculateCPU(*computeClass, memDefault, memoryQuantity)
		if err != nil {
			return nil, err
		}
		if cpuQuantity.Value() != 0 {
			requirements.Requests[corev1.ResourceCPU] = cpuQuantity
		}
	}

	return requirements, nil
}
