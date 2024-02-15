package defaults

import (
	"github.com/acorn-io/baaah/pkg/router"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	adminv1 "github.com/acorn-io/runtime/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/computeclasses"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/strings/slices"
)

// defaultMemory calculates the default that should be used and considers the defaults from the Config, ComputeClass, and
// runtime ComputeClass
func addDefaultMemory(req router.Request, cfg *apiv1.Config, appInstance *v1.AppInstance) error {
	if appInstance.Status.Defaults.Memory == nil {
		appInstance.Status.Defaults.Memory = v1.MemoryMap{}
	}

	var (
		defaultCC string
		err       error
	)
	if value, ok := appInstance.Spec.ComputeClasses[""]; ok {
		defaultCC = value
	} else {
		defaultCC, err = adminv1.GetDefaultComputeClassName(req.Ctx, req.Client, appInstance.Namespace)
		if err != nil {
			return err
		}
	}

	appInstance.Status.Defaults.Memory[""] = cfg.WorkloadMemoryDefault
	cc, err := computeclasses.GetAsProjectComputeClassInstance(req.Ctx, req.Client, appInstance.Status.Namespace, defaultCC)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}

	if cc != nil {
		parsedMemory, err := computeclasses.ParseComputeClassMemoryInternal(cc.Memory)
		if err != nil {
			return err
		}
		def := parsedMemory.Def.Value()
		appInstance.Status.Defaults.Memory[""] = &def
	}

	if err := addWorkloadMemoryDefault(req, appInstance, cfg.WorkloadMemoryDefault, appInstance.Status.AppSpec.Containers); err != nil {
		return err
	}

	if err := addWorkloadMemoryDefault(req, appInstance, cfg.WorkloadMemoryDefault, appInstance.Status.AppSpec.Jobs); err != nil {
		return err
	}

	// Remove any memory defaults for containers that are no longer defined in the app.
	allContainers := appInstance.GetAllContainerNames()
	for containerName := range appInstance.Status.Defaults.Memory {
		if containerName == "" {
			continue
		}
		if !slices.Contains(allContainers, containerName) {
			delete(appInstance.Status.Defaults.Memory, containerName)
		}
	}

	return nil
}

func addWorkloadMemoryDefault(req router.Request, appInstance *v1.AppInstance, configDefault *int64, containers map[string]v1.Container) error {
	for name, container := range containers {
		memory := configDefault
		computeClass, err := computeclasses.GetClassForWorkload(req.Ctx, req.Client, appInstance.Spec.ComputeClasses, container, name, appInstance.Namespace)
		if err != nil {
			return err
		}

		if computeClass != nil {
			parsedMemory, err := computeclasses.ParseComputeClassMemoryInternal(computeClass.Memory)
			if err != nil {
				return err
			}
			def := parsedMemory.Def.Value()
			memory = &def
		}
		appInstance.Status.Defaults.Memory[name] = memory

		for sidecarName := range container.Sidecars {
			appInstance.Status.Defaults.Memory[sidecarName] = memory
		}
	}

	return nil
}
