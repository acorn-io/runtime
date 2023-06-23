package secrets

import (
	"testing"

	"github.com/acorn-io/runtime/integration/helper"
	"github.com/acorn-io/runtime/pkg/client"
	kclient "github.com/acorn-io/runtime/pkg/k8sclient"
	"github.com/stretchr/testify/assert"
)

func TestSecretCreate(t *testing.T) {
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	c, err := client.New(restConfig, "", ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	secret, err := c.SecretCreate(ctx, "foo", "opaque", map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
	})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "foo", secret.Name)
	assert.Equal(t, "opaque", secret.Type)
	assert.Len(t, secret.Data, 0)
	assert.Len(t, secret.Keys, 2)
	assert.Equal(t, "key1", secret.Keys[0])
	assert.Equal(t, "key2", secret.Keys[1])
}

func TestSecretList(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	c, err := client.New(restConfig, "", ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	secret1, err := c.SecretCreate(ctx, "secret2", "opaque", map[string][]byte{"key": []byte("value")})
	if err != nil {
		t.Fatal(err)
	}

	secret2, err := c.SecretCreate(ctx, "secret1", "opaque", map[string][]byte{"key2": []byte("value2")})
	if err != nil {
		t.Fatal(err)
	}

	secrets, err := c.SecretList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, secret1.ObjectMeta, secrets[1].ObjectMeta)
	assert.Equal(t, secret1.Type, secrets[1].Type)
	assert.Equal(t, secret1.Keys, secrets[1].Keys)
	assert.Equal(t, secret1.Data, secrets[1].Data)

	assert.Equal(t, secret2.ObjectMeta, secrets[0].ObjectMeta)
	assert.Equal(t, secret2.Type, secrets[0].Type)
	assert.Equal(t, secret2.Keys, secrets[0].Keys)
	assert.Equal(t, secret2.Data, secrets[0].Data)
}

func TestSecretGet(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	c, err := client.New(restConfig, "", ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.SecretCreate(ctx, "secret1", "opaque", map[string][]byte{"key": []byte("value")})
	if err != nil {
		t.Fatal(err)
	}

	secret, err := c.SecretCreate(ctx, "secret2", "opaque", map[string][]byte{"key2": []byte("value2")})
	if err != nil {
		t.Fatal(err)
	}

	newSecret, err := c.SecretGet(ctx, "secret2")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, secret, newSecret)
}

func TestSecretReveal(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	c, err := client.New(restConfig, "", ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.SecretCreate(ctx, "secret1", "opaque", map[string][]byte{"key": []byte("value")})
	if err != nil {
		t.Fatal(err)
	}

	secret, err := c.SecretCreate(ctx, "secret2", "opaque", map[string][]byte{"key2": []byte("value2")})
	if err != nil {
		t.Fatal(err)
	}

	newSecret, err := c.SecretReveal(ctx, "secret2")
	if err != nil {
		t.Fatal(err)
	}

	assert.NotEqual(t, secret, newSecret)
	assert.Equal(t, map[string][]byte{"key2": []byte("value2")}, newSecret.Data)
}

func TestSecretUpdate(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	c, err := client.New(restConfig, "", ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.SecretCreate(ctx, "secret1", "opaque", map[string][]byte{"key": []byte("value")})
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.SecretCreate(ctx, "secret2", "opaque", map[string][]byte{"key2": []byte("value2")})
	if err != nil {
		t.Fatal(err)
	}

	secretNew, err := c.SecretUpdate(ctx, "secret2", map[string][]byte{"key3": []byte("value3")})
	if err != nil {
		t.Fatal(err)
	}

	secretNewNew, err := c.SecretGet(ctx, "secret2")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "secret2", secretNewNew.Name)
	assert.Equal(t, "opaque", secretNewNew.Type)
	assert.Len(t, secretNewNew.Data, 0)
	assert.Equal(t, []string{"key3"}, secretNewNew.Keys)
	assert.Equal(t, secretNew, secretNewNew)
}

func TestSecretDelete(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	c, err := client.New(restConfig, "", ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.SecretCreate(ctx, "secret1", "opaque", map[string][]byte{"key": []byte("value")})
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.SecretCreate(ctx, "secret2", "opaque", map[string][]byte{"key2": []byte("value2")})
	if err != nil {
		t.Fatal(err)
	}

	secrets, err := c.SecretList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, secrets, 2)

	secret, err := c.SecretDelete(ctx, "secret1")
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, secret)

	secrets, err = c.SecretList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, secrets, 1)
}
