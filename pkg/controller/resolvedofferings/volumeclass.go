package resolvedofferings

import (
	"context"
	"fmt"

	"github.com/acorn-io/baaah/pkg/typed"
	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/volume"
	corev1 "k8s.io/api/core/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
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
			// Try to find the volume by public name first
			var boundVolumeList corev1.PersistentVolumeList
			if err := c.List(ctx, &boundVolumeList, &kclient.ListOptions{
				LabelSelector: klabels.SelectorFromSet(map[string]string{
					labels.AcornPublicName:   volumeBindings[name].Volume,
					labels.AcornAppNamespace: app.Namespace,
					labels.AcornManaged:      "true",
				}),
			}); err != nil {
				return fmt.Errorf("error while looking for bound volume %s in namespace %s: %w", volumeBindings[name].Volume, app.Namespace, err)
			} else if len(boundVolumeList.Items) == 0 {
				// See if the user provided a PV name instead
				var boundPV corev1.PersistentVolume
				if err := c.Get(ctx, kclient.ObjectKey{Name: volumeBindings[name].Volume}, &boundPV); err != nil {
					return fmt.Errorf("error while looking for bound volume %s in namespace %s: %w", volumeBindings[name].Volume, app.Namespace, err)
				} else if boundPV.ObjectMeta.Labels[labels.AcornAppNamespace] != app.Namespace {
					return fmt.Errorf("could not find volume %s in project %s", volumeBindings[name].Volume, app.Namespace)
				}

				boundVolumeList.Items = []corev1.PersistentVolume{boundPV}
			}

			// If we found the volume, then use its values as the resolved offerings
			if len(boundVolumeList.Items) == 1 {
				for _, a := range boundVolumeList.Items[0].Spec.AccessModes {
					resolvedVolume.AccessModes = append(resolvedVolume.AccessModes, internalv1.AccessMode(a))
				}
				resolvedVolume.Class = boundVolumeList.Items[0].ObjectMeta.Labels[labels.AcornVolumeClass]
				resolvedVolume.Size = internalv1.Quantity(boundVolumeList.Items[0].Spec.Capacity.Storage().String())
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
