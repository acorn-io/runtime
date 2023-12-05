package resolvedofferings

import (
	"context"
	"fmt"

	"github.com/acorn-io/baaah/pkg/typed"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/volume"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func resolveVolumeClasses(ctx context.Context, c kclient.Client, app *v1.AppInstance) error {
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
			return fmt.Errorf("cannot resolve volume classes because two defaults volume classes exist: %s and %s", defaultVolumeClass.Name, vc.Name)
		}
	}

	if app.Status.ResolvedOfferings.Volumes == nil {
		app.Status.ResolvedOfferings.Volumes = make(map[string]v1.VolumeResolvedOffering)
	}

	volumeBindings := volume.SliceToMap(app.Spec.Volumes, func(vb v1.VolumeBinding) string {
		return vb.Target
	})

	for name, vol := range app.Status.AppSpec.Volumes {
		resolvedVolume := app.Status.ResolvedOfferings.Volumes[name]
		vol = volume.ResolveVolumeRequest(vol, volumeBindings[name], resolvedVolume)

		// This is a bit of a hack as we're migrating away from the VolumeSize field. Essentially,
		// we want to ensure that app.Status.Volumes[name] always has a size set. If the VolumeSize
		// field has been set in the past, we want to migrate that over to be set on app.Status.Volumes[name].
		// There is another edge case where the Size field was set by a VolumeClass's default size. In this
		// case we want to leave the Size field alone.
		if app.Status.ResolvedOfferings.VolumeSize != nil && resolvedVolume.Size == "" {
			resolvedVolume.Size = v1.Quantity(app.Status.ResolvedOfferings.VolumeSize.String())
		}

		if _, alreadySet := app.Status.ResolvedOfferings.Volumes[name]; !alreadySet {
			resolvedVolume.Class = vol.Class
			if vol.Class == "" && defaultVolumeClass != nil {
				resolvedVolume.Class = defaultVolumeClass.Name
				vol.Class = defaultVolumeClass.Name
			}

			resolvedVolume.AccessModes = vol.AccessModes
			if len(vol.AccessModes) == 0 {
				resolvedVolume.AccessModes = volumeClasses[vol.Class].AllowedAccessModes
			}

			resolvedVolume.Size = vol.Size
			if vol.Size == "" {
				resolvedVolume.Size = volumeClasses[vol.Class].Size.Default
				if resolvedVolume.Size == "" {
					defaultSize, err := getDefaultVolumeSize(ctx, c)
					if err != nil {
						return err
					}
					resolvedVolume.Size = defaultSize
				}
			}
		}

		app.Status.ResolvedOfferings.Volumes[name] = resolvedVolume
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
