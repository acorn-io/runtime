package dns

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestBasicInit tests the basic scenarios around acornDNS being enabled, auto, and disabled
func TestBasicInit(t *testing.T) {
	ch := &configHandler{
		dns: &mockClient{},
	}

	h := tester.Harness{
		Scheme: scheme.Scheme,
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "acorn-config",
			Namespace: "acorn-system",
		},
		Data: map[string]string{
			"config": `{"acornDNS": "enabled"}`,
		},
	}

	// Test when AcornDNS set to "enabled"
	resp, err := h.Invoke(t, cm, ch)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, resp.Client.Created, 1)
	assert.Len(t, resp.Client.Updated, 0)
	secret := resp.Client.Created[0].(*corev1.Secret)
	assert.Equal(t, "acorn-dns", secret.Name)
	assert.Equal(t, "acorn-system", secret.Namespace)
	assert.Equal(t, "enabled", secret.Annotations[labels.AcornDNSState])
	assert.Equal(t, []byte("test.on-acorn.io"), secret.Data["domain"])
	assert.Equal(t, []byte("token"), secret.Data["token"])

	// Test when AcornDNS set to "auto" and no cluster domains
	cm.Data["config"] = `{"acornDNS": "auto"}`
	resp, err = h.Invoke(t, cm, ch)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, resp.Client.Created, 1)
	assert.Len(t, resp.Client.Updated, 0)
	secret = resp.Client.Created[0].(*corev1.Secret)
	assert.Equal(t, "auto", secret.Annotations[labels.AcornDNSState])
	assert.Equal(t, []byte("test.on-acorn.io"), secret.Data["domain"])
	assert.Equal(t, []byte("token"), secret.Data["token"])

	// Test when AcornDNS set to "auto" and there is cluster domains
	cm.Data["config"] = `{"acornDNS": "auto", "clusterDomains": ["foo.com"]}`
	resp, err = h.Invoke(t, cm, ch)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, resp.Client.Created, 1)
	assert.Len(t, resp.Client.Updated, 0)
	secret = resp.Client.Created[0].(*corev1.Secret)
	assert.Equal(t, "auto", secret.Annotations[labels.AcornDNSState])
	assert.Equal(t, []byte("test.on-acorn.io"), secret.Data["domain"])
	assert.Equal(t, []byte("token"), secret.Data["token"])

	// Test when AcornDNS set to "disabled"
	cm.Data["config"] = `{"acornDNS": "disabled"}`
	resp, err = h.Invoke(t, cm, ch)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, resp.Client.Created, 1)
	assert.Len(t, resp.Client.Updated, 0)
	secret = resp.Client.Created[0].(*corev1.Secret)
	assert.Equal(t, "disabled", secret.Annotations[labels.AcornDNSState])
	assert.Equal(t, 0, len(secret.Data["domain"]))
	assert.Equal(t, 0, len(secret.Data["token"]))
}

// TestDisabling tests scenarios where acornDNS is going from enabled to disabled
func TestDisabling(t *testing.T) {
	ch := &configHandler{
		dns: &mockClient{},
	}

	h := tester.Harness{
		Scheme: scheme.Scheme,
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "acorn-config",
			Namespace: "acorn-system",
		},
		Data: map[string]string{
			"config": `{"acornDNS": "disabled"}`,
		},
	}

	h.Existing = append(h.Existing, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "acorn-dns",
			Namespace: "acorn-system",
			Annotations: map[string]string{
				labels.AcornDNSState: "enabled",
			},
		},
		Data: map[string][]byte{
			"domain": []byte("test.on-acorn.io"),
			"token":  []byte("token"),
		},
	})

	// Test when AcornDNS set to "enabled"
	resp, err := h.Invoke(t, cm, ch)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, resp.Client.Created, 0)
	assert.Len(t, resp.Client.Updated, 1)
	secret := resp.Client.Updated[0].(*corev1.Secret)
	assert.Equal(t, "acorn-dns", secret.Name)
	assert.Equal(t, "acorn-system", secret.Namespace)
	assert.Equal(t, "disabled", secret.Annotations[labels.AcornDNSState])
	assert.Equal(t, []byte("test.on-acorn.io"), secret.Data["domain"])
	assert.Equal(t, []byte("token"), secret.Data["token"])
}
