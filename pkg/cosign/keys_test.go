package cosign

import (
	"testing"
)

func TestKeyImport(t *testing.T) {
	testcases := []struct {
		name     string
		key      string
		password []byte
	}{
		{
			name:     "OpenSSH RSA without password",
			key:      "testdata/keys/openssh-rsa-nopw.key",
			password: nil,
		},
		{
			name:     "OpenSSH ED25519 with password",
			key:      "testdata/keys/openssh-ed25519-pw_foobar.key",
			password: []byte("foobar"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			// Test with correct password - should never fail
			_, err := ImportKeyPair(tc.key, tc.password)
			if err != nil {
				t.Errorf("ImportKeyPair(%s, %v) errored where it should not: %v", tc.key, string(tc.password), err)
			}

			// Test with incorrect password - should always fail
			_, err = ImportKeyPair(tc.key, []byte("wrong"))
			if err == nil {
				t.Errorf("ImportKeyPair(%s, wrong) succeeded where it should error", tc.key)
			}

			// Test with no password - should fail if password is required
			_, err = ImportKeyPair(tc.key, nil)
			if err == nil && tc.password != nil || err != nil && tc.password == nil {
				t.Errorf("ImportKeyPair(%s, nil) errored where it should not: %v", tc.key, err)
			}
		})
	}
}
