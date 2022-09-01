package appdefinition

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
)

func TestVolumeController(t *testing.T) {
	dirs, err := os.ReadDir("testdata/volumes")
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range dirs {
		tester.DefaultTest(t, scheme.Scheme, filepath.Join("testdata/volumes", dir.Name()), DeploySpec)
	}
}
