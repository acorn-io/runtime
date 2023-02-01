package defaults

import (
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	adminv1 "github.com/acorn-io/acorn/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/acorn-io/baaah/pkg/router"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// defaultMemory calculates the default that should be used and considers the defaults from the Config, WorkloadClass, and
// runtime WorkloadClass
func addDefaultMemory(req router.Request, cfg *apiv1.Config, appInstance *v1.AppInstance) error {
	if appInstance.Status.Defaults.Memory == nil {
		appInstance.Status.Defaults.Memory = v1.MemoryMap{}
	}

	var (
		defaultWC string
		err       error
	)
	if value, ok := appInstance.Spec.WorkloadClass[""]; ok {
		defaultWC = value
	} else {
		defaultWC, err = adminv1.GetDefaultWorkloadClass(req.Ctx, req.Client, appInstance.Namespace)
		if err != nil {
			return err
		}
	}

	appInstance.Status.Defaults.Memory[""] = cfg.WorkloadMemoryDefault
	wc, err := adminv1.GetAsProjectWorkloadClassInstance(req.Ctx, req.Client, appInstance.Status.Namespace, defaultWC)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}

	if wc != nil {
		parsedMemory, err := adminv1.ParseWorkloadClassMemory(wc.Memory)
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

	return nil
}

func addWorkloadMemoryDefault(req router.Request, appInstance *v1.AppInstance, configDefault *int64, containers map[string]v1.Container) error {
	for name, container := range containers {
		memory := configDefault
		workloadClass, err := adminv1.GetClassForWorkload(req.Ctx, req.Client, appInstance.Spec.WorkloadClass, container, name, appInstance.Namespace)
		if workloadClass == nil && err != nil {
			return err
		}

		if workloadClass != nil {
			parsedMemory, err := adminv1.ParseWorkloadClassMemory(workloadClass.Memory)
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
