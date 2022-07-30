package cli

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
