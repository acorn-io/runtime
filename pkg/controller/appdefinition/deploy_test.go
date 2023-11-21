package appdefinition

import (
	"encoding/base64"
	"testing"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/router/tester"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/controller/namespace"
	"github.com/acorn-io/runtime/pkg/digest"
	"github.com/acorn-io/runtime/pkg/scheme"
	"github.com/acorn-io/runtime/pkg/secrets"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	testTag = name.MustParseReference("test")
)

func TestTemplate(t *testing.T) {
	// This is the basic hello world all other tests can start from
	tester.DefaultTest(t, scheme.Scheme, "testdata/template", DeploySpec)
}

func TestGlobalEnv(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/globalenv", DeploySpec)
}

func TestDeploySpec(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/deployspec/basic", DeploySpec)
}

func TestDeploySpecUserDefinedLabelsAnnotations(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/deployspec/labels", FilterLabelsAndAnnotationsConfig(router.HandlerFunc(DeploySpec)).Handle)
}

func TestDeploySpecUserDefinedLabelsAnnotationsNamespace(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/deployspec/labels-namespace", namespace.AddNamespace)
}

func TestDeploySpecIgnoreUserDefinedLabelsAnnotations(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/deployspec/no-user-labels", FilterLabelsAndAnnotationsConfig(router.HandlerFunc(DeploySpec)).Handle)
}

func TestDeploySpecIgnoreUserDefinedLabelsAnnotationsNamespace(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/deployspec/no-user-labels-namespace", FilterLabelsAndAnnotationsConfig(router.HandlerFunc(namespace.AddNamespace)).Handle)
}

func TestDeploySpecFilterUserDefinedLabelsAnnotations(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/deployspec/filter-user-labels", FilterLabelsAndAnnotationsConfig(router.HandlerFunc(DeploySpec)).Handle)
}

func TestDeploySpecFilterUserDefinedLabelsAnnotationsNamespace(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/deployspec/filter-user-labels-namespace", FilterLabelsAndAnnotationsConfig(router.HandlerFunc(namespace.AddNamespace)).Handle)
}

func TestDeploySpecScale(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/deployspec/scale", DeploySpec)
}

func TestDeploySpecStop(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/deployspec/stop", DeploySpec)
}

func TestDeploySpecMetrics(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/deployspec/metrics", DeploySpec)
}

func TestProbe(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/probes", DeploySpec)
}

func TestKarpenterAnnotation(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/deployspec/karpenter", DeploySpec)
}

func ToDeploymentsTest(t *testing.T, appInstance *v1.AppInstance, tag name.Reference, pullSecrets *PullSecrets) (result []kclient.Object) {
	t.Helper()

	req := tester.NewRequest(t, scheme.Scheme, appInstance)
	interpolator := secrets.NewInterpolator(req.Ctx, req.Client, appInstance)
	deps, err := ToDeployments(req, appInstance, tag, pullSecrets, interpolator)
	if err != nil {
		t.Fatal(err)
	}
	if err := interpolator.Err(); err != nil {
		t.Fatal(err)
	}
	return append(deps, interpolator.Objects()...)
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
	}, testTag, nil)[1].(*appsv1.Deployment)
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
	}, testTag, nil)[1].(*appsv1.Deployment)
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
	}, testTag, nil)[1].(*appsv1.Deployment)
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
	}, testTag, nil)[1].(*appsv1.Deployment)
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
	}, testTag, nil)[1].(*appsv1.Deployment)
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
										Port:       90,
										TargetPort: 91,
										Protocol:   v1.ProtocolHTTP,
									},
								},
							},
						},
						WorkingDir: "something",
						Ports: []v1.PortDef{
							{
								Port:       80,
								TargetPort: 81,
								Protocol:   v1.ProtocolHTTP,
							},
						},
					},
				},
			},
		},
	}, testTag, nil)[1].(*appsv1.Deployment)
	assert.Equal(t, int32(81), dep.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)
	assert.Equal(t, corev1.ProtocolTCP, dep.Spec.Template.Spec.Containers[0].Ports[0].Protocol)
	assert.Equal(t, int32(91), dep.Spec.Template.Spec.Containers[1].Ports[0].ContainerPort)
	assert.Equal(t, corev1.ProtocolTCP, dep.Spec.Template.Spec.Containers[1].Ports[0].Protocol)
}

func TestFiles(t *testing.T) {
	app := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "app",
			UID:  "123",
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

	objs := ToDeploymentsTest(t, app, testTag, nil)
	dep := objs[1].(*appsv1.Deployment)

	toHash := func(s string) string {
		t.Helper()
		data, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			t.Fatal(err)
		}
		return digest.SHA256(string(data))
	}

	assert.Equal(t, "secrets-123", dep.Spec.Template.Spec.Containers[0].VolumeMounts[0].Name)
	assert.Equal(t, "/a1/b/c", dep.Spec.Template.Spec.Containers[0].VolumeMounts[0].MountPath)
	assert.Equal(t, toHash("ZQ=="), dep.Spec.Template.Spec.Containers[0].VolumeMounts[0].SubPath)
	assert.Equal(t, "secrets-123", dep.Spec.Template.Spec.Containers[0].VolumeMounts[1].Name)
	assert.Equal(t, "/a2/b/c", dep.Spec.Template.Spec.Containers[0].VolumeMounts[1].MountPath)
	assert.Equal(t, toHash("ZA=="), dep.Spec.Template.Spec.Containers[0].VolumeMounts[1].SubPath)

	assert.Equal(t, "secrets-123", dep.Spec.Template.Spec.Containers[1].VolumeMounts[0].Name)
	assert.Equal(t, "/a/b1/c", dep.Spec.Template.Spec.Containers[1].VolumeMounts[0].MountPath)
	assert.Equal(t, toHash("ZQ=="), dep.Spec.Template.Spec.Containers[1].VolumeMounts[0].SubPath)
	assert.Equal(t, "secrets-123", dep.Spec.Template.Spec.Containers[1].VolumeMounts[1].Name)
	assert.Equal(t, "/a/b2/c", dep.Spec.Template.Spec.Containers[1].VolumeMounts[1].MountPath)
	assert.Equal(t, toHash("ZA=="), dep.Spec.Template.Spec.Containers[1].VolumeMounts[1].SubPath)

	configMap := objs[6].(*corev1.Secret)

	assert.Len(t, configMap.Data, 2)
	assert.Equal(t, []byte("d"), configMap.Data[toHash("ZA==")])
	assert.Equal(t, []byte("e"), configMap.Data[toHash("ZQ==")])
}

func TestUserContext(t *testing.T) {
	app := &v1.AppInstance{
		Status: v1.AppInstanceStatus{
			AppSpec: v1.AppSpec{
				Containers: map[string]v1.Container{
					"foo": {
						Image: "foo:latest",
						UserContext: &v1.UserContext{
							UID: 1000,
							GID: 2000,
						},
						Sidecars: map[string]v1.Container{
							"bar": {
								Image: "bar:latest",
								UserContext: &v1.UserContext{
									UID: 3000,
									GID: 4000,
								},
							},
						},
					},
				},
			},
		},
	}

	objs := ToDeploymentsTest(t, app, testTag, nil)
	dep := objs[1].(*appsv1.Deployment)

	require.Equal(t, int64(1000), *dep.Spec.Template.Spec.Containers[0].SecurityContext.RunAsUser)
	require.Equal(t, int64(2000), *dep.Spec.Template.Spec.Containers[0].SecurityContext.RunAsGroup)
	require.Equal(t, int64(3000), *dep.Spec.Template.Spec.Containers[1].SecurityContext.RunAsUser)
	require.Equal(t, int64(4000), *dep.Spec.Template.Spec.Containers[1].SecurityContext.RunAsGroup)
}
