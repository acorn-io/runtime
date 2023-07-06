package credentials

import (
	"sort"
	"testing"

	"github.com/acorn-io/runtime/integration/helper"
	"github.com/acorn-io/runtime/pkg/client"
	kclient "github.com/acorn-io/runtime/pkg/k8sclient"
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
	project := helper.TempProject(t, kclient)

	c, err := client.New(restConfig, "", project.Name)
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
	project := helper.TempProject(t, kclient)

	c, err := client.New(restConfig, "", project.Name)
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

	assert.Equal(t, cred1.ObjectMeta, creds[0].ObjectMeta)
	assert.Equal(t, cred1.ServerAddress, creds[0].ServerAddress)
	assert.Equal(t, cred1.Username, creds[0].Username)
	assert.Equal(t, cred1.Password, creds[0].Password)
	assert.Equal(t, cred1.SkipChecks, creds[0].SkipChecks)

	assert.Equal(t, cred2.ObjectMeta, creds[1].ObjectMeta)
	assert.Equal(t, cred2.ServerAddress, creds[1].ServerAddress)
	assert.Equal(t, cred2.Username, creds[1].Username)
	assert.Equal(t, cred2.Password, creds[1].Password)
	assert.Equal(t, cred2.SkipChecks, creds[1].SkipChecks)
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
	project := helper.TempProject(t, kclient)

	c, err := client.New(restConfig, "", project.Name)
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
	project := helper.TempProject(t, kclient)

	c, err := client.New(restConfig, "", project.Name)
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
	project := helper.TempProject(t, kclient)

	c, err := client.New(restConfig, "", project.Name)
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
