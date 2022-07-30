package v1

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseHostnameBinding(t *testing.T) {
	f, err := ParsePortBindings(true, []string{"example.com:service"})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, PortBinding{
		Protocol:          ProtocolHTTP,
		Publish:           true,
		ServiceName:       "example.com",
		TargetServiceName: "service",
	}, f[0])
}

func TestParseEnv(t *testing.T) {
	assert.Nil(t, os.Setenv("x111", "y111"))
	input := []string{
		"k=v",
		"x111",
	}
	f := ParseNameValues(false, input...)
	assert.Equal(t, NameValue{
		Name:  "k",
		Value: "v",
	}, f[0])
	assert.Equal(t, NameValue{
		Name:  "x111",
		Value: "",
	}, f[1])

	f = ParseNameValues(true, input...)
	assert.Equal(t, NameValue{
		Name:  "k",
		Value: "v",
	}, f[0])
	assert.Equal(t, NameValue{
		Name:  "x111",
		Value: "y111",
	}, f[1])
}
