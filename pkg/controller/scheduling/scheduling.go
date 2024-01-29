package scheduling

import (
	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	adminv1 "github.com/acorn-io/runtime/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/computeclasses"
	"github.com/acorn-io/runtime/pkg/condition"
	"github.com/acorn-io/runtime/pkg/config"
	tl "github.com/acorn-io/runtime/pkg/tolerations"
	"github.com/acorn-io/z"
	corev1 "k8s.io/api/core/v1"
	nodev1 "k8s.io/api/node/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
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

		computeClass, err := computeclasses.GetClassForWorkload(req.Ctx, req.Client, appInstance.Spec.ComputeClasses, container, name, appInstance.Namespace, appInstance.Spec.Region)
		if err != nil {
			return err
		}

		requirements, err := ResourceRequirements(req, appInstance, name, container, computeClass)
		if err != nil {
			return err
		}

		for sidecarName, sidecarContainer := range container.Sidecars {
			// disable extra resource indication (GPU et al. for sidecars)
			if computeClass != nil && computeClass.Resources != nil {
				computeClass.Resources = &corev1.ResourceRequirements{}
			}
			sidecarRequirements, err := ResourceRequirements(req, appInstance, sidecarName, sidecarContainer, computeClass)
			if err != nil {
				return err
			}
			appInstance.Status.Scheduling[sidecarName] = v1.Scheduling{Requirements: *sidecarRequirements}
		}

		affinity, tolerations = Nodes(req, computeClass)

		priorityClassName, err := priorityClassName(req, computeClass)
		if err != nil {
			return err
		}

		runtimeClassName, err := runtimeClassName(req, computeClass)
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
			Requirements:      *requirements,
			Affinity:          affinity,
			Tolerations:       tolerations,
			PriorityClassName: priorityClassName,
			RuntimeClassName:  runtimeClassName,
		}
	}
	return nil
}

// Nodes returns the Affinity and Tolerations from a ComputeClass if they exist
func Nodes(req router.Request, computeClass *adminv1.ProjectComputeClassInstance) (*corev1.Affinity, []corev1.Toleration) {
	if computeClass == nil {
		return nil, nil
	}
	return computeClass.Affinity, computeClass.Tolerations
}

// PriorityClassName checks that a defined PriorityClass exists and returns the name of it
func priorityClassName(req router.Request, computeClass *adminv1.ProjectComputeClassInstance) (string, error) {
	if computeClass == nil || computeClass.PriorityClassName == "" {
		return "", nil
	}

	// Verify that the PriorityClass exists
	priorityClassName := &schedulingv1.PriorityClass{}
	if err := req.Client.Get(req.Ctx, router.Key("", computeClass.PriorityClassName), priorityClassName); err != nil {
		return "", err
	}

	return computeClass.PriorityClassName, nil
}

// RuntimeClassName checks that a defined RuntimeClass exists and returns the name of it
func runtimeClassName(req router.Request, computeClass *adminv1.ProjectComputeClassInstance) (string, error) {
	if computeClass == nil || computeClass.RuntimeClassName == "" {
		return "", nil
	}

	// Verify that the RuntimeClass exists
	runtimeClassName := &nodev1.RuntimeClass{}
	if err := req.Client.Get(req.Ctx, router.Key("", computeClass.RuntimeClassName), runtimeClassName); err != nil {
		return "", err
	}

	return computeClass.RuntimeClassName, nil
}

// ResourceRequirements determines the cpu and memory amount to be set for the limits/requests of the Pod
func ResourceRequirements(req router.Request, app *v1.AppInstance, containerName string, container v1.Container, computeClass *adminv1.ProjectComputeClassInstance) (*corev1.ResourceRequirements, error) {
	cfg, err := config.Get(req.Ctx, req.Client)
	if err != nil {
		return nil, err
	}

	requirements := &corev1.ResourceRequirements{Limits: corev1.ResourceList{}, Requests: corev1.ResourceList{}}
	if z.Dereference(cfg.IgnoreResourceRequirements) {
		return requirements, nil
	}

	if computeClass != nil && computeClass.Resources != nil {
		if computeClass.Resources.Requests != nil {
			requirements.Requests = computeClass.Resources.Requests
		}
		if computeClass.Resources.Limits != nil {
			requirements.Limits = computeClass.Resources.Limits
		}
	}

	var memDefault *int64
	if val := app.Status.Defaults.Memory[containerName]; val != nil {
		memDefault = val
	} else if val := app.Status.Defaults.Memory[""]; val != nil {
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
			memMax = z.Pointer(maxQuantity.Value())
		}
	}

	memoryLimit, err := v1.ValidateMemory(app.Spec.Memory, containerName, container, memDefault, memMax)
	if err != nil {
		return nil, err
	}

	// Figure out the scaled value of memory to request based on the compute class
	memoryRequest := memoryLimit.DeepCopy()
	if computeClass != nil && computeClass.Memory.RequestScaler != 0 {
		// The following line should hold up without loss of precision up to 4 petabytes
		memoryRequest.Set(int64(memoryLimit.AsApproximateFloat64() * computeClass.Memory.RequestScaler))
	}

	if memoryLimit.Value() != 0 {
		requirements.Requests[corev1.ResourceMemory] = memoryRequest
		requirements.Limits[corev1.ResourceMemory] = memoryLimit
	}

	if computeClass != nil {
		cpuQuantity, err := computeclasses.CalculateCPU(*computeClass, memoryRequest)
		if err != nil {
			return nil, err
		}
		if cpuQuantity.Value() != 0 {
			requirements.Requests[corev1.ResourceCPU] = cpuQuantity
		}
	}

	return requirements, nil
}
