package v1

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestBaseResourcesAdd(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name     string
		current  BaseResources
		incoming BaseResources
		expected BaseResources
	}{
		{
			name:    "add to empty BaseResources resources",
			current: BaseResources{},
			incoming: BaseResources{
				Apps:          1,
				VolumeStorage: resource.MustParse("1Mi"),
			},
			expected: BaseResources{
				Apps:          1,
				VolumeStorage: resource.MustParse("1Mi"),
			},
		},
		{
			name: "add to existing BaseResources resources",
			current: BaseResources{
				Apps:          1,
				VolumeStorage: resource.MustParse("1Mi"),
			},
			incoming: BaseResources{
				Apps:          1,
				Images:        1,
				VolumeStorage: resource.MustParse("1Mi"),
				CPU:           resource.MustParse("20m"),
			},
			expected: BaseResources{
				Apps:          2,
				Images:        1,
				VolumeStorage: resource.MustParse("2Mi"),
				CPU:           resource.MustParse("20m"),
			},
		},
		{
			name: "add where current has a resource specified with unlimited",
			current: BaseResources{
				Apps:   Unlimited,
				Memory: UnlimitedQuantity(),
			},
			incoming: BaseResources{
				Apps:   1,
				Memory: resource.MustParse("1Mi"),
			},
			expected: BaseResources{
				Apps:   Unlimited,
				Memory: UnlimitedQuantity(),
			},
		},
		{
			name: "add where incoming has a resource specified with unlimited",
			current: BaseResources{
				Apps:   1,
				Memory: resource.MustParse("1Mi"),
			},
			incoming: BaseResources{
				Apps:   Unlimited,
				Memory: UnlimitedQuantity(),
			},
			expected: BaseResources{
				Apps:   Unlimited,
				Memory: UnlimitedQuantity(),
			},
		},
		{
			name: "add where current and incoming have a resource specified with unlimited",
			current: BaseResources{
				Apps:   Unlimited,
				Memory: UnlimitedQuantity(),
			},
			incoming: BaseResources{
				Apps:   Unlimited,
				Memory: UnlimitedQuantity(),
			},
			expected: BaseResources{
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

func TestBaseResourcesRemove(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name     string
		current  BaseResources
		incoming BaseResources
		all      bool
		expected BaseResources
	}{
		{
			name:    "remove from empty BaseResources resources",
			current: BaseResources{},
			incoming: BaseResources{
				Apps:   1,
				Memory: resource.MustParse("1Mi"),
			},
			expected: BaseResources{},
		},
		{
			name: "remove from existing BaseResources resources",
			current: BaseResources{
				Apps:   1,
				Memory: resource.MustParse("1Mi"),
			},
			incoming: BaseResources{
				Apps:   1,
				Memory: resource.MustParse("1Mi"),
			},
			expected: BaseResources{},
		},
		{
			name: "should never get negative values",
			all:  true,
			current: BaseResources{
				Apps:          1,
				Memory:        resource.MustParse("1Mi"),
				VolumeStorage: resource.MustParse("1Mi"),
			},
			incoming: BaseResources{
				Apps:          2,
				Memory:        resource.MustParse("2Mi"),
				VolumeStorage: resource.MustParse("2Mi"),
			},
			expected: BaseResources{},
		},
		{
			name: "remove persistent resources with all",
			current: BaseResources{
				VolumeStorage: resource.MustParse("1Mi"),
			},
			incoming: BaseResources{
				VolumeStorage: resource.MustParse("1Mi"),
			},
			all:      true,
			expected: BaseResources{},
		},
		{
			name: "does not remove persistent resources without all",
			current: BaseResources{
				VolumeStorage: resource.MustParse("1Mi"),
			},
			incoming: BaseResources{
				VolumeStorage: resource.MustParse("1Mi"),
			},
			expected: BaseResources{
				VolumeStorage: resource.MustParse("1Mi"),
			},
		},
		{
			name: "remove where current has a resource specified with unlimited",
			current: BaseResources{
				Apps:   Unlimited,
				Memory: UnlimitedQuantity(),
			},
			incoming: BaseResources{
				Apps:   1,
				Memory: resource.MustParse("1Mi"),
			},
			expected: BaseResources{
				Apps:   Unlimited,
				Memory: UnlimitedQuantity(),
			},
		},
		{
			name: "remove where incoming has a resource specified with unlimited",
			current: BaseResources{
				Apps:   1,
				Memory: resource.MustParse("1Mi"),
			},
			incoming: BaseResources{
				Apps:   Unlimited,
				Memory: UnlimitedQuantity(),
			},
			expected: BaseResources{
				Apps:   1,
				Memory: resource.MustParse("1Mi"),
			},
		},
		{
			name: "remove where current and incoming have a resource specified with unlimited",
			current: BaseResources{
				Apps:   Unlimited,
				Memory: UnlimitedQuantity(),
			},
			incoming: BaseResources{
				Apps:   Unlimited,
				Memory: UnlimitedQuantity(),
			},
			expected: BaseResources{
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

func TestBaseResourcesEquals(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name     string
		current  BaseResources
		incoming BaseResources
		expected bool
	}{
		{
			name:     "empty BaseResources resources",
			current:  BaseResources{},
			incoming: BaseResources{},
			expected: true,
		},
		{
			name: "equal BaseResources resources",
			current: BaseResources{
				Apps:          1,
				VolumeStorage: resource.MustParse("1Mi"),
			},
			incoming: BaseResources{
				Apps:          1,
				VolumeStorage: resource.MustParse("1Mi"),
			},
			expected: true,
		},
		{
			name: "unequal BaseResources resources",
			current: BaseResources{
				Apps:          1,
				VolumeStorage: resource.MustParse("1Mi"),
			},
			incoming: BaseResources{
				Apps:          2,
				VolumeStorage: resource.MustParse("1Mi"),
			},
			expected: false,
		},
		{
			name: "equal BaseResources resources with unlimited values",
			current: BaseResources{
				Apps:          Unlimited,
				VolumeStorage: UnlimitedQuantity(),
			},
			incoming: BaseResources{
				Apps:          Unlimited,
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

func TestBaseResourcesFits(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name        string
		current     BaseResources
		incoming    BaseResources
		expectedErr error
	}{
		{
			name:     "empty BaseResources resources",
			current:  BaseResources{},
			incoming: BaseResources{},
		},
		{
			name: "fits BaseResources",
			current: BaseResources{
				Apps:          1,
				VolumeStorage: resource.MustParse("1Mi"),
			},
			incoming: BaseResources{
				Apps:          1,
				VolumeStorage: resource.MustParse("1Mi"),
			},
		},

		{
			name: "does not fit BaseResources resources",
			current: BaseResources{
				Apps:          1,
				VolumeStorage: resource.MustParse("1Mi"),
			},
			incoming: BaseResources{
				Apps:          2,
				VolumeStorage: resource.MustParse("1Mi"),
			},
			expectedErr: ErrExceededResources,
		},
		{
			name: "fits BaseResources resources with specified unlimited values",
			current: BaseResources{
				Apps:          Unlimited,
				VolumeStorage: UnlimitedQuantity(),
			},
			incoming: BaseResources{
				Apps:          2,
				VolumeStorage: resource.MustParse("2Mi"),
			},
		},
		{
			name: "fits count BaseResources resources with specified unlimited values but not others",
			current: BaseResources{
				Jobs: 0,
				Apps: Unlimited,
			},
			incoming: BaseResources{
				Jobs: 2,
				Apps: 2,
			},
			expectedErr: ErrExceededResources,
		},

		{
			name: "fits quantity BaseResources resources with specified unlimited values but not others",
			current: BaseResources{
				VolumeStorage: UnlimitedQuantity(),
			},
			incoming: BaseResources{
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

func TestBaseResourcesToString(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name     string
		current  BaseResources
		expected string
	}{
		{
			name:     "empty BaseResources",
			current:  BaseResources{},
			expected: "",
		},
		{
			name: "populated BaseResources",
			current: BaseResources{
				Apps:          1,
				VolumeStorage: resource.MustParse("1Mi"),
			},
			expected: "Apps: 1, VolumeStorage: 1Mi",
		},
		{
			name: "populated BaseResources with unlimited values",
			current: BaseResources{
				Apps:          Unlimited,
				VolumeStorage: UnlimitedQuantity(),
			},
			expected: "Apps: unlimited, VolumeStorage: unlimited",
		},
	}

	// Run the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.current.ToString())
		})
	}
}
