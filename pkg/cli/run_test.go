package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/stretchr/testify/assert"
)

func ArgsToApp(t *testing.T, args ...string) *apiv1.App {
	buf := &bytes.Buffer{}
	cmd := NewRun(buf)
	cmd.SetArgs(append([]string{"-o", "json"}, args...))
	err := cmd.Execute()
	if err != nil {
		t.Fatal(err)
	}

	result := &apiv1.App{}
	err = json.Unmarshal(buf.Bytes(), result)
	if err != nil {
		t.Fatal(err)
	}

	return result
}

func TestVolumeNoSplit(t *testing.T) {
	// skip because in CI Acorn isn't installed the client blows up
	t.Skip()

	app := ArgsToApp(t, "-v", "mysql-data-0,class=longhorn", "ghcr.io/acorn-io/library/mariadb:v10.6.8-focal-acorn.1")
	assert.Len(t, app.Spec.Volumes, 1)
	assert.Equal(t, "longhorn", app.Spec.Volumes[0].Class)
	assert.Equal(t, "mysql-data-0", app.Spec.Volumes[0].Target)
}

func TestRunArgs_Env(t *testing.T) {
	os.Setenv("x222", "y333")
	runArgs := RunArgs{
		Env: []string{"x222", "y=1"},
	}
	opts, err := runArgs.ToOpts()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "x222", opts.Env[0].Name)
	assert.Equal(t, "y333", opts.Env[0].Value)
	assert.Equal(t, "y", opts.Env[1].Name)
	assert.Equal(t, "1", opts.Env[1].Value)
}
