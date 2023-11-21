package v1

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseHostnameBinding(t *testing.T) {
	p, err := ParsePortBindings([]string{"example.com:service"})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "example.com", p[0].Hostname)
	assert.Equal(t, "service", p[0].TargetServiceName)
	assert.Equal(t, int32(0), p[0].TargetPort)
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

func TestUserContextUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected *UserContext
		wantErr  bool
	}{
		{
			name: "valid uid+gid",
			data: []byte(`"123:456"`),
			expected: &UserContext{
				UID: 123,
				GID: 456,
			},
			wantErr: false,
		},
		{
			name: "valid uid only int",
			data: []byte(`123`),
			expected: &UserContext{
				UID: 123,
				GID: 123,
			},
			wantErr: false,
		},
		{
			name:     "invalid uid+gid no quotes",
			data:     []byte(`123:456`),
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "invalid uid",
			data:     []byte(`"abc:456"`),
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "invalid gid",
			data:     []byte(`"123:def"`),
			expected: nil,
			wantErr:  true,
		},
		{
			name: "valid uid only string",
			data: []byte(`"123"`),
			expected: &UserContext{
				UID: 123,
				GID: 123,
			},
			wantErr: false,
		},
		{
			name:     "invalid extra field",
			data:     []byte(`"1000:1000:1000"`),
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "invalid missing uid",
			data:     []byte(`":1000"`),
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "invalid missing gid",
			data:     []byte(`"1000:"`),
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var uc UserContext
			err := uc.UnmarshalJSON(tt.data)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, tt.expected)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, &uc)
			}
		})
	}
}

func FuzzUserContextUnmarshalJSON(f *testing.F) {
	// Add seed corpus
	f.Add([]byte(`123`))                      // Simple integer
	f.Add([]byte(`"123:456"`))                // String with UID and GID
	f.Add([]byte(`":123"`))                   // String with UID missing
	f.Add([]byte(`"123:"`))                   // String with GID missing
	f.Add([]byte(`"not a number"`))           // Invalid string
	f.Add([]byte(`{"UID": 456, "GID": 789}`)) // JSON object
	f.Add([]byte(`{"invalid": "json"}`))      // Invalid JSON for UserContext
	f.Add([]byte(`["not", "object"]`))        // Invalid type

	f.Fuzz(func(t *testing.T, data []byte) {
		var userContext UserContext
		err := userContext.UnmarshalJSON(data)

		if isString(data) {
			// If the data is string, check for valid UID/GID parsing or error
			s, _ := parseString(data)
			parts := strings.Split(s, ":")
			if len(parts) == 1 || len(parts) == 2 {
				_, errUID := strconv.ParseInt(parts[0], 10, 64)
				if len(parts) == 2 {
					_, errGID := strconv.ParseInt(parts[1], 10, 64)
					if errUID != nil || errGID != nil {
						require.Error(t, err, fmt.Errorf("Expected error for invalid UID/GID in string but got nil"))
					}
				} else if errUID != nil {
					require.Error(t, err, fmt.Errorf("Expected error for invalid UID in string but got nil"))
				}
			}
		} else {
			// Check for valid int or JSON
			var tempInt int64
			if errUnmarshal := json.Unmarshal(data, &tempInt); errUnmarshal == nil {
				// If it's a valid integer, expect no error from UnmarshalJSON
				require.NoError(t, err, fmt.Errorf("Expected no error for valid integer input but got %w", err))
				if userContext.UID != tempInt || userContext.GID != tempInt {
					t.Errorf("Expected UID and GID to be %v, got UID: %v, GID: %v", tempInt, userContext.UID, userContext.GID)
				}
			} else {
				// Check if it's valid JSON that can be unmarshalled
				var temp interface{}
				if json.Unmarshal(data, &temp) == nil {
					_, ok := temp.(map[string]interface{})
					if !ok {
						require.Error(t, err, fmt.Errorf("Expected error for non-object JSON but got nil"))
					}
				} else {
					require.Error(t, err, fmt.Errorf("Expected error for invalid JSON but got nil"))
				}
			}
		}
	})
}

func TestParseVolumeReference(t *testing.T) {
	s, subPath, preload, err := parseVolumeReference("name")
	require.NoError(t, err)
	require.Equal(t, "name", s)
	require.Empty(t, subPath)
	require.False(t, preload)

	_, subPath, preload, err = parseVolumeReference("volume://foo?preload=true&subPath=bar")
	require.NoError(t, err)

	require.Equal(t, "bar", subPath)
	require.True(t, preload)

	_, subPath, preload, err = parseVolumeReference("volume://foo?preload=false")
	require.NoError(t, err)
	require.Empty(t, subPath)
	require.False(t, preload)

	_, _, _, err = parseVolumeReference("volume://foo?preload=foo")
	require.Error(t, err)
}
