package v1

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseVolumesWithoutBinding(t *testing.T) {
	input := []string{
		"bar",
	}

	vs, err := ParseVolumes(input, false)
	assert.NoError(t, err)
	assert.Equal(t, "", vs[0].Volume)
	assert.Equal(t, "bar", vs[0].Target)

	input = []string{
		"bar:bar",
	}

	_, err = ParseVolumes(input, false)
	assert.Error(t, err)

	input = []string{
		"bar:bar",
		"bar,size=11,class=aclass",
	}

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
		"bar:bar,size=11G,class=aclass",
	}
	_, err := ParseVolumes(input, false)
	assert.Error(t, err)
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

func TestParsePorts(t *testing.T) {
	tests := []struct {
		name       string
		port       string
		wantResult PortDef
		wantErr    assert.ErrorAssertionFunc
	}{
		{
			port: "80",
			wantResult: PortDef{
				TargetPort: 80,
			},
			wantErr: assert.NoError,
		},
		{
			port: "80/http",
			wantResult: PortDef{
				Protocol:   ProtocolHTTP,
				TargetPort: 80,
			},
			wantErr: assert.NoError,
		},
		{
			port: "81:80",
			wantResult: PortDef{
				Port:       81,
				TargetPort: 80,
			},
			wantErr: assert.NoError,
		},
		{
			port: "81:80/tcp",
			wantResult: PortDef{
				Port:       81,
				TargetPort: 80,
				Protocol:   ProtocolTCP,
			},
			wantErr: assert.NoError,
		},
		{
			port: "81:80/http",
			wantResult: PortDef{
				Port:       81,
				TargetPort: 80,
				Protocol:   ProtocolHTTP,
			},
			wantErr: assert.NoError,
		},
		{
			port:    "app:80",
			wantErr: assert.Error,
		},
		{
			port: "example.com:80",
			wantResult: PortDef{
				Hostname:   "example.com",
				Protocol:   ProtocolHTTP,
				TargetPort: 80,
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		if tt.name == "" {
			tt.name = tt.port
		}
		t.Run(tt.name, func(t *testing.T) {
			gotResult, err := ParsePorts([]string{tt.port})
			if !tt.wantErr(t, err, fmt.Sprintf("ParsePorts(%v)", tt.port)) {
				return
			}
			if err == nil {
				assert.Equalf(t, tt.wantResult, gotResult[0], "ParsePorts(%v)", tt.port)
			}
		})
	}
}

func TestParsePortBindings(t *testing.T) {
	tests := []struct {
		name       string
		port       string
		wantResult PortBinding
		wantErr    assert.ErrorAssertionFunc
	}{
		{
			port: "80",
			wantResult: PortBinding{
				TargetPort: 80,
			},
			wantErr: assert.NoError,
		},
		{
			port: "80/http",
			wantResult: PortBinding{
				Protocol:   ProtocolHTTP,
				TargetPort: 80,
			},
			wantErr: assert.NoError,
		},
		{
			port: "81:80",
			wantResult: PortBinding{
				Port:       81,
				TargetPort: 80,
			},
			wantErr: assert.NoError,
		},
		{
			port: "81:80/tcp",
			wantResult: PortBinding{
				Port:       81,
				TargetPort: 80,
				Protocol:   ProtocolTCP,
			},
			wantErr: assert.NoError,
		},
		{
			port: "81:80/http",
			wantResult: PortBinding{
				Port:       81,
				TargetPort: 80,
				Protocol:   ProtocolHTTP,
			},
			wantErr: assert.Error,
		},
		{
			port: "app:80",
			wantResult: PortBinding{
				TargetPort:        80,
				TargetServiceName: "app",
			},
			wantErr: assert.NoError,
		},
		{
			port: "example.com:80",
			wantResult: PortBinding{
				Hostname:   "example.com",
				Protocol:   ProtocolHTTP,
				TargetPort: 80,
			},
			wantErr: assert.NoError,
		},
		{
			port: "app:80/tcp",
			wantResult: PortBinding{
				TargetPort:        80,
				TargetServiceName: "app",
				Protocol:          ProtocolTCP,
			},
			wantErr: assert.NoError,
		},
		{
			port:    "example.com:80/tcp",
			wantErr: assert.Error,
		},
		{
			port: "example.com:80/http",
			wantResult: PortBinding{
				Hostname:   "example.com",
				Protocol:   ProtocolHTTP,
				TargetPort: 80,
			},
			wantErr: assert.NoError,
		},
		{
			port:    "foo:bar",
			wantErr: assert.Error,
		},
		{
			port: "example.com:bar",
			wantResult: PortBinding{
				Protocol:          ProtocolHTTP,
				Hostname:          "example.com",
				TargetServiceName: "bar",
			},
			wantErr: assert.NoError,
		},
		{
			port: "example.com:bar:82",
			wantResult: PortBinding{
				Protocol:          ProtocolHTTP,
				TargetPort:        82,
				Hostname:          "example.com",
				TargetServiceName: "bar",
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.port, func(t *testing.T) {
			gotResult, err := ParsePortBindings([]string{tt.port})
			if !tt.wantErr(t, err, fmt.Sprintf("ParsePorts(%v)", tt.port)) {
				return
			}
			if err == nil {
				assert.Equalf(t, tt.wantResult, gotResult[0], "ParsePorts(%v)", tt.port)
			}
		})
	}
}
