package appdefinition

import (
	"testing"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSecretDirsToMounts(t *testing.T) {
	app := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "app",
		},
		Status: v1.AppInstanceStatus{
			AppImage: v1.AppImage{
				Name: "test",
			},
			AppSpec: v1.AppSpec{
				Containers: map[string]v1.Container{
					"test": {
						Dirs: map[string]v1.VolumeMount{
							"/dir": {
								Secret: v1.VolumeSecretMount{
									Name: "dir-secret",
								},
							},
						},
						Sidecars: map[string]v1.Container{
							"left": {
								Dirs: map[string]v1.VolumeMount{
									"/dir-side": {
										Secret: v1.VolumeSecretMount{
											Name: "dir-side-secret",
										},
									},
								},
							},
						},
					},
				},
				Secrets: map[string]v1.Secret{
					"dir-side-secret": {},
				},
			},
		},
	}

	dep := ToDeploymentsTest(t, app, testTag, nil)[1].(*appsv1.Deployment)
	assert.Equal(t, "/dir", dep.Spec.Template.Spec.Containers[0].VolumeMounts[0].MountPath)
	assert.Equal(t, "secret--dir-secret", dep.Spec.Template.Spec.Containers[0].VolumeMounts[0].Name)
	assert.Equal(t, "/dir-side", dep.Spec.Template.Spec.Containers[1].VolumeMounts[0].MountPath)
	assert.Equal(t, "secret--dir-side-secret", dep.Spec.Template.Spec.Containers[1].VolumeMounts[0].Name)
	assert.Equal(t, "secret--dir-side-secret", dep.Spec.Template.Spec.Containers[1].VolumeMounts[0].Name)
	assert.Equal(t, "secret--dir-secret", dep.Spec.Template.Spec.Volumes[0].Name)
	assert.Equal(t, "dir-secret", dep.Spec.Template.Spec.Volumes[0].Secret.SecretName)
	assert.Equal(t, "secret--dir-side-secret", dep.Spec.Template.Spec.Volumes[1].Name)
	assert.Equal(t, "dir-side-secret", dep.Spec.Template.Spec.Volumes[1].Secret.SecretName)
}

func TestSecretRedeploy(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/secret", DeploySpec)
}
