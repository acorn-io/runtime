package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	v1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	snapshotv1 "github.com/acorn-io/runtime/pkg/apis/snapshot.storage.k8s.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/z"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
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
	name, ok := pvc.Labels["acorn.io/custom-name"]
	if !ok {
		// generate a name if one wasn't given by the user
		name = pvc.Labels[labels.AcornPublicName] + "-" + strconv.FormatInt(time.Now().Unix(), 10)
	}

	var err error
	snapshotClass, ok := pvc.Labels["acorn.io/snapshot-class"]
	if !ok {
		// select the default snapshot class if one wasn't given by the user
		snapshotClass, err = getDefaultSnapshotClass(ctx, c, pvc.Labels[labels.AcornVolumeClass])
		if err != nil {
			return nil, err
		}
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
			VolumeSnapshotClassName: z.Pointer(snapshotClass),
			Source: snapshotv1.VolumeSnapshotSource{
				PersistentVolumeClaimName: z.Pointer(pvc.Name),
			},
		},
	}

	return snapshot, c.Client.Create(ctx, snapshot)
}

func getDefaultSnapshotClass(ctx context.Context, c *DefaultClient, volumeClass string) (string, error) {
	defaultSnapshotClasses := &snapshotv1.VolumeSnapshotClassList{}
	err := c.Client.List(ctx, defaultSnapshotClasses, &kclient.ListOptions{
		LabelSelector: klabels.SelectorFromSet(map[string]string{
			labels.AcornIsDefaultForStorageClass: volumeClass,
		}),
	})
	if err != nil {
		return "", err
	}

	if len(defaultSnapshotClasses.Items) > 1 {
		return "", errors.New("Multiple default snapshot classes found for storage class " + volumeClass + ".")
	}

	if len(defaultSnapshotClasses.Items) == 0 {
		return "", errors.New("No default snapshot class found for storage class " + volumeClass + ".")
	}

	return defaultSnapshotClasses.Items[0].Name, nil
}

// SnapshotList lists VolumeSnapshot. Returns []VolumeSnapshot or an error if one occurred.
func (c *DefaultClient) SnapshotList(ctx context.Context) ([]snapshotv1.VolumeSnapshot, error) {
	aggregatedSnapshots := make([]snapshotv1.VolumeSnapshot, 0)

	apps := &v1.AppList{}
	err := c.Client.List(ctx, apps, &kclient.ListOptions{
		Namespace: c.Namespace,
	})
	if err != nil {
		return nil, err
	}

	for _, app := range apps.Items {
		snapshots := &snapshotv1.VolumeSnapshotList{}
		err = c.Client.List(ctx, snapshots, &kclient.ListOptions{
			Namespace: app.Status.Namespace,
		})
		if err != nil {
			return nil, err
		}

		aggregatedSnapshots = append(aggregatedSnapshots, snapshots.Items...)
	}

	sort.Slice(aggregatedSnapshots, func(i, j int) bool {
		iTime := aggregatedSnapshots[i].CreationTimestamp.Time
		jTime := aggregatedSnapshots[j].CreationTimestamp.Time

		if iTime == jTime {
			return aggregatedSnapshots[i].Name < aggregatedSnapshots[j].Name
		}

		return iTime.After(jTime)
	})

	return aggregatedSnapshots, nil
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

	var resources corev1.VolumeResourceRequirements
	err = json.Unmarshal([]byte(snapshot.Annotations[annotationResources]), &resources)
	if err != nil {
		return err
	}

	_, err = c.VolumeGet(ctx, volumeName)
	if err == nil {
		return fmt.Errorf("a volume named %s already exists", volumeName)
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
				APIGroup: z.Pointer(snapshotv1.GroupName),
			},
		},
	}

	cl, err := c.GetClient()
	if err != nil {
		return err
	}

	return cl.Create(ctx, newPVC)
}
