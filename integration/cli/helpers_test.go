package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func findBin(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 5; i++ {
		test := filepath.Join(dir, "bin", "acorn")
		if _, err := os.Stat(test); err == nil {
			return test
		}
		dir = filepath.Dir(dir)
	}

	t.Fatal("failed to find acorn binary, expecting it in ./bin/acorn in the git project directory")
	return ""
}

func output(args []string, out, errOut string) string {
	return fmt.Sprintf("args: %v, stdout: [%s], stderr: [%s]", args, out, errOut)
}

func acorn(t *testing.T, args ...string) (string, string) {
	t.Helper()

	out, errOut, code := acornWithExitCode(t, args...)
	assert.Equal(t, 0, code, output(args, out, errOut))
	return out, errOut
}

func acornWithExitCode(t *testing.T, args ...string) (string, string, int) {
	t.Helper()

	bin := findBin(t)
	cmd := exec.Command(bin, args...)

	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	cmd.Stdout = out
	cmd.Stderr = errOut
	cmd.Stdin = &bytes.Buffer{}
	err := cmd.Run()
	if target := (*exec.ExitError)(nil); errors.As(err, &target) {
		return out.String(), errOut.String(), target.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return out.String(), errOut.String(), 0
}
