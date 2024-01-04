package snapshot

import (
	"context"
	"errors"

	"github.com/acorn-io/baaah/pkg/router"
	snapshotv1 "github.com/acorn-io/runtime/pkg/apis/snapshot.storage.k8s.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/util/storage"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ClassName = "acorn-snapshot-class"
)

// SyncSnapshotClasses creates the snapshot class using the default storage class.
// May return an error.
func SyncSnapshotClasses(req router.Request, resp router.Response) error {
	// skip if no-object or deleted
	if req.Object == nil || !req.Object.GetDeletionTimestamp().IsZero() {
		return nil
	}

	storageClass := req.Object.(*storagev1.StorageClass)

	// skip if not the default storage class
	if storageClass.Annotations[storage.IsDefaultStorageClassAnnotation] != "true" {
		return nil
	}

	resp.Objects(&snapshotv1.VolumeSnapshotClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: ClassName,
			Labels: map[string]string{
				labels.AcornManaged: "true",
			},
		},
		Driver:         storageClass.Provisioner,
		DeletionPolicy: "Delete",
	})

	return nil
}

// CreateSnapshotClass creates a snapshot class if one does not yet exist.
// May return an error.
func CreateSnapshotClass(ctx context.Context, client kclient.WithWatch) error {
	acornSnapshotClass := &snapshotv1.VolumeSnapshotClass{}
	err := client.Get(ctx, kclient.ObjectKey{
		Name: ClassName,
	}, acornSnapshotClass)
	if err == nil {
		// it already exists
		return nil
	} else if !apierrors.IsNotFound(err) {
		return err
	}

	storageClasses := &storagev1.StorageClassList{}
	err = client.List(ctx, storageClasses)
	if err != nil {
		return err
	}

	var defaultStorageClass *storagev1.StorageClass
	for _, sc := range storageClasses.Items {
		if sc.Annotations[storage.IsDefaultStorageClassAnnotation] == "true" {
			if defaultStorageClass != nil {
				return errors.New("multiple default storage classes")
			}

			copySc := sc

			defaultStorageClass = &copySc
		}
	}

	return client.Create(ctx, &snapshotv1.VolumeSnapshotClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: ClassName,
			Labels: map[string]string{
				labels.AcornManaged: "true",
			},
		},
		Driver:         defaultStorageClass.Provisioner,
		DeletionPolicy: "Delete",
	})
}
