package appdefinition

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestVolumeController(t *testing.T) {
	dirs, err := ioutil.ReadDir("testdata/volumes")
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range dirs {
		tester.DefaultTest(t, scheme.Scheme, filepath.Join("testdata/volumes", dir.Name()), DeploySpec)
	}
}

func TestVolumes(t *testing.T) {
	app := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "app",
		},
		Spec: v1.AppInstanceSpec{
			Volumes: []v1.VolumeBinding{
				{
					Volume:        "v1-random",
					VolumeRequest: "v1",
				},
				{
					Volume:        "v2-random",
					VolumeRequest: "v2",
				},
			},
		},
		Status: v1.AppInstanceStatus{
			AppSpec: v1.AppSpec{
				Volumes: map[string]v1.VolumeRequest{
					"v1": {
						Class: "v1-class",
						Size:  5,
					},
					"v2": {
						Size: 10,
					},
					"v3": {
						Class: "ephemeral",
						Size:  15,
					},
					"v4": {
						Size: 20,
					},
				},
				Containers: map[string]v1.Container{
					"test": {
						Dirs: map[string]v1.VolumeMount{
							"/asdf": {
								Volume: "v1",
							},
							"/qwerty": {
								Volume: "v2",
							},
						},
						Sidecars: map[string]v1.Container{
							"left": {
								Dirs: map[string]v1.VolumeMount{
									"/as-df": {
										Volume: "v3",
									},
									"/qwe-rty": {
										Volume: "v4",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	dep := ToDeployments(app, testTag, nil)[0].(*appsv1.Deployment)

	assert.Equal(t, "v1-bind", dep.Spec.Template.Spec.Volumes[0].PersistentVolumeClaim.ClaimName)
	assert.Equal(t, "v2-bind", dep.Spec.Template.Spec.Volumes[1].PersistentVolumeClaim.ClaimName)
	assert.Nil(t, dep.Spec.Template.Spec.Volumes[2].PersistentVolumeClaim)
	assert.NotNil(t, dep.Spec.Template.Spec.Volumes[2].EmptyDir)
	assert.Equal(t, "v4", dep.Spec.Template.Spec.Volumes[3].PersistentVolumeClaim.ClaimName)

	pvs := toPVCs(app)
	v1 := pvs[0].(*corev1.PersistentVolumeClaim)
	v2 := pvs[1].(*corev1.PersistentVolumeClaim)
	v4 := pvs[2].(*corev1.PersistentVolumeClaim)

	assert.Equal(t, "v1-bind", v1.Name)
	assert.Equal(t, "v1-random", v1.Spec.VolumeName)
	assert.Equal(t, "v2-bind", v2.Name)
	assert.Equal(t, "v2-random", v2.Spec.VolumeName)
	assert.Equal(t, "v4", v4.Name)
	assert.Nil(t, v4.Spec.StorageClassName)
	req := v4.Spec.Resources.Requests[corev1.ResourceStorage]
	assert.Equal(t, int64(20000000000), req.Value())
}
