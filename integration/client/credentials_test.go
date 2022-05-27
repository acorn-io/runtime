package client

import (
	"sort"
	"testing"

	"github.com/acorn-io/acorn/integration/helper"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	kclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/stretchr/testify/assert"
)

func TestCredentialCreate(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	c, err := client.New(restConfig, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	cred, err := c.CredentialCreate(ctx, "example.com:443", "user", "pass")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "example.com:443", cred.Name)
	assert.Equal(t, apiv1.CredentialStorageTypeCluster, cred.Storage)
	assert.Equal(t, "example.com:443", cred.ServerAddress)
	assert.Equal(t, "user", cred.Username)
	assert.Equal(t, "", cred.Password)

	cred1, err := c.CredentialCreate(ctx, "two.example.com:443", "user2", "pass2")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "two.example.com:443", cred1.Name)
	assert.Equal(t, apiv1.CredentialStorageTypeCluster, cred1.Storage)
	assert.Equal(t, "two.example.com:443", cred1.ServerAddress)
	assert.Equal(t, "user2", cred1.Username)
	assert.Equal(t, "", cred1.Password)

	cred1New, err := c.CredentialGet(ctx, "two.example.com:443")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, cred1, cred1New)
}

func TestCredentialList(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	c, err := client.New(restConfig, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	cred1, err := c.CredentialCreate(ctx, "example.com:443", "user", "pass")
	if err != nil {
		t.Fatal(err)
	}

	cred2, err := c.CredentialCreate(ctx, "two.example.com:443", "user2", "pass2")
	if err != nil {
		t.Fatal(err)
	}

	creds, err := c.CredentialList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	sort.Slice(creds, func(i, j int) bool {
		return creds[i].Name < creds[j].Name
	})

	assert.Equal(t, cred1, &creds[0])
	assert.Equal(t, cred2, &creds[1])
}

func TestCredentialGet(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	c, err := client.New(restConfig, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.CredentialCreate(ctx, "example.com:443", "user", "pass")
	if err != nil {
		t.Fatal(err)
	}

	cred1, err := c.CredentialCreate(ctx, "two.example.com:443", "user2", "pass2")
	if err != nil {
		t.Fatal(err)
	}

	cred1New, err := c.CredentialGet(ctx, "two.example.com:443")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, cred1, cred1New)
}

func TestCredentialUpdate(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	c, err := client.New(restConfig, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.CredentialCreate(ctx, "example.com:443", "user", "pass")
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.CredentialCreate(ctx, "two.example.com:443", "user2", "pass2")
	if err != nil {
		t.Fatal(err)
	}

	cred1New, err := c.CredentialUpdate(ctx, "two.example.com:443", "user3", "pass3")
	if err != nil {
		t.Fatal(err)
	}

	cred1NewNew, err := c.CredentialGet(ctx, "two.example.com:443")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "user3", cred1NewNew.Username)
	assert.Equal(t, "", cred1NewNew.Password)
	assert.Equal(t, cred1New, cred1NewNew)
}

func TestCredentialDelete(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	c, err := client.New(restConfig, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.CredentialCreate(ctx, "example.com:443", "user", "pass")
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.CredentialCreate(ctx, "two.example.com:443", "user2", "pass2")
	if err != nil {
		t.Fatal(err)
	}

	creds, err := c.CredentialList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, creds, 2)

	_, err = c.CredentialDelete(ctx, "two.example.com:443")
	if err != nil {
		t.Fatal(err)
	}

	creds, err = c.CredentialList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, creds, 1)
}
