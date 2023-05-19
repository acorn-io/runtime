package event

import (
	"testing"
	"time"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestContentID(t *testing.T) {
	for _, tt := range []struct {
		name  string
		equal bool
		a, b  apiv1.Event
	}{
		{
			name:  "Equal/Duplicate",
			equal: true,
			a:     apiv1.Event{},
			b:     apiv1.Event{},
		},
		{
			name:  "Equal/Diff/Context",
			equal: true,
			a: apiv1.Event{
				Details: v1.GenericMap{
					"info": "1",
				},
			},
			b: apiv1.Event{
				Details: v1.GenericMap{
					"info": 1,
				},
			},
		},
		{
			name:  "NotEqual/Diff/Observed",
			equal: false,
			a: apiv1.Event{
				Observed: metav1.Now(),
			},
			b: apiv1.Event{
				Observed: metav1.NewTime(metav1.Now().Add(time.Hour)),
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			idA, err := ContentID(&tt.a)
			require.NoError(t, err)

			idB, err := ContentID(&tt.b)
			require.NoError(t, err)

			if tt.equal {
				// IDs should match
				require.Equal(t, idA, idB)
				return
			}
			// IDs shouldn't match
			require.NotEqual(t, idA, idB)
		})
	}
}
