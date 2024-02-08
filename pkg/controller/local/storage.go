package local

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/acorn-io/baaah/pkg/apply"
	"github.com/acorn-io/baaah/pkg/name"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/z"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	storageClass = "acorn-local"
	path         = "/var/lib/rancher/k3s/storage/local"
	FinalizerID  = "local-storage.acorn.io/finalizer"
)

func NewCreateFolder() (router.Handler, error) {
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, err
	}
	return router.HandlerFunc(createFolder), nil
}

func CleanupStorage(req router.Request, _ router.Response) error {
	pv := req.Object.(*corev1.PersistentVolume)
	if pv.DeletionTimestamp.IsZero() || len(pv.Finalizers) == 0 || pv.Finalizers[0] != FinalizerID {
		return nil
	}

	if pv.Spec.PersistentVolumeSource.HostPath != nil {
		pvPath := pv.Spec.PersistentVolumeSource.HostPath.Path
		suffix := strings.TrimPrefix(pvPath, path)
		if len(suffix) > 2 && suffix != pvPath {
			if err := exec.Command("rm", "-rf", pvPath).Run(); err != nil {
				return fmt.Errorf("failed to delete %s: %w", pvPath, err)
			}
		}
	}

	pv.Finalizers = pv.Finalizers[1:]
	return req.Client.Update(req.Ctx, pv)
}

func createFolder(req router.Request, _ router.Response) error {
	pvc := req.Object.(*corev1.PersistentVolumeClaim)
	if z.Dereference(pvc.Spec.StorageClassName) != storageClass || pvc.Status.Phase != corev1.ClaimPending || pvc.Spec.VolumeName != "" {
		return nil
	}

	pvName := name.SafeConcatName(pvc.Name, string(pvc.UID))
	path := filepath.Join(path, pvName)
	if err := os.MkdirAll(path, 0777); err != nil {
		return err
	}

	if err := os.Chmod(path, 0777); err != nil {
		return err
	}

	err := apply.New(req.Client).Ensure(req.Ctx, &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:       pvName,
			Finalizers: []string{FinalizerID},
		},
		Spec: corev1.PersistentVolumeSpec{
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: path,
					Type: z.Pointer(corev1.HostPathDirectory),
				},
			},
			AccessModes: pvc.Spec.AccessModes,
			Capacity:    pvc.Spec.Resources.Requests,
			ClaimRef: &corev1.ObjectReference{
				Kind:            "PersistentVolumeClaim",
				Namespace:       pvc.Namespace,
				Name:            pvc.Name,
				UID:             pvc.UID,
				APIVersion:      "v1",
				ResourceVersion: pvc.ResourceVersion,
			},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
			StorageClassName:              storageClass,
			VolumeMode:                    z.Pointer(corev1.PersistentVolumeFilesystem),
		},
	})
	if err != nil {
		return err
	}

	pvc.Spec.VolumeName = pvName
	return req.Client.Update(req.Ctx, pvc)
}
