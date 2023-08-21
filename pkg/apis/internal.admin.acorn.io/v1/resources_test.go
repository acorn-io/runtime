package v1

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestAdd(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name     string
		current  Resources
		incoming Resources
		expected Resources
	}{
		{
			name:    "add to empty resources",
			current: Resources{},
			incoming: Resources{
				Apps:          1,
				VolumeStorage: resource.MustParse("1Mi"),
			},
			expected: Resources{
				Apps:          1,
				VolumeStorage: resource.MustParse("1Mi"),
			},
		},
		{
			name: "add to existing resources",
			current: Resources{
				Apps:          1,
				VolumeStorage: resource.MustParse("1Mi"),
			},
			incoming: Resources{
				Apps:          1,
				Images:        1,
				VolumeStorage: resource.MustParse("1Mi"),
				CPU:           resource.MustParse("20m"),
			},
			expected: Resources{
				Apps:          2,
				Images:        1,
				VolumeStorage: resource.MustParse("2Mi"),
				CPU:           resource.MustParse("20m"),
			},
		},
		{
			name:    "does not change flags",
			current: Resources{},
			incoming: Resources{
				Unlimited:     true,
				Apps:          1,
				VolumeStorage: resource.MustParse("1Mi"),
			},
			expected: Resources{
				Apps:          1,
				VolumeStorage: resource.MustParse("1Mi"),
			},
		},
		{
			name: "add where current has a resource specified with unlimited",
			current: Resources{
				Apps:   Unlimited,
				Memory: UnlimitedQuantity(),
			},
			incoming: Resources{
				Apps:   1,
				Memory: resource.MustParse("1Mi"),
			},
			expected: Resources{
				Apps:   Unlimited,
				Memory: UnlimitedQuantity(),
			},
		},
		{
			name: "add where incoming has a resource specified with unlimited",
			current: Resources{
				Apps:   1,
				Memory: resource.MustParse("1Mi"),
			},
			incoming: Resources{
				Apps:   Unlimited,
				Memory: UnlimitedQuantity(),
			},
			expected: Resources{
				Apps:   Unlimited,
				Memory: UnlimitedQuantity(),
			},
		},
		{
			name: "add where current and incoming have a resource specified with unlimited",
			current: Resources{
				Apps:   Unlimited,
				Memory: UnlimitedQuantity(),
			},
			incoming: Resources{
				Apps:   Unlimited,
				Memory: UnlimitedQuantity(),
			},
			expected: Resources{
				Apps:   Unlimited,
				Memory: UnlimitedQuantity(),
			},
		},
	}

	// Run the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.current.Add(tc.incoming)
			assert.True(t, tc.current.Equals(tc.expected))
		})
	}
}

func TestRemove(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name     string
		current  Resources
		incoming Resources
		all      bool
		expected Resources
	}{
		{
			name:    "remove from empty resources",
			current: Resources{},
			incoming: Resources{
				Apps:   1,
				Memory: resource.MustParse("1Mi"),
			},
			expected: Resources{},
		},
		{
			name: "remove from existing resources",
			current: Resources{
				Apps:   1,
				Memory: resource.MustParse("1Mi"),
			},
			incoming: Resources{
				Apps:   1,
				Memory: resource.MustParse("1Mi"),
			},
			expected: Resources{},
		},
		{
			name: "should never get negative values",
			all:  true,
			current: Resources{
				Apps:          1,
				Memory:        resource.MustParse("1Mi"),
				Secrets:       1,
				VolumeStorage: resource.MustParse("1Mi"),
			},
			incoming: Resources{
				Apps:          2,
				Memory:        resource.MustParse("2Mi"),
				Secrets:       2,
				VolumeStorage: resource.MustParse("2Mi"),
			},
			expected: Resources{},
		},
		{
			name: "remove persistent counts with all",
			current: Resources{
				Secrets:       1,
				VolumeStorage: resource.MustParse("1Mi"),
			},
			incoming: Resources{
				Secrets:       1,
				VolumeStorage: resource.MustParse("1Mi"),
			},
			all:      true,
			expected: Resources{},
		},
		{
			name: "does not remove persistent counts without all",
			current: Resources{
				Secrets:       1,
				VolumeStorage: resource.MustParse("1Mi"),
			},
			incoming: Resources{
				Secrets:       1,
				VolumeStorage: resource.MustParse("1Mi"),
			},
			expected: Resources{
				Secrets:       1,
				VolumeStorage: resource.MustParse("1Mi"),
			},
		},
		{
			name: "remove where current has a resource specified with unlimited",
			current: Resources{
				Apps:   Unlimited,
				Memory: UnlimitedQuantity(),
			},
			incoming: Resources{
				Apps:   1,
				Memory: resource.MustParse("1Mi"),
			},
			expected: Resources{
				Apps:   Unlimited,
				Memory: UnlimitedQuantity(),
			},
		},
		{
			name: "remove where incoming has a resource specified with unlimited",
			current: Resources{
				Apps:   1,
				Memory: resource.MustParse("1Mi"),
			},
			incoming: Resources{
				Apps:   Unlimited,
				Memory: UnlimitedQuantity(),
			},
			expected: Resources{
				Apps:   1,
				Memory: resource.MustParse("1Mi"),
			},
		},
		{
			name: "remove where current and incoming have a resource specified with unlimited",
			current: Resources{
				Apps:   Unlimited,
				Memory: UnlimitedQuantity(),
			},
			incoming: Resources{
				Apps:   Unlimited,
				Memory: UnlimitedQuantity(),
			},
			expected: Resources{
				Apps:   Unlimited,
				Memory: UnlimitedQuantity(),
			},
		},
	}

	// Run the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.current.Remove(tc.incoming, tc.all)
			assert.True(t, tc.current.Equals(tc.expected))
		})
	}
}

func TestEquals(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name     string
		current  Resources
		incoming Resources
		expected bool
	}{
		{
			name:     "empty resources",
			current:  Resources{},
			incoming: Resources{},
			expected: true,
		},
		{
			name: "equal resources",
			current: Resources{
				Apps:          1,
				Secrets:       1,
				VolumeStorage: resource.MustParse("1Mi"),
			},
			incoming: Resources{
				Apps:          1,
				Secrets:       1,
				VolumeStorage: resource.MustParse("1Mi"),
			},
			expected: true,
		},
		{
			name: "unequal resources",
			current: Resources{
				Apps:          1,
				Secrets:       1,
				VolumeStorage: resource.MustParse("1Mi"),
			},
			incoming: Resources{
				Apps:          2,
				Secrets:       2,
				VolumeStorage: resource.MustParse("1Mi"),
			},
			expected: false,
		},
		{
			name: "equal resources with unlimited values",
			current: Resources{
				Apps:          Unlimited,
				Secrets:       Unlimited,
				VolumeStorage: UnlimitedQuantity(),
			},
			incoming: Resources{
				Apps:          Unlimited,
				Secrets:       Unlimited,
				VolumeStorage: UnlimitedQuantity(),
			},
			expected: true,
		},
	}

	// Run the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.current.Equals(tc.incoming))
		})
	}
}

func TestFits(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name        string
		current     Resources
		incoming    Resources
		expectedErr error
	}{
		{
			name:     "empty resources",
			current:  Resources{},
			incoming: Resources{},
		},
		{
			name: "fits resources",
			current: Resources{
				Apps:          1,
				Secrets:       1,
				VolumeStorage: resource.MustParse("1Mi"),
			},
			incoming: Resources{
				Apps:          1,
				Secrets:       1,
				VolumeStorage: resource.MustParse("1Mi"),
			},
		},

		{
			name: "does not fit resources",
			current: Resources{
				Apps:          1,
				Secrets:       1,
				VolumeStorage: resource.MustParse("1Mi"),
			},
			incoming: Resources{
				Apps:          2,
				Secrets:       2,
				VolumeStorage: resource.MustParse("1Mi"),
			},
			expectedErr: ErrExceededResources,
		},
		{
			name: "fits resources with unlimited flag set",
			current: Resources{
				Unlimited:     true,
				Apps:          1,
				Secrets:       1,
				VolumeStorage: resource.MustParse("1Mi"),
			},
			incoming: Resources{
				Apps:          2,
				Secrets:       2,
				VolumeStorage: resource.MustParse("2Mi"),
			},
		},
		{
			name: "fits resources with specified unlimited values",
			current: Resources{
				Apps:          Unlimited,
				Secrets:       Unlimited,
				VolumeStorage: UnlimitedQuantity(),
			},
			incoming: Resources{
				Apps:          2,
				Secrets:       2,
				VolumeStorage: resource.MustParse("2Mi"),
			},
		},
		{
			name: "fits count resources with specified unlimited values but not others",
			current: Resources{
				Jobs:    0,
				Apps:    Unlimited,
				Secrets: Unlimited,
			},
			incoming: Resources{
				Jobs:    2,
				Apps:    2,
				Secrets: 2,
			},
			expectedErr: ErrExceededResources,
		},

		{
			name: "fits quantity resources with specified unlimited values but not others",
			current: Resources{
				VolumeStorage: UnlimitedQuantity(),
			},
			incoming: Resources{
				CPU:           resource.MustParse("100m"),
				VolumeStorage: resource.MustParse("2Mi"),
			},
			expectedErr: ErrExceededResources,
		},
	}

	// Run the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.current.Fits(tc.incoming)
			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("expected %v, got %v", tc.expectedErr, err)
			}
		})
	}
}
