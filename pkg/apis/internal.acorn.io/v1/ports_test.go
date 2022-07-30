package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseVolumeConfigure(t *testing.T) {
	input := []string{
		"bar:bar",
		"bar,size=11,class=aclass",
	}

	vs, err := ParseVolumes(input, false)
	assert.NoError(t, err)
	assert.Equal(t, vs[0], vs[1])

	vs, err = ParseVolumes(input, true)
	assert.NoError(t, err)
	assert.Equal(t, VolumeBinding{
		Volume: "",
		Target: "bar",
		Size:   "11G",
		Class:  "aclass",
	}, vs[1])

}

func TestParseVolumes(t *testing.T) {
	input := []string{
		"foo:bar",
		"foo:bar,size=11G,class=aclass",
	}

	vs, err := ParseVolumes(input, false)
	assert.NoError(t, err)
	assert.Equal(t, vs[0], vs[1])

	vs, err = ParseVolumes(input, true)
	assert.NoError(t, err)
	assert.NotEqual(t, vs[0], vs[1])
	assert.Equal(t, VolumeBinding{
		Volume: "foo",
		Target: "bar",
	}, vs[0])
	assert.Equal(t, VolumeBinding{
		Volume: "foo",
		Target: "bar",
		Size:   "11G",
		Class:  "aclass",
	}, vs[1])
}
