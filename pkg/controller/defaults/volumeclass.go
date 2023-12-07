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
		volDefaults := app.Status.Defaults.Volumes[name]
		vol = volume.CopyVolumeDefaults(vol, volumeBindings[name], volDefaults)

		// This is a bit of a hack as we're migrating away from the VolumeSize field. Essentially,
		// we want to ensure that app.Status.Volumes[name] always has a size set. If the VolumeSize
		// field has been set in the past, we want to mirgrate that over to be set on app.Status.Volumes[name].
		// There is another edge case where the Size field was set by a VolumeClass's default size. In this
		// case we want to leave the Size field alone.
		if app.Status.Defaults.VolumeSize != nil && volDefaults.Size == "" {
			volDefaults.Size = v1.Quantity(app.Status.Defaults.VolumeSize.String())
		}

		// If the Volume already has defaults, skip these steps. We don't want to overwrite
		// default class or access modes for volumes as it can lead to unexpected behavior when
		// volume classes are updated.
		if _, alreadySet := app.Status.Defaults.Volumes[name]; !alreadySet {
			if vol.Class == "" && defaultVolumeClass != nil {
				volDefaults.Class = defaultVolumeClass.Name
				vol.Class = volDefaults.Class
			}
			if len(vol.AccessModes) == 0 {
				volDefaults.AccessModes = volumeClasses[vol.Class].AllowedAccessModes
			}
			if vol.Size == "" {
				volDefaults.Size = volumeClasses[vol.Class].Size.Default
				if volDefaults.Size == "" {
					defaultSize, err := getDefaultVolumeSize(ctx, c)
					if err != nil {
						return err
					}
					volDefaults.Size = defaultSize
				}
			}
		}

		app.Status.Defaults.Volumes[name] = volDefaults
	}

	return nil
}

func getDefaultVolumeSize(ctx context.Context, c kclient.Client) (v1.Quantity, error) {
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
