package credentials

import (
	"sort"
	"testing"

	"github.com/acorn-io/acorn/integration/helper"
	"github.com/acorn-io/acorn/pkg/client"
	kclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/stretchr/testify/assert"
)

func TestCredentialCreate(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	reg, close := helper.StartRegistry(t)
	reg1, close1 := helper.StartRegistry(t)
	defer close()
	defer close1()

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	c, err := client.New(restConfig, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	cred, err := c.CredentialCreate(ctx, reg, "user", "pass", false)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, reg, cred.Name)
	assert.Equal(t, reg, cred.ServerAddress)
	assert.Equal(t, "user", cred.Username)
	assert.Nil(t, cred.Password)

	cred1, err := c.CredentialCreate(ctx, reg1, "user2", "pass2", false)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, reg1, cred1.Name)
	assert.Equal(t, reg1, cred1.ServerAddress)
	assert.Equal(t, "user2", cred1.Username)
	assert.Nil(t, cred1.Password)

	cred1New, err := c.CredentialGet(ctx, reg1)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, cred1, cred1New)
}

func TestCredentialList(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	reg, close := helper.StartRegistry(t)
	reg1, close1 := helper.StartRegistry(t)
	defer close()
	defer close1()

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	c, err := client.New(restConfig, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	cred1, err := c.CredentialCreate(ctx, reg, "user", "pass", false)
	if err != nil {
		t.Fatal(err)
	}

	cred2, err := c.CredentialCreate(ctx, reg1, "user2", "pass2", false)
	if err != nil {
		t.Fatal(err)
	}

	creds, err := c.CredentialList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	sort.Slice(creds, func(i, j int) bool {
		return creds[i].Username < creds[j].Username
	})

	assert.Equal(t, cred1, &creds[0])
	assert.Equal(t, cred2, &creds[1])
}

func TestCredentialGet(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	reg, close := helper.StartRegistry(t)
	reg1, close1 := helper.StartRegistry(t)
	defer close()
	defer close1()

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	c, err := client.New(restConfig, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.CredentialCreate(ctx, reg, "user", "pass", false)
	if err != nil {
		t.Fatal(err)
	}

	cred1, err := c.CredentialCreate(ctx, reg1, "user2", "pass2", false)
	if err != nil {
		t.Fatal(err)
	}

	cred1New, err := c.CredentialGet(ctx, reg1)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, cred1, cred1New)
}

func TestCredentialUpdate(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	reg, close := helper.StartRegistry(t)
	reg1, close1 := helper.StartRegistry(t)
	defer close()
	defer close1()

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	c, err := client.New(restConfig, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.CredentialCreate(ctx, reg, "user", "pass", false)
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.CredentialCreate(ctx, reg1, "user2", "pass2", false)
	if err != nil {
		t.Fatal(err)
	}

	cred1New, err := c.CredentialUpdate(ctx, reg1, "user3", "pass3", false)
	if err != nil {
		t.Fatal(err)
	}

	cred1NewNew, err := c.CredentialGet(ctx, reg1)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "user3", cred1NewNew.Username)
	assert.Nil(t, cred1NewNew.Password)
	assert.Equal(t, cred1New, cred1NewNew)
}

func TestCredentialDelete(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	reg, close := helper.StartRegistry(t)
	reg1, close1 := helper.StartRegistry(t)
	defer close()
	defer close1()

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	c, err := client.New(restConfig, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.CredentialCreate(ctx, reg, "user", "pass", false)
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.CredentialCreate(ctx, reg1, "user2", "pass2", false)
	if err != nil {
		t.Fatal(err)
	}

	creds, err := c.CredentialList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, creds, 2)

	_, err = c.CredentialDelete(ctx, reg1)
	if err != nil {
		t.Fatal(err)
	}

	creds, err = c.CredentialList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, creds, 1)
}
