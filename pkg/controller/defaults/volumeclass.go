package defaults

import (
	"context"
	"fmt"

	"github.com/acorn-io/baaah/pkg/typed"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/volume"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func addVolumeClassDefaults(ctx context.Context, c kclient.Client, app *v1.AppInstance) error {
	if len(app.Status.AppSpec.Volumes) == 0 {
		return nil
	}

	volumeClasses, defaultVolumeClass, err := volume.GetVolumeClassInstances(ctx, c, app.Namespace)
	if err != nil {
		return err
	}

	for _, entry := range typed.Sorted(volumeClasses) {
		vc := entry.Value
		if vc.Default && vc.Name != defaultVolumeClass.Name {
			return fmt.Errorf("cannot establish defaults because two defaults volume classes exist: %s and %s", defaultVolumeClass.Name, vc.Name)
		}
	}

	if app.Status.Defaults.Volumes == nil {
		app.Status.Defaults.Volumes = make(map[string]v1.VolumeDefault)
	}

	volumeBindings := volume.SliceToMap(app.Spec.Volumes, func(vb v1.VolumeBinding) string {
		return vb.Target
	})

	for name, vol := range app.Status.AppSpec.Volumes {
		// If the Volume already has defaults, skip it. We don't want to overwrite
		// defaults for volumes values as it can lead to unexpected behavior when volume
		// classes are updated. One example is a volume class going down in size, which
		// in turn will cause the volume to be sized down and likely go into an error state
		// on the next app update.
		if _, alreadySet := app.Status.Defaults.Volumes[name]; alreadySet {
			continue
		}

		var volDefaults v1.VolumeDefault
		vol = volume.CopyVolumeDefaults(vol, volumeBindings[name], volDefaults)
		if vol.Class == "" && defaultVolumeClass != nil {
			volDefaults.Class = defaultVolumeClass.Name
			vol.Class = volDefaults.Class
		}

		// Temporary migration step to ensure that the VolumeSize is set on the volDefaults.
		if app.Status.Defaults.VolumeSize != nil {
			volDefaults.Size = v1.Quantity(app.Status.Defaults.VolumeSize.String())
		} else if vol.Size == "" {
			if volumeClasses[vol.Class].Size.Default == "" {
				defaultSize, err := getDefaultVolumeSize(ctx, c, app)
				if err != nil {
					return err
				}
				volDefaults.Size = defaultSize
			} else {
				volDefaults.Size = volumeClasses[vol.Class].Size.Default
			}
		}
		if len(vol.AccessModes) == 0 {
			volDefaults.AccessModes = volumeClasses[vol.Class].AllowedAccessModes
		}

		app.Status.Defaults.Volumes[name] = volDefaults
	}

	return nil
}

func getDefaultVolumeSize(ctx context.Context, c kclient.Client, appInstance *v1.AppInstance) (v1.Quantity, error) {
	// If the Status.Defaults.VolumeSize has been set, use that.
	if appInstance.Status.Defaults.VolumeSize != nil {
		return v1.Quantity(appInstance.Status.Defaults.VolumeSize.String()), nil
	}

	cfg, err := config.Get(ctx, c)
	if err != nil {
		return "", err
	}

	// If the default volume size is set in the config, use that. Otherwise use the
	// package level default in v1.
	defaultVolumeSize := v1.DefaultSizeQuantity
	if cfg.VolumeSizeDefault != "" {
		defaultVolumeSize = v1.Quantity(cfg.VolumeSizeDefault)
	}

	return defaultVolumeSize, nil
}
