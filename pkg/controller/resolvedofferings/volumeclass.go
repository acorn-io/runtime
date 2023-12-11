package resolvedofferings

import (
	"context"
	"fmt"

	"github.com/acorn-io/baaah/pkg/typed"
	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/volume"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func resolveVolumeClasses(ctx context.Context, c kclient.Client, app *internalv1.AppInstance) error {
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
		app.Status.ResolvedOfferings.Volumes = make(map[string]internalv1.VolumeResolvedOffering)
	}

	volumeBindings := volume.SliceToMap(app.Spec.Volumes, func(vb internalv1.VolumeBinding) string {
		return vb.Target
	})

	for name, vol := range app.Status.AppSpec.Volumes {
		vol, err = volume.ResolveVolumeRequest(ctx, c, app.Namespace, vol, volumeBindings[name], volumeClasses, defaultVolumeClass)
		resolvedVolume := internalv1.VolumeResolvedOffering{
			AccessModes: vol.AccessModes,
			Class:       vol.Class,
			Size:        vol.Size,
		}

		// This is a bit of a hack as we're migrating away from the VolumeSize field. Essentially,
		// we want to ensure that app.Status.Volumes[name] always has a size set. If the VolumeSize
		// field has been set in the past, we want to migrate that over to be set on app.Status.Volumes[name].
		// There is another edge case where the Size field was set by a VolumeClass's default size. In this
		// case we want to leave the Size field alone.
		if app.Status.ResolvedOfferings.VolumeSize != nil && resolvedVolume.Size == "" {
			resolvedVolume.Size = internalv1.Quantity(app.Status.ResolvedOfferings.VolumeSize.String())
		}

		app.Status.ResolvedOfferings.Volumes[name] = resolvedVolume
	}

	return nil
}
