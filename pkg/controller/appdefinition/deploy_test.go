package appdefinition

import (
	"encoding/hex"
	"strings"
	"testing"

	"cuelang.org/go/pkg/crypto/sha256"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	testTag = name.MustParseReference("test")
)

func TestDeploySpec(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/deployspec", DeploySpec)
}

func TestDeploySpecStop(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/deployspec-stop", DeploySpec)
}

func TestProbe(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/probes", DeploySpec)
}

func ToDeploymentsTest(t *testing.T, appInstance *v1.AppInstance, tag name.Reference, pullSecrets *PullSecrets) (result []kclient.Object) {
	req := tester.NewRequest(t, scheme.Scheme, appInstance)
	deps, err := ToDeployments(req, appInstance, tag, pullSecrets)
	if err != nil {
		t.Fatal(err)
	}
	return deps
}

func TestEntrypointCommand(t *testing.T) {
	dep := ToDeploymentsTest(t, &v1.AppInstance{
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
	}, testTag, nil)[0].(*appsv1.Deployment)
	assert.Equal(t, []string{"hi", "bye"}, dep.Spec.Template.Spec.Containers[0].Command)
	assert.Equal(t, []string{"hi2", "bye2"}, dep.Spec.Template.Spec.Containers[0].Args)
}

func TestEnvironment(t *testing.T) {
	dep := ToDeploymentsTest(t, &v1.AppInstance{
		Status: v1.AppInstanceStatus{
			AppSpec: v1.AppSpec{
				Containers: map[string]v1.Container{
					"test": {
						Environment: []v1.EnvVar{
							{
								Name:  "hi",
								Value: "bye",
							},
							{
								Name: "foo",
							},
						},
					},
				},
			},
		},
	}, testTag, nil)[0].(*appsv1.Deployment)
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
	dep := ToDeploymentsTest(t, &v1.AppInstance{
		Status: v1.AppInstanceStatus{
			AppSpec: v1.AppSpec{
				Containers: map[string]v1.Container{
					"test": {
						WorkingDir: "something",
					},
				},
			},
		},
	}, testTag, nil)[0].(*appsv1.Deployment)
	assert.Equal(t, "something", dep.Spec.Template.Spec.Containers[0].WorkingDir)
}

func TestInteractive(t *testing.T) {
	dep := ToDeploymentsTest(t, &v1.AppInstance{
		Status: v1.AppInstanceStatus{
			AppSpec: v1.AppSpec{
				Containers: map[string]v1.Container{
					"test": {
						Interactive: true,
					},
				},
			},
		},
	}, testTag, nil)[0].(*appsv1.Deployment)
	assert.True(t, dep.Spec.Template.Spec.Containers[0].TTY)
	assert.True(t, dep.Spec.Template.Spec.Containers[0].Stdin)
}

func TestSidecar(t *testing.T) {
	dep := ToDeploymentsTest(t, &v1.AppInstance{
		Status: v1.AppInstanceStatus{
			AppSpec: v1.AppSpec{
				Containers: map[string]v1.Container{
					"test": {
						Sidecars: map[string]v1.Container{
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
	}, testTag, nil)[0].(*appsv1.Deployment)
	assert.Equal(t, "sidecar", dep.Spec.Template.Spec.InitContainers[0].Image)
	assert.Equal(t, "sidecar2", dep.Spec.Template.Spec.Containers[1].Image)
}

func TestPorts(t *testing.T) {
	dep := ToDeploymentsTest(t, &v1.AppInstance{
		Status: v1.AppInstanceStatus{
			AppSpec: v1.AppSpec{
				Containers: map[string]v1.Container{
					"test": {
						Sidecars: map[string]v1.Container{
							"left": {
								Ports: []v1.PortDef{
									{
										Port:         90,
										InternalPort: 91,
										Protocol:     v1.ProtocolHTTP,
									},
								},
							},
						},
						WorkingDir: "something",
						Ports: []v1.PortDef{
							{
								Port:         80,
								InternalPort: 81,
								Protocol:     v1.ProtocolHTTP,
							},
						},
					},
				},
			},
		},
	}, testTag, nil)[0].(*appsv1.Deployment)
	assert.Equal(t, int32(81), dep.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)
	assert.Equal(t, corev1.ProtocolTCP, dep.Spec.Template.Spec.Containers[0].Ports[0].Protocol)
	assert.Equal(t, int32(91), dep.Spec.Template.Spec.Containers[1].Ports[0].ContainerPort)
	assert.Equal(t, corev1.ProtocolTCP, dep.Spec.Template.Spec.Containers[1].Ports[0].Protocol)
}

func TestFiles(t *testing.T) {
	app := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "app",
		},
		Status: v1.AppInstanceStatus{
			AppSpec: v1.AppSpec{
				Containers: map[string]v1.Container{
					"test2": {
						Files: map[string]v1.File{
							"/a2/b/c": {
								Content: "ZA==",
							},
							"/a1/b/c": {
								Content: "ZQ==",
							},
						},
						Sidecars: map[string]v1.Container{
							"left": {
								Files: map[string]v1.File{
									"/a/b2//c":      {Content: "ZA=="},
									"/a/b1/c2/../c": {Content: "ZQ=="},
								},
							},
						},
					},
					"test": {
						Files: map[string]v1.File{
							"/a2/b/c": {
								Content: "ZA==",
							},
							"/a1/b/c": {
								Content: "ZQ==",
							},
						},
						Sidecars: map[string]v1.Container{
							"left": {
								Files: map[string]v1.File{
									"/a/b2//c":      {Content: "ZA=="},
									"/a/b1/c2/../c": {Content: "ZQ=="},
								},
							},
						},
					},
				},
			},
		},
	}

	dep := ToDeploymentsTest(t, app, testTag, nil)[0].(*appsv1.Deployment)

	assert.Equal(t, "files", dep.Spec.Template.Spec.Containers[0].VolumeMounts[0].Name)
	assert.Equal(t, "/a1/b/c", dep.Spec.Template.Spec.Containers[0].VolumeMounts[0].MountPath)
	assert.Equal(t, toPathHash("/app/test/test/a1/b/c"), dep.Spec.Template.Spec.Containers[0].VolumeMounts[0].SubPath)
	assert.Equal(t, "files", dep.Spec.Template.Spec.Containers[0].VolumeMounts[1].Name)
	assert.Equal(t, "/a2/b/c", dep.Spec.Template.Spec.Containers[0].VolumeMounts[1].MountPath)
	assert.Equal(t, toPathHash("/app/test/test/a2/b/c"), dep.Spec.Template.Spec.Containers[0].VolumeMounts[1].SubPath)

	assert.Equal(t, "files", dep.Spec.Template.Spec.Containers[1].VolumeMounts[0].Name)
	assert.Equal(t, "/a/b1/c", dep.Spec.Template.Spec.Containers[1].VolumeMounts[0].MountPath)
	assert.Equal(t, toPathHash("/app/test/left/a/b1/c"), dep.Spec.Template.Spec.Containers[1].VolumeMounts[0].SubPath)
	assert.Equal(t, "files", dep.Spec.Template.Spec.Containers[1].VolumeMounts[1].Name)
	assert.Equal(t, "/a/b2/c", dep.Spec.Template.Spec.Containers[1].VolumeMounts[1].MountPath)
	assert.Equal(t, toPathHash("/app/test/left/a/b2/c"), dep.Spec.Template.Spec.Containers[1].VolumeMounts[1].SubPath)

	configMaps, err := toConfigMaps(app)
	if err != nil {
		t.Fatal(err)
	}
	configMap := configMaps[0].(*corev1.ConfigMap)

	assert.Len(t, configMap.BinaryData, 8)
	assert.Equal(t, []byte("d"), configMap.BinaryData[toPathHash("/app/test/test/a2/b/c")])
	assert.Equal(t, []byte("d"), configMap.BinaryData[toPathHash("/app/test2/test2/a2/b/c")])
	assert.Equal(t, []byte("d"), configMap.BinaryData[toPathHash("/app/test/left/a/b2/c")])
	assert.Equal(t, []byte("d"), configMap.BinaryData[toPathHash("/app/test2/left/a/b2/c")])
	assert.Equal(t, []byte("e"), configMap.BinaryData[toPathHash("/app/test/test/a1/b/c")])
	assert.Equal(t, []byte("e"), configMap.BinaryData[toPathHash("/app/test/left/a/b1/c")])
	assert.Equal(t, []byte("e"), configMap.BinaryData[toPathHash("/app/test2/test2/a1/b/c")])
	assert.Equal(t, []byte("e"), configMap.BinaryData[toPathHash("/app/test2/left/a/b1/c")])
}

func toPathHash(path string) string {
	path = strings.TrimPrefix(path, "/")
	return hex.EncodeToString(sha256.Sum256([]byte(path))[:])[:12]
}
