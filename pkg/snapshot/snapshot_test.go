package snapshot

import (
	"testing"

	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/acorn-io/runtime/pkg/scheme"
)

func TestAcornCreatedSnapshotClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/default-storage-class", SyncSnapshotClasses)
}
