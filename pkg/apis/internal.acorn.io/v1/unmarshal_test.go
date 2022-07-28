package v1

import (
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
