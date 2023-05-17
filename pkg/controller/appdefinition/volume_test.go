package appdefinition

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestVolumeController(t *testing.T) {
	dirs, err := os.ReadDir("testdata/volumes")
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range dirs {
		tester.DefaultTest(t, scheme.Scheme, filepath.Join("testdata/volumes", dir.Name()), DeploySpec)
	}
}

func TestVolumeLabelsAnnotations(t *testing.T) {
	h := tester.Harness{
		Scheme: scheme.Scheme,
	}
	resp, err := h.InvokeFunc(t, &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-name",
			Namespace: "app-ns",

			// These SHOULDN'T propagate to the volume
			Annotations: map[string]string{
				"fromapp": "val",
			},
			Labels: map[string]string{
				"fromapp": "val",
			},
		},
		Spec: v1.AppInstanceSpec{
			Labels: []v1.ScopedLabel{
				// --label global=val - Apply to all resources
				{ResourceType: "", ResourceName: "", Key: "global", Value: "val"},

				// --label volumes:allvol=val - All volumes
				{ResourceType: "volume", ResourceName: "", Key: "allvol", Value: "val"},

				// --label volumees:vol:vol1key=val - Type and name specified. Land on volume of same name
				{ResourceType: "volume", ResourceName: "volume1", Key: "vol1key", Value: "val"},

				// --label volume2:vol2key=val - No resourceType, but name specified. Land on volume of same name
				{ResourceType: "", ResourceName: "volume2", Key: "vol2key", Value: "val"},

				// --label containers:con=val - For containers, shouldn't land on volume
				{ResourceType: "container", ResourceName: "", Key: "con", Value: "val"},
			},
			Annotations: []v1.ScopedLabel{
				// --annotation globala=val - Apply to all resources
				{ResourceType: "", ResourceName: "", Key: "globala", Value: "val"},

				// --annotation volumes:allvola=val - All volumes
				{ResourceType: "volume", ResourceName: "", Key: "allvola", Value: "val"},

				// --annotation volume:volume1:vol1keya=val - Type and name specified. Land on volume of same name
				{ResourceType: "volume", ResourceName: "volume1", Key: "vol1keya", Value: "val"},

				// --annotation volume2:vol2keya=val - No resourceType, but name specified. Land on volume of same name
				{ResourceType: "", ResourceName: "volume2", Key: "vol2keya", Value: "val"},

				// --annotation containers:con=val - For containers, shouldn't land on volume
				{ResourceType: "container", ResourceName: "", Key: "con", Value: "val"},
			},
			Image: "image",
		},
		Status: v1.AppInstanceStatus{
			Namespace: "app-target-ns",
			AppImage: v1.AppImage{
				ID: "image",
			},
			AppSpec: v1.AppSpec{
				Labels: map[string]string{
					"globalfromacornfile": "val",
				},
				Annotations: map[string]string{
					"globalfromacornfilea": "val",
				},
				Volumes: map[string]v1.VolumeRequest{
					"volume1": {
						Labels: map[string]string{
							"vol1fromacornfile": "val",
						},
						Annotations: map[string]string{
							"vol1fromacornfilea": "val",
						},
						AccessModes: []v1.AccessMode{v1.AccessModeReadWriteOnce},
					},
					"volume2": {
						Labels:      nil,
						Annotations: nil,
						AccessModes: []v1.AccessMode{v1.AccessModeReadWriteOnce},
					},
				},
			},
		},
	}, DeploySpec)
	if err != nil {
		t.Fatal(err)
	}

	var pvc1, pvc2 *corev1.PersistentVolumeClaim
	for _, i := range resp.Collected {
		if i.GetName() == "volume1" {
			pvc1 = i.(*corev1.PersistentVolumeClaim)
		} else if i.GetName() == "volume2" {
			pvc2 = i.(*corev1.PersistentVolumeClaim)
		}
	}
	assert.NotNil(t, pvc1)
	assert.NotNil(t, pvc2)

	assert.True(t, strings.HasPrefix(pvc1.Name, "volume1"))
	// labels
	assert.Contains(t, pvc1.Labels, labels.AcornManaged) // prove we aren't stomping on the acorn.io labels
	assert.NotContains(t, pvc1.Labels, "fromapp")
	assert.Contains(t, pvc1.Labels, "global")
	assert.Contains(t, pvc1.Labels, "allvol")
	assert.Contains(t, pvc1.Labels, "vol1key")
	assert.NotContains(t, pvc1.Labels, "vol2key")
	assert.NotContains(t, pvc1.Labels, "con")
	assert.Contains(t, pvc1.Labels, "globalfromacornfile")
	assert.Contains(t, pvc1.Labels, "vol1fromacornfile")
	// annotations
	assert.NotContains(t, pvc1.Annotations, "fromapp")
	assert.Contains(t, pvc1.Annotations, "globala")
	assert.Contains(t, pvc1.Annotations, "allvola")
	assert.Contains(t, pvc1.Annotations, "vol1keya")
	assert.NotContains(t, pvc1.Annotations, "vol2keya")
	assert.NotContains(t, pvc1.Annotations, "con")
	assert.Contains(t, pvc1.Annotations, "globalfromacornfilea")
	assert.Contains(t, pvc1.Annotations, "vol1fromacornfilea")

	assert.True(t, strings.HasPrefix(pvc2.Name, "volume2"))
	// Labels
	assert.Contains(t, pvc2.Labels, labels.AcornManaged)
	assert.NotContains(t, pvc2.Labels, "fromapp")
	assert.Contains(t, pvc2.Labels, "global")
	assert.Contains(t, pvc2.Labels, "allvol")
	assert.NotContains(t, pvc2.Labels, "vol1key")
	assert.Contains(t, pvc2.Labels, "vol2key")
	assert.NotContains(t, pvc2.Labels, "con")
	assert.Contains(t, pvc2.Labels, "globalfromacornfile")
	assert.NotContains(t, pvc2.Labels, "vol1fromacornfile")
	// Annotations
	assert.NotContains(t, pvc2.Annotations, "fromappa")
	assert.Contains(t, pvc2.Annotations, "globala")
	assert.Contains(t, pvc2.Annotations, "allvola")
	assert.NotContains(t, pvc2.Annotations, "vol1keya")
	assert.Contains(t, pvc2.Annotations, "vol2keya")
	assert.NotContains(t, pvc2.Annotations, "con")
	assert.Contains(t, pvc2.Annotations, "globalfromacornfilea")
	assert.NotContains(t, pvc2.Annotations, "vol1fromacornfilea")
}

func TestFindPVForBinding(t *testing.T) {
	pvList := buildPVs(t)
	err := scheme.AddToScheme(scheme.Scheme)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name                string
		appInstance         v1.AppInstance
		volumeBinding       v1.VolumeBinding
		expectedPVName      string
		expectNotFoundError bool
		extraPV             *corev1.PersistentVolume
	}{
		{
			name: "Bind PV created outside of Acorn",
			appInstance: v1.AppInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myApp",
					Namespace: "proj1",
				},
			},
			volumeBinding: v1.VolumeBinding{
				Volume: "external-pv",
				Target: "targetVol",
			},
			expectedPVName: "external-pv",
		},
		{
			name: "Bind nonexistent PV",
			appInstance: v1.AppInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myApp",
					Namespace: "proj1",
				},
			},
			volumeBinding: v1.VolumeBinding{
				Volume: "dne",
				Target: "targetVol",
			},
			expectNotFoundError: true,
		},
		{
			name: "Bind PV created by another app, same project",
			appInstance: v1.AppInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myApp",
					Namespace: "proj1",
				},
			},
			volumeBinding: v1.VolumeBinding{
				Volume: "app2.volume",
				Target: "targetVol",
			},
			expectedPVName: "pv2",
		},
		{
			name: "Bind PV created by another app, different project",
			appInstance: v1.AppInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myApp",
					Namespace: "proj1",
				},
			},
			volumeBinding: v1.VolumeBinding{
				Volume: "app3.volume",
				Target: "targetVol",
			},
			expectNotFoundError: true,
		},
		{
			name: "Bind PV created outside of Acorn without the Acorn managed label",
			appInstance: v1.AppInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myApp",
					Namespace: "proj1",
				},
			},
			volumeBinding: v1.VolumeBinding{
				Volume: "not-acorn-managed",
				Target: "targetVol",
			},
			expectNotFoundError: true,
		},
		{
			name: "Bind PV that was already bound and changed to the new public name",
			appInstance: v1.AppInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myApp",
					Namespace: "proj1",
				},
			},
			volumeBinding: v1.VolumeBinding{
				Volume: "app4.volume",
				Target: "targetVol",
			},
			extraPV: &corev1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pv4",
					Labels: map[string]string{
						labels.AcornAppNamespace: "proj1",
						labels.AcornPublicName:   "myApp.targetVol",
						labels.AcornManaged:      "true",
					},
				},
			},
			expectedPVName: "pv4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tester.NewRequest(t, scheme.Scheme, &tt.appInstance, pvList...)

			if tt.extraPV != nil {
				if err := req.Client.Create(req.Ctx, tt.extraPV); err != nil {
					t.Fatal(err)
				}
			}

			pv, err := getPVForVolumeBinding(req, &tt.appInstance, tt.volumeBinding)
			if err != nil {
				if (apierrors.IsNotFound(err) && !tt.expectNotFoundError) || !apierrors.IsNotFound(err) {
					t.Fatalf("Unexpected error: %v", err)
				}
			} else if tt.expectNotFoundError {
				t.Fatalf("Found PersistentVolume [%s] when none was expected", pv.Name)
			} else if pv.Name != tt.expectedPVName {
				t.Fatalf("Expected PersistentVolume [%s] but found [%s]", tt.expectedPVName, pv.Name)
			}
		})
	}
}

func buildPVs(t *testing.T) []kclient.Object {
	t.Helper()

	pvList := []corev1.PersistentVolume{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pv1",
				Labels: map[string]string{
					labels.AcornAppNamespace: "proj1",
					labels.AcornPublicName:   "app1.volume",
					labels.AcornManaged:      "true",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pv2",
				Labels: map[string]string{
					labels.AcornAppNamespace: "proj1",
					labels.AcornPublicName:   "app2.volume",
					labels.AcornManaged:      "true",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pv3",
				Labels: map[string]string{
					labels.AcornAppNamespace: "other-project",
					labels.AcornPublicName:   "app3.volume",
					labels.AcornManaged:      "true",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "external-pv",
				Labels: map[string]string{
					labels.AcornManaged: "true",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "not-acorn-managed",
				Labels: map[string]string{},
			},
		},
	}

	// I don't know of a better way to convert this to generic kclient.Object
	var objList []kclient.Object
	for _, pv := range pvList {
		objList = append(objList, pv.DeepCopy())
	}

	return objList
}
