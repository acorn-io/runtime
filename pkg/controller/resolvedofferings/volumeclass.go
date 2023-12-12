package resolvedofferings

import (
	"context"
	"fmt"

	"github.com/acorn-io/baaah/pkg/typed"
	v1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/volume"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
		vol, err = volume.ResolveVolumeRequest(ctx, c, vol, volumeBindings[name], volumeClasses, defaultVolumeClass, app.Status.ResolvedOfferings.Volumes[name])
		if err != nil {
			return err
		}
		resolvedVolume := internalv1.VolumeResolvedOffering{
			AccessModes: vol.AccessModes,
			Class:       vol.Class,
			Size:        vol.Size,
		}

		// If an existing volume is bound to this volume request, then find it and use its values as the resolved offerings
		if volumeBindings[name].Volume != "" {
			// try to find the volume
			var boundVolume v1.Volume
			if err := c.Get(ctx, kclient.ObjectKey{Namespace: app.Namespace, Name: volumeBindings[name].Volume}, &boundVolume); err != nil && !apierrors.IsNotFound(err) {
				return fmt.Errorf("error while looking for bound volume %s in namespace %s: %w", volumeBindings[name].Volume, app.Namespace, err)
			}

			resolvedVolume.AccessModes = boundVolume.Spec.AccessModes
			resolvedVolume.Class = boundVolume.Spec.Class
			resolvedVolume.Size, err = internalv1.ParseQuantity(boundVolume.Spec.Capacity.String())
			if err != nil {
				return err
			}
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
