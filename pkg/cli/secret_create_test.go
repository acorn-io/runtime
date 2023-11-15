package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildSecret(t *testing.T) {
	c := SecretCreate{
		SecretFactory: SecretFactory{
			Data: []string{"key1=value1", "@key2=testdata/secret/value2.txt"},
			File: "testdata/secret/secret.yaml",
			Type: "fancy",
		},
	}

	secret, err := c.buildSecret()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "fancy", secret.Type)
	assert.Equal(t, map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
		"key4": []byte("value4"),
	}, secret.Data)
}
