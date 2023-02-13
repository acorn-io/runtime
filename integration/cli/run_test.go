package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunHelloWorld(t *testing.T) {
	out, _ := acorn(t, "fail")
	assert.Contains(t, out, "bad")
}
