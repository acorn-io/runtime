package appdefinition

import (
	"crypto/x509"
	"regexp"
	"strings"
	"testing"

	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/certs"
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
					"dir-side-secret": {
						Optional: &[]bool{true}[0],
					},
				},
			},
		},
	}

	dep := toDeployments(app, testTag, nil)[0].(*appsv1.Deployment)
	assert.Equal(t, "/dir", dep.Spec.Template.Spec.Containers[0].VolumeMounts[0].MountPath)
	assert.Equal(t, "secret--dir-secret", dep.Spec.Template.Spec.Containers[0].VolumeMounts[0].Name)
	assert.Equal(t, "/dir-side", dep.Spec.Template.Spec.Containers[1].VolumeMounts[0].MountPath)
	assert.Equal(t, "secret--dir-side-secret", dep.Spec.Template.Spec.Containers[1].VolumeMounts[0].Name)
	assert.Equal(t, "secret--dir-side-secret", dep.Spec.Template.Spec.Containers[1].VolumeMounts[0].Name)
	assert.Equal(t, "secret--dir-secret", dep.Spec.Template.Spec.Volumes[0].Name)
	assert.Equal(t, "dir-secret", dep.Spec.Template.Spec.Volumes[0].Secret.SecretName)
	assert.Equal(t, "secret--dir-side-secret", dep.Spec.Template.Spec.Volumes[1].Name)
	assert.Equal(t, "dir-side-secret", dep.Spec.Template.Spec.Volumes[1].Secret.SecretName)
	assert.Equal(t, true, *dep.Spec.Template.Spec.Volumes[1].Secret.Optional)
	assert.Equal(t, false, *dep.Spec.Template.Spec.Volumes[0].Secret.Optional)
}

func TestTLSGen(t *testing.T) {
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
			AppSpec: v1.AppSpec{
				Secrets: map[string]v1.Secret{
					"tls": {
						Type: "tls",
						Params: map[string]interface{}{
							"algorithm":    "rsa",
							"usage":        "client",
							"commonName":   "cn",
							"organization": []string{"org"},
							"sans":         []string{"san1", "192.168.1.1"},
							"durationDays": 2,
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
	assert.Equal(t, "app-ns", secret.Namespace)
	assert.True(t, strings.HasPrefix(secret.Name, "tls-"))
	assert.False(t, strings.Contains(secret.Name, "--"))
	assert.True(t, len(secret.Data[corev1.TLSCertKey]) > 0)
	assert.True(t, len(secret.Data[corev1.TLSPrivateKeyKey]) > 0)

	cert, _, err := certs.ParseCert(secret.Data[corev1.TLSCertKey])
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, x509.RSA, cert.PublicKeyAlgorithm)
	assert.Equal(t, "cn", cert.Subject.CommonName)
	assert.Equal(t, "org", cert.Subject.Organization[0])
	assert.Equal(t, "san1", cert.DNSNames[0])
	assert.Equal(t, "192.168.1.1", cert.IPAddresses[0].String())

	targetSecret := resp.Collected[0].(*corev1.Secret)
	assert.Equal(t, secret.Data[corev1.TLSCertKey], targetSecret.Data[corev1.TLSCertKey])
	assert.Equal(t, secret.Data[corev1.TLSPrivateKeyKey], targetSecret.Data[corev1.TLSPrivateKeyKey])
	assert.Equal(t, 4, len(targetSecret.Data))
	assert.Equal(t, "tls", targetSecret.Name)
	assert.Equal(t, "app-target-ns", targetSecret.Namespace)
}

func TestTLS_ExternalCA_Gen(t *testing.T) {
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
			AppSpec: v1.AppSpec{
				Secrets: map[string]v1.Secret{
					"tls-ca": {Type: "tls"},
					"tls": {
						Type: "tls",
						Params: map[string]interface{}{
							"caSecret": "tls-ca",
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
	assert.Equal(t, "tls-ca", secret.Labels[labels.AcornSecretName])
	assert.True(t, strings.HasPrefix(secret.Name, "tls-ca"))
	assert.True(t, len(secret.Data[corev1.TLSCertKey]) > 0)
	assert.True(t, len(secret.Data[corev1.TLSPrivateKeyKey]) > 0)
	assert.True(t, len(secret.Data["ca.crt"]) > 0)
	assert.True(t, len(secret.Data["ca.key"]) > 0)

	secret = resp.Client.Created[1].(*corev1.Secret)
	assert.Equal(t, "tls", secret.Labels[labels.AcornSecretName])
	assert.True(t, strings.HasPrefix(secret.Name, "tls-"))
	assert.True(t, len(secret.Data[corev1.TLSCertKey]) > 0)
	assert.True(t, len(secret.Data[corev1.TLSPrivateKeyKey]) > 0)
	assert.True(t, len(secret.Data["ca.crt"]) == 0)
	assert.True(t, len(secret.Data["ca.key"]) == 0)
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

func TestTemplateToken_Gen(t *testing.T) {
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
			AppSpec: v1.AppSpec{
				Secrets: map[string]v1.Secret{
					"pass": {Type: "token",
						Params: map[string]interface{}{
							"characters": "abc",
							"length":     5,
						},
					},
					"pass2": {Type: "token",
						Params: map[string]interface{}{
							"characters": "xyz",
							"length":     6,
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
