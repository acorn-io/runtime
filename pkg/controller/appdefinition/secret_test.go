package appdefinition

import (
	"regexp"
	"strings"
	"testing"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSecretDirsToMounts(t *testing.T) {
	app := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "app",
		},
		Status: v1.AppInstanceStatus{
			AppImage: v1.AppImage{
				ID: "test",
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

	dep := ToDeploymentsTest(t, app, testTag, nil)[0].(*appsv1.Deployment)
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

func TestOpaque_Gen(t *testing.T) {
	h := tester.Harness{
		Scheme: scheme.Scheme,
	}
	resp, err := h.InvokeFunc(t, &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-name",
			Namespace: "app-ns",
		},
		Status: v1.AppInstanceStatus{
			Namespace: "app-target-ns",
			AppImage: v1.AppImage{
				ID: "test",
			},
			AppSpec: v1.AppSpec{
				Secrets: map[string]v1.Secret{
					"pass": {
						Type: "opaque",
						Data: map[string]string{
							"key1": "",
							"key2": "value",
						},
					},
				},
			},
		},
	}, CreateSecrets)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, resp.Client.Created, 1)
	assert.Len(t, resp.Collected, 2)

	secret := resp.Client.Created[0].(*corev1.Secret)
	assert.Equal(t, "pass", secret.Labels[labels.AcornSecretName])
	assert.True(t, strings.HasPrefix(secret.Name, "pass-"))
	_, ok := secret.Data["key1"]
	assert.True(t, ok)
	assert.True(t, len(secret.Data["key2"]) > 0)
}

func TestBasic_Gen(t *testing.T) {
	h := tester.Harness{
		Scheme: scheme.Scheme,
	}
	resp, err := h.InvokeFunc(t, &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-name",
			Namespace: "app-ns",
		},
		Status: v1.AppInstanceStatus{
			Namespace: "app-target-ns",
			AppImage: v1.AppImage{
				ID: "test",
			},
			AppSpec: v1.AppSpec{
				Secrets: map[string]v1.Secret{
					"pass": {Type: "basic",
						Data: map[string]string{
							// cue will populate empty string if not sent
							"username": "",
							"password": "",
						},
					},
					"passuname": {
						Type: "basic",
						Data: map[string]string{
							"username": "admin",
							"password": "",
						},
					},
				},
			},
		},
	}, CreateSecrets)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, resp.Client.Created, 2)
	assert.Len(t, resp.Collected, 3)

	secret := resp.Client.Created[0].(*corev1.Secret)
	assert.Equal(t, "pass", secret.Labels[labels.AcornSecretName])
	assert.True(t, strings.HasPrefix(secret.Name, "pass-"))
	assert.True(t, len(secret.Data["username"]) > 0)
	assert.True(t, len(secret.Data["password"]) > 0)

	secret = resp.Client.Created[1].(*corev1.Secret)
	assert.Equal(t, "passuname", secret.Labels[labels.AcornSecretName])
	assert.True(t, strings.HasPrefix(secret.Name, "passuname-"))
	assert.Equal(t, []byte("admin"), secret.Data["username"])
	assert.True(t, len(secret.Data["password"]) > 0)
}

func TestTemplateTokenMissing_Gen(t *testing.T) {
	h := tester.Harness{
		Scheme: scheme.Scheme,
	}
	resp, err := h.InvokeFunc(t, &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-name",
			Namespace: "app-ns",
		},
		Spec: v1.AppInstanceSpec{
			Image: "image",
		},
		Status: v1.AppInstanceStatus{
			Namespace: "app-target-ns",
			AppImage: v1.AppImage{
				ID: "image",
			},
			AppSpec: v1.AppSpec{
				Secrets: map[string]v1.Secret{
					"template": {
						Type: "template",
						Data: map[string]string{
							"template": "A happy little ${secret://pass/token} in a string",
						},
					},
				},
			},
		},
	}, CreateSecrets)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, resp.Client.Created, 0)
	assert.Len(t, resp.Collected, 1)

	app := resp.Collected[0].(*v1.AppInstance)
	assert.Equal(t, "missing: [pass]", app.Status.Condition("secrets").Message)
}

func TestTemplateToken_Gen(t *testing.T) {
	h := tester.Harness{
		Scheme: scheme.Scheme,
	}
	resp, err := h.InvokeFunc(t, &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-name",
			Namespace: "app-ns",
		},
		Spec: v1.AppInstanceSpec{
			Image: "image",
		},
		Status: v1.AppInstanceStatus{
			Namespace: "app-target-ns",
			AppImage: v1.AppImage{
				ID: "image",
			},
			AppSpec: v1.AppSpec{
				Secrets: map[string]v1.Secret{
					"pass": {Type: "token",
						Params: map[string]any{
							"characters": "abc",
							"length":     int64(5),
						},
					},
					"pass2": {Type: "token",
						Params: map[string]any{
							"characters": "xyz",
							"length":     int64(6),
						},
					},
					"template": {
						Type: "template",
						Data: map[string]string{
							"template": "A happy little ${secret://pass/token} in a string followed by ${secret://pass2/token}",
						},
					},
				},
			},
		},
	}, CreateSecrets)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, resp.Client.Created, 3)
	assert.Len(t, resp.Collected, 4)

	secret := resp.Client.Created[0].(*corev1.Secret)
	assert.Equal(t, "pass", secret.Labels[labels.AcornSecretName])
	assert.True(t, strings.HasPrefix(secret.Name, "pass-"))
	assert.True(t, len(secret.Data["token"]) == 5)
	assert.Len(t, regexp.MustCompile("[abc]").ReplaceAllString(string(secret.Data["token"]), ""), 0)

	secret2 := resp.Client.Created[1].(*corev1.Secret)
	assert.Equal(t, "pass2", secret2.Labels[labels.AcornSecretName])
	assert.True(t, strings.HasPrefix(secret2.Name, "pass2-"))
	assert.True(t, len(secret2.Data["token"]) == 6)
	assert.Len(t, regexp.MustCompile("[xyz]").ReplaceAllString(string(secret2.Data["token"]), ""), 0)

	secret3 := resp.Client.Created[2].(*corev1.Secret)
	assert.Equal(t, "template", secret3.Labels[labels.AcornSecretName])
	assert.True(t, strings.HasPrefix(secret3.Name, "template-"))
	assert.Equal(t, "A happy little "+string(secret.Data["token"])+
		" in a string followed by "+string(secret2.Data["token"]), string(secret3.Data["template"]))
}

func TestSecretRedeploy(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/secret", DeploySpec)
}

func TestSecretImageReference(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/secret-image", CreateSecrets)
}

func TestSecretEncrypted(t *testing.T) {
	resp := tester.DefaultTest(t, scheme.Scheme, "testdata/secret-encrypted", CreateSecrets)
	secret := resp.Client.Created[0].(*corev1.Secret)
	assert.Equal(t, "foo-", secret.GenerateName)
	assert.Equal(t, "app-namespace", secret.Namespace)
	assert.Equal(t, "ACORNENC:eyJzNmc2QWx2V05ER09MUnVkMWo2eVdoNHVUQndVU2NPa0ZJLUluYktYTXpvIjoiaTZ"+
		"DTl96TnpYM2wxYTVMaEdKTXpLalZnNlhPV2NZM0NYc21lQ2JETTNHWENySzBnSzVMdTg3bE45OGszcUdReGd6V1JSUHMifQ",
		string(secret.Data["key"]))
}

func TestSecretLabelsAnnotations(t *testing.T) {
	h := tester.Harness{
		Scheme: scheme.Scheme,
	}
	resp, err := h.InvokeFunc(t, &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-name",
			Namespace: "app-ns",

			// These SHOULDN'T propagate to the secret
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

				// --label secrets:allsec=val - All secrets
				{ResourceType: "secret", ResourceName: "", Key: "allsec", Value: "val"},

				// --label secret:secret1:sec1key=val - Type and name specified. Land on secret of same name
				{ResourceType: "secret", ResourceName: "secret1", Key: "sec1key", Value: "val"},

				// --label secret2:sec2key=val - No resourceType, but name specified. Land on secret of same name
				{ResourceType: "", ResourceName: "secret2", Key: "sec2key", Value: "val"},

				// --label containers:con=val - For containers, shouldn't land on secret
				{ResourceType: "container", ResourceName: "", Key: "con", Value: "val"},
			},
			Annotations: []v1.ScopedLabel{
				// --annotation globala=val - Apply to all resources
				{ResourceType: "", ResourceName: "", Key: "globala", Value: "val"},

				// --annotation secrets:allseca=val - All secrets
				{ResourceType: "secret", ResourceName: "", Key: "allseca", Value: "val"},

				// --annotation secret:secret1:sec1keya=val - Type and name specified. Land on secret of same name
				{ResourceType: "secret", ResourceName: "secret1", Key: "sec1keya", Value: "val"},

				// --annotation secret2:sec2keya=val - No resourceType, but name specified. Land on secret of same name
				{ResourceType: "", ResourceName: "secret2", Key: "sec2keya", Value: "val"},

				// --annotation containers:con=val - For containers, shouldn't land on secret
				{ResourceType: "container", ResourceName: "", Key: "con", Value: "val"},
			},
		},
		Status: v1.AppInstanceStatus{
			Namespace: "app-target-ns",
			AppImage: v1.AppImage{
				ID: "test",
			},
			AppSpec: v1.AppSpec{
				Labels: map[string]string{
					"globalfromacornfile": "val",
				},
				Annotations: map[string]string{
					"globalfromacornfilea": "val",
				},
				Secrets: map[string]v1.Secret{
					"secret1": {Type: "basic",
						Labels: map[string]string{
							"sec1fromacornfile": "val",
						},
						Annotations: map[string]string{
							"sec1fromacornfilea": "val",
						},
						Data: map[string]string{
							// cue will populate empty string if not sent
							"username": "",
							"password": "",
						},
					},
					"secret2": {
						Labels:      nil,
						Annotations: nil,
						Type:        "basic",
						Data: map[string]string{
							"username": "",
							"password": "",
						},
					},
				},
			},
		},
	}, CreateSecrets)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, resp.Client.Created, 2)
	assert.Len(t, resp.Collected, 3)

	secret := resp.Client.Created[0].(*corev1.Secret)
	assert.Equal(t, "secret1", secret.Labels[labels.AcornSecretName])
	assert.True(t, strings.HasPrefix(secret.Name, "secret1-"))
	// labels
	assert.Contains(t, secret.Labels, labels.AcornManaged) // prove we aren't stomping on the acorn.io labels
	assert.NotContains(t, secret.Labels, "fromapp")
	assert.Contains(t, secret.Labels, "global")
	assert.Contains(t, secret.Labels, "allsec")
	assert.Contains(t, secret.Labels, "sec1key")
	assert.NotContains(t, secret.Labels, "sec2key")
	assert.NotContains(t, secret.Labels, "con")
	assert.Contains(t, secret.Labels, "globalfromacornfile")
	assert.Contains(t, secret.Labels, "sec1fromacornfile")
	// annotations
	assert.NotContains(t, secret.Annotations, "fromapp")
	assert.Contains(t, secret.Annotations, "globala")
	assert.Contains(t, secret.Annotations, "allseca")
	assert.Contains(t, secret.Annotations, "sec1keya")
	assert.NotContains(t, secret.Annotations, "sec2keya")
	assert.NotContains(t, secret.Annotations, "con")
	assert.Contains(t, secret.Annotations, "globalfromacornfilea")
	assert.Contains(t, secret.Annotations, "sec1fromacornfilea")

	secret = resp.Client.Created[1].(*corev1.Secret)
	assert.Equal(t, "secret2", secret.Labels[labels.AcornSecretName])
	assert.True(t, strings.HasPrefix(secret.Name, "secret2-"))
	// Labels
	assert.Contains(t, secret.Labels, labels.AcornManaged)
	assert.NotContains(t, secret.Labels, "fromapp")
	assert.Contains(t, secret.Labels, "global")
	assert.Contains(t, secret.Labels, "allsec")
	assert.NotContains(t, secret.Labels, "sec1key")
	assert.Contains(t, secret.Labels, "sec2key")
	assert.NotContains(t, secret.Labels, "con")
	assert.Contains(t, secret.Labels, "globalfromacornfile")
	assert.NotContains(t, secret.Labels, "sec1fromacornfile")
	// Annotations
	assert.NotContains(t, secret.Annotations, "fromappa")
	assert.Contains(t, secret.Annotations, "globala")
	assert.Contains(t, secret.Annotations, "allseca")
	assert.NotContains(t, secret.Annotations, "sec1keya")
	assert.Contains(t, secret.Annotations, "sec2keya")
	assert.NotContains(t, secret.Annotations, "con")
	assert.Contains(t, secret.Annotations, "globalfromacornfilea")
	assert.NotContains(t, secret.Annotations, "sec1fromacornfilea")
}
