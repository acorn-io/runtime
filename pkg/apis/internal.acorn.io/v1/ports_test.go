package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseVolumesWithoutBinding(t *testing.T) {
	input := []string{
		"bar:bar",
		"foo:bar",
		"bar:bar,size=11G,class=aclass",
		"foo:bar,size=11G,class=aclass",
	}

	vs, err := ParseVolumes(input, false)
	assert.NoError(t, err)
	assert.Equal(t, vs[0], vs[2])
	assert.Equal(t, vs[1], vs[3])
}

func TestParseVolumesWithBinding(t *testing.T) {
	input := []string{
		"bar:bar",
		"foo:bar",
		"bar:bar,size=11G,class=aclass",
		"foo:bar,size=11G,class=aclass",
		"foo:bar,class=aclass",
		"foo:bar,size=11G",
		"foo,size=11G,class=aclass",
	}
	vs, err := ParseVolumes(input, true)
	assert.NoError(t, err)
	assert.NotEqual(t, vs[0], vs[2])
	assert.NotEqual(t, vs[1], vs[3])
	assert.Equal(t, VolumeBinding{
		Volume: "bar",
		Target: "bar",
	}, vs[0])
	assert.Equal(t, VolumeBinding{
		Volume: "foo",
		Target: "bar",
	}, vs[1])
	assert.Equal(t, VolumeBinding{
		Volume: "bar",
		Target: "bar",
		Size:   "11G",
		Class:  "aclass",
	}, vs[2])
	assert.Equal(t, VolumeBinding{
		Volume: "foo",
		Target: "bar",
		Size:   "11G",
		Class:  "aclass",
	}, vs[3])
	assert.Equal(t, VolumeBinding{
		Volume: "foo",
		Target: "bar",
		Class:  "aclass",
	}, vs[4])
	assert.Equal(t, VolumeBinding{
		Volume: "foo",
		Target: "bar",
		Size:   "11G",
	}, vs[5])
	assert.Equal(t, VolumeBinding{
		Target: "foo",
		Size:   "11G",
		Class:  "aclass",
	}, vs[6])
}
