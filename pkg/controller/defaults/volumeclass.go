package defaults

import (
	"context"
	"fmt"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	adminv1 "github.com/acorn-io/acorn/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/volume"
	"github.com/acorn-io/baaah/pkg/typed"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func addVolumeClassDefaults(ctx context.Context, c kclient.Client, app *v1.AppInstance) error {
	if len(app.Status.AppSpec.Volumes) == 0 {
		return nil
	}

	volumeClasses, err := volume.GetVolumeClasses(ctx, c, app.Namespace)
	if err != nil {
		return err
	}

	var defaultVolumeClass adminv1.ProjectVolumeClassInstance
	for _, entry := range typed.Sorted(volumeClasses) {
		vc := entry.Value
		if vc.Default && !vc.Inactive {
			if defaultVolumeClass.Default {
				return fmt.Errorf("cannot establish defaults because two defaults volume classes exist: %s and %s", defaultVolumeClass.Name, vc.Name)
			}
			defaultVolumeClass = vc
		}
	}

	if app.Status.Defaults.Volumes == nil {
		app.Status.Defaults.Volumes = make(map[string]v1.VolumeDefault)
	}

	volumeBindings := volume.SliceToMap(app.Spec.Volumes, func(vb v1.VolumeBinding) string {
		return vb.Target
	})

	for name, vol := range app.Status.AppSpec.Volumes {
		var volDefaults v1.VolumeDefault
		vol = volume.CopyVolumeDefaults(vol, volumeBindings[name], volDefaults)
		if vol.Class == "" {
			volDefaults.Class = defaultVolumeClass.Name
			vol.Class = volDefaults.Class
		}
		if vol.Size == "" {
			volDefaults.Size = volumeClasses[vol.Class].Size.Default
		}
		if len(vol.AccessModes) == 0 {
			volDefaults.AccessModes = volumeClasses[vol.Class].AllowedAccessModes
		}

		app.Status.Defaults.Volumes[name] = volDefaults
	}

	return nil
}
