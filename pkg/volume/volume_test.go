package volume

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
)

func TestAcornCreatedVolumeClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/default-storage-class", SyncVolumeClasses)
}

func TestUserChangedField(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/user-changed-volume-class", SyncVolumeClasses)
}

func TestManuallyManagedVolumeClasses(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/manually-managed", SyncVolumeClasses)
}

func TestEphemeralCreatedVolumeClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/ephemeral", CreateEphemeralVolumeClass)
}

func TestUserChangedEphemeral(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/ephemeral-user-changed", CreateEphemeralVolumeClass)
}

func TestManuallyManagedEphemeral(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/ephemeral-manually-managed", CreateEphemeralVolumeClass)
}
