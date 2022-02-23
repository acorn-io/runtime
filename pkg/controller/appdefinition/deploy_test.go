package appdefinition

import (
	"testing"

	"github.com/ibuildthecloud/baaah/pkg/router/tester"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/scheme"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestDeploySpec(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/deployspec", DeploySpec)
}

func TestDeploySpecStop(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/deployspec-stop", DeploySpec)
}

func TestEntrypointCommand(t *testing.T) {
	dep := toDeployments(&v1.AppInstance{
		Status: v1.AppInstanceStatus{
			AppSpec: v1.AppSpec{
				Containers: map[string]v1.Container{
					"test": {
						Entrypoint: []string{"hi", "bye"},
						Command:    []string{"hi2", "bye2"},
					},
				},
			},
		},
	})[0].(*appsv1.Deployment)
	assert.Equal(t, []string{"hi", "bye"}, dep.Spec.Template.Spec.Containers[0].Command)
	assert.Equal(t, []string{"hi2", "bye2"}, dep.Spec.Template.Spec.Containers[0].Args)
}

func TestEnvironment(t *testing.T) {
	dep := toDeployments(&v1.AppInstance{
		Status: v1.AppInstanceStatus{
			AppSpec: v1.AppSpec{
				Containers: map[string]v1.Container{
					"test": {
						Environment: []string{
							"hi=bye",
							"foo",
						},
					},
				},
			},
		},
	})[0].(*appsv1.Deployment)
	assert.Equal(t, []corev1.EnvVar{
		{
			Name:  "hi",
			Value: "bye",
		},
		{
			Name:  "foo",
			Value: "",
		},
	}, dep.Spec.Template.Spec.Containers[0].Env)
}

func TestWorkdir(t *testing.T) {
	dep := toDeployments(&v1.AppInstance{
		Status: v1.AppInstanceStatus{
			AppSpec: v1.AppSpec{
				Containers: map[string]v1.Container{
					"test": {
						WorkingDir: "something",
					},
				},
			},
		},
	})[0].(*appsv1.Deployment)
	assert.Equal(t, "something", dep.Spec.Template.Spec.Containers[0].WorkingDir)
}

func TestInteractive(t *testing.T) {
	dep := toDeployments(&v1.AppInstance{
		Status: v1.AppInstanceStatus{
			AppSpec: v1.AppSpec{
				Containers: map[string]v1.Container{
					"test": {
						Interactive: true,
					},
				},
			},
		},
	})[0].(*appsv1.Deployment)
	assert.True(t, dep.Spec.Template.Spec.Containers[0].TTY)
	assert.True(t, dep.Spec.Template.Spec.Containers[0].Stdin)
}

func TestSidecar(t *testing.T) {
	dep := toDeployments(&v1.AppInstance{
		Status: v1.AppInstanceStatus{
			AppSpec: v1.AppSpec{
				Containers: map[string]v1.Container{
					"test": {
						Sidecars: map[string]v1.Sidecar{
							"left": {
								Image: "sidecar",
								Init:  true,
							},
							"right": {
								Image: "sidecar2",
							},
						},
						WorkingDir: "something",
					},
				},
			},
		},
	})[0].(*appsv1.Deployment)
	assert.Equal(t, "sidecar", dep.Spec.Template.Spec.InitContainers[0].Image)
	assert.Equal(t, "sidecar2", dep.Spec.Template.Spec.Containers[1].Image)
}
