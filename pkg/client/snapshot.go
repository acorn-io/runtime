package client

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	snapshotv1 "github.com/acorn-io/runtime/pkg/apis/snapshot.storage.k8s.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
	snapshot2 "github.com/acorn-io/runtime/pkg/snapshot"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

var (
	annotationAccessModes = "original-pvc-access-modes"
	annotationResources   = "original-pvc-resource"
	annotationVolumeName  = "original-pvc-volume-name"
	annotationVolumeClass = "original-pvc-volume-class"
)

// SnapshotCreate creates a VolumeSnapshot resource in the cluster using the details of the given *PersistentVolumeClaim.
// Returns a *VolumeSnapshot or error if one occurred.
func (c *DefaultClient) SnapshotCreate(ctx context.Context, pvc *corev1.PersistentVolumeClaim) (*snapshotv1.VolumeSnapshot, error) {
	err := snapshot2.CreateSnapshotClass(ctx, c.Client)
	if err != nil {
		return nil, err
	}

	name, ok := pvc.Labels["custom-name"]
	if !ok {
		name = pvc.Labels[labels.AcornPublicName] + "-" + strconv.FormatInt(time.Now().Unix(), 10)
	}

	accessModesJSON, err := json.Marshal(pvc.Spec.AccessModes)
	if err != nil {
		return nil, err
	}

	resourcesJSON, err := json.Marshal(pvc.Spec.Resources)
	if err != nil {
		return nil, err
	}

	snapshot := &snapshotv1.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: pvc.Namespace,
			Labels: map[string]string{
				labels.AcornAppName:      pvc.Labels[labels.AcornAppName],
				labels.AcornAppNamespace: pvc.Labels[labels.AcornAppNamespace],
				labels.AcornPublicName:   pvc.Labels[labels.AcornPublicName],
				labels.AcornManaged:      "true",
			},
			Annotations: map[string]string{
				annotationAccessModes: string(accessModesJSON),
				annotationResources:   string(resourcesJSON),
				annotationVolumeName:  pvc.Labels[labels.AcornVolumeName],
				annotationVolumeClass: pvc.Labels[labels.AcornVolumeClass],
			},
		},
		Spec: snapshotv1.VolumeSnapshotSpec{
			VolumeSnapshotClassName: pointer.String(snapshot2.ClassName),
			Source: snapshotv1.VolumeSnapshotSource{
				PersistentVolumeClaimName: pointer.String(pvc.Name),
			},
		},
	}

	return snapshot, c.Client.Create(ctx, snapshot)
}

// SnapshotList lists VolumeSnapshot. Returns []VolumeSnapshot or an error if one occurred.
func (c *DefaultClient) SnapshotList(ctx context.Context) ([]snapshotv1.VolumeSnapshot, error) {
	snapshots := &snapshotv1.VolumeSnapshotList{}
	err := c.Client.List(ctx, snapshots)
	if err != nil {
		return nil, err
	}

	sort.Slice(snapshots.Items, func(i, j int) bool {
		iTime := snapshots.Items[i].CreationTimestamp.Time
		jTime := snapshots.Items[j].CreationTimestamp.Time

		if iTime == jTime {
			return snapshots.Items[i].Name < snapshots.Items[j].Name
		}

		return iTime.After(jTime)
	})

	return snapshots.Items, nil
}

// SnapshotGet gets you a *VolumeSnapshot by name or error.
func (c *DefaultClient) SnapshotGet(ctx context.Context, name string) (*snapshotv1.VolumeSnapshot, error) {
	snapshots, err := c.SnapshotList(ctx)
	if err != nil {
		return nil, err
	}

	for _, snapshot := range snapshots {
		if snapshot.Name == name {
			return &snapshot, nil
		}
	}

	return nil, errors.New("snapshot not found")
}

// SnapshotDelete deletes the snapshot with the given name.
// May return an error.
func (c *DefaultClient) SnapshotDelete(ctx context.Context, name string) error {
	snapshot, err := c.SnapshotGet(ctx, name)
	if err != nil {
		return err
	}

	return c.Client.Delete(ctx, snapshot)
}

func (c *DefaultClient) SnapshotRestore(ctx context.Context, snapshotName string, volumeName string) error {
	snapshot, err := c.SnapshotGet(ctx, snapshotName)
	if err != nil {
		return err
	}

	var accessModes []corev1.PersistentVolumeAccessMode
	err = json.Unmarshal([]byte(snapshot.Annotations[annotationAccessModes]), &accessModes)
	if err != nil {
		return err
	}

	var resources corev1.ResourceRequirements
	err = json.Unmarshal([]byte(snapshot.Annotations[annotationResources]), &resources)
	if err != nil {
		return err
	}

	_, err = c.VolumeGet(ctx, volumeName)
	if err == nil {
		return errors.New("a volume by that name already exists")
	} else if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	newPVC := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      volumeName,
			Namespace: snapshot.Namespace,
			Labels: map[string]string{
				labels.AcornManaged:      "true",
				labels.AcornPublicName:   strings.Replace(snapshot.Labels[labels.AcornPublicName], snapshot.Annotations[annotationVolumeName], volumeName, -1),
				labels.AcornVolumeName:   volumeName,
				labels.AcornAppName:      snapshot.Labels[labels.AcornAppName],
				labels.AcornAppNamespace: snapshot.Labels[labels.AcornAppNamespace],
				labels.AcornVolumeClass:  snapshot.Annotations[annotationVolumeClass],
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: accessModes,
			Resources:   resources,
			DataSource: &corev1.TypedLocalObjectReference{
				Name:     snapshot.Name,
				Kind:     "VolumeSnapshot",
				APIGroup: pointer.String(snapshotv1.GroupName),
			},
		},
	}

	cl, err := c.GetClient()
	if err != nil {
		return err
	}

	return cl.Create(ctx, newPVC)
}
