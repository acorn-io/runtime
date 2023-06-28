package ports

import (
	"testing"

	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/stretchr/testify/assert"
)

func TestCollectPorts(t *testing.T) {
	testCases := []struct {
		name     string
		ports    []v1.PortDef
		expected []v1.PortDef
	}{
		{
			name:     "empty",
			ports:    []v1.PortDef{},
			expected: nil,
		},
		{
			name:     "single",
			ports:    []v1.PortDef{{TargetPort: 80}},
			expected: []v1.PortDef{{TargetPort: 80, Port: 80}},
		},
		{
			name: "duplicate public port",
			ports: []v1.PortDef{
				{TargetPort: 8080, Port: 8000},
				{TargetPort: 9090, Port: 8000},
			},
			expected: []v1.PortDef{{TargetPort: 8080, Port: 8000}},
		},
		{
			name: "duplicate target port",
			ports: []v1.PortDef{
				{TargetPort: 8080, Port: 8000},
				{TargetPort: 8080, Port: 9000},
			},
			expected: []v1.PortDef{
				{TargetPort: 8080, Port: 8000},
				{TargetPort: 8080, Port: 9000},
			},
		},
		{
			name: "one undefined hostname",
			ports: []v1.PortDef{
				{TargetPort: 8080, Port: 8000},
				{TargetPort: 8080, Port: 8000, Hostname: "myapp.local"},
			},
			expected: []v1.PortDef{
				{TargetPort: 8080, Port: 8000},
				{TargetPort: 8080, Port: 8000, Hostname: "myapp.local"},
			},
		},
		{
			name: "duplicate everything",
			ports: []v1.PortDef{
				{TargetPort: 8080, Port: 8000, Hostname: "myapp.local"},
				{TargetPort: 8080, Port: 8000, Hostname: "myapp.local"},
			},
			expected: []v1.PortDef{{TargetPort: 8080, Port: 8000, Hostname: "myapp.local"}},
		},
		{
			name: "duplicate port and target port with different hostnames",
			ports: []v1.PortDef{
				{TargetPort: 8080, Port: 8000, Hostname: "myapp.local"},
				{TargetPort: 8080, Port: 8000, Hostname: "myapp2.local"},
			},
			expected: []v1.PortDef{
				{TargetPort: 8080, Port: 8000, Hostname: "myapp.local"},
				{TargetPort: 8080, Port: 8000, Hostname: "myapp2.local"},
			},
		},
		{
			name: "duplicate port and hostname with different target ports",
			ports: []v1.PortDef{
				{TargetPort: 8080, Port: 8000, Hostname: "myapp.local"},
				{TargetPort: 9090, Port: 8000, Hostname: "myapp.local"},
			},
			expected: []v1.PortDef{{TargetPort: 8080, Port: 8000, Hostname: "myapp.local"}},
		},
		{
			name: "duplicate target port and hostname with different public ports",
			ports: []v1.PortDef{
				{TargetPort: 8080, Port: 8000, Hostname: "myapp.local"},
				{TargetPort: 8080, Port: 9000, Hostname: "myapp.local"},
			},
			expected: []v1.PortDef{
				{TargetPort: 8080, Port: 8000, Hostname: "myapp.local"},
			},
		},
		{
			name: "duplicate port, different target ports and hostnames",
			ports: []v1.PortDef{
				{TargetPort: 8080, Port: 8000, Hostname: "myapp.local"},
				{TargetPort: 9090, Port: 8000, Hostname: "myapp2.local"},
			},
			expected: []v1.PortDef{{TargetPort: 8080, Port: 8000, Hostname: "myapp.local"}},
		},
		{
			name: "duplicate target port, different ports and hostnames",
			ports: []v1.PortDef{
				{TargetPort: 8080, Port: 8000, Hostname: "myapp.local"},
				{TargetPort: 8080, Port: 9000, Hostname: "myapp2.local"},
			},
			expected: []v1.PortDef{
				{TargetPort: 8080, Port: 8000, Hostname: "myapp.local"},
				{TargetPort: 8080, Port: 9000, Hostname: "myapp2.local"},
			},
		},
		{
			name: "duplicate hostnames, different ports and target ports",
			ports: []v1.PortDef{
				{TargetPort: 8080, Port: 8000, Hostname: "myapp.local"},
				{TargetPort: 9090, Port: 9000, Hostname: "myapp.local"},
			},
			expected: []v1.PortDef{
				{TargetPort: 8080, Port: 8000, Hostname: "myapp.local"},
			},
		},
		{
			name: "three completely different PortDefs",
			ports: []v1.PortDef{
				{TargetPort: 8080, Port: 8000, Hostname: "myapp.local"},
				{TargetPort: 9090, Port: 9000, Hostname: "myapp2.local"},
				{TargetPort: 7070, Port: 7000, Hostname: "myapp3.local"},
			},
			expected: []v1.PortDef{
				{TargetPort: 8080, Port: 8000, Hostname: "myapp.local"},
				{TargetPort: 9090, Port: 9000, Hostname: "myapp2.local"},
				{TargetPort: 7070, Port: 7000, Hostname: "myapp3.local"},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			seen := map[int32][]int32{}
			seenHostname := map[string]struct{}{}
			assert.Equal(t, tt.expected, collectPorts(seen, seenHostname, tt.ports, false))
		})
	}
}
