package v1

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestVolumeClassResourcesAdd(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name     string
		current  VolumeClassResources
		incoming VolumeClassResources
		expected VolumeClassResources
	}{
		{
			name:     "add to empty VolumeClassResources resources",
			current:  VolumeClassResources{},
			incoming: VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
			expected: VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
		},
		{
			name:     "add to existing VolumeClassResources resources",
			current:  VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
			incoming: VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
			expected: VolumeClassResources{"foo": {resource.MustParse("2Mi")}},
		},
		{
			name:     "add where current has a resource specified with unlimited",
			current:  VolumeClassResources{"foo": {UnlimitedQuantity()}},
			incoming: VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
			expected: VolumeClassResources{"foo": {UnlimitedQuantity()}},
		},
		{
			name:     "add where incoming has a resource specified with unlimited",
			current:  VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
			incoming: VolumeClassResources{"foo": {UnlimitedQuantity()}},
			expected: VolumeClassResources{"foo": {UnlimitedQuantity()}},
		},
		{
			name:     "add where current and incoming have a resource specified with unlimited",
			current:  VolumeClassResources{"foo": {UnlimitedQuantity()}},
			incoming: VolumeClassResources{"foo": {UnlimitedQuantity()}},
			expected: VolumeClassResources{"foo": {UnlimitedQuantity()}},
		},
		{
			name: "add where current and incoming have a AllVolumeClasses specified with non-unlimited values",
			current: VolumeClassResources{AllVolumeClasses: {
				VolumeStorage: resource.MustParse("1Mi"),
			}},
			incoming: VolumeClassResources{AllVolumeClasses: {
				VolumeStorage: resource.MustParse("1Mi"),
			}},
			expected: VolumeClassResources{AllVolumeClasses: {
				VolumeStorage: resource.MustParse("2Mi"),
			}},
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

func TestVolumeClassResourcesRemove(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name     string
		current  VolumeClassResources
		incoming VolumeClassResources
		expected VolumeClassResources
	}{
		{
			name:     "remove from empty VolumeClassResources resources",
			current:  VolumeClassResources{},
			incoming: VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
			expected: VolumeClassResources{},
		},
		{
			name:     "remove from existing VolumeClassResources resources",
			current:  VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
			incoming: VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
			expected: VolumeClassResources{},
		},
		{
			name:     "should never get negative values",
			current:  VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
			incoming: VolumeClassResources{"foo": {resource.MustParse("2Mi")}},
			expected: VolumeClassResources{},
		},
		{
			name:     "remove where current has a resource specified with unlimited",
			current:  VolumeClassResources{"foo": {UnlimitedQuantity()}},
			incoming: VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
			expected: VolumeClassResources{"foo": {UnlimitedQuantity()}},
		},
		{
			name:     "remove where incoming has a resource specified with unlimited",
			current:  VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
			incoming: VolumeClassResources{"foo": {UnlimitedQuantity()}},
			expected: VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
		},
		{
			name:     "remove where current and incoming have a resource specified with unlimited",
			current:  VolumeClassResources{"foo": {UnlimitedQuantity()}},
			incoming: VolumeClassResources{"foo": {UnlimitedQuantity()}},
			expected: VolumeClassResources{"foo": {UnlimitedQuantity()}},
		},
		{
			name: "remove where current and incoming have a AllVolumeClasses specified with non-unlimited values",
			current: VolumeClassResources{AllVolumeClasses: {
				VolumeStorage: resource.MustParse("2Mi"),
			}},
			incoming: VolumeClassResources{AllVolumeClasses: {
				VolumeStorage: resource.MustParse("1Mi"),
			}},
			expected: VolumeClassResources{AllVolumeClasses: {
				VolumeStorage: resource.MustParse("1Mi"),
			}},
		},
		{
			name: "remove where current has two volume classes and incoming has one",
			current: VolumeClassResources{
				"foo": {resource.MustParse("2Mi")},
				"bar": {resource.MustParse("2Mi")},
			},
			incoming: VolumeClassResources{
				"foo": {resource.MustParse("2Mi")},
			},
			expected: VolumeClassResources{
				"bar": {resource.MustParse("2Mi")},
			},
		},
	}

	// Run the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.current.Remove(tc.incoming)
			assert.True(t, tc.current.Equals(tc.expected))
		})
	}
}

func TestVolumeClassResourcesEquals(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name     string
		current  VolumeClassResources
		incoming VolumeClassResources
		expected bool
	}{
		{
			name:     "empty VolumeClassResources resources",
			current:  VolumeClassResources{},
			incoming: VolumeClassResources{},
			expected: true,
		},
		{
			name:     "equal VolumeClassResources resources",
			current:  VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
			incoming: VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
			expected: true,
		},
		{
			name:     "unequal VolumeClassResources resources",
			current:  VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
			incoming: VolumeClassResources{"foo": {resource.MustParse("2Mi")}},
			expected: false,
		},
		{
			name:     "equal VolumeClassResources resources with unlimited values",
			current:  VolumeClassResources{"foo": {UnlimitedQuantity()}},
			incoming: VolumeClassResources{"foo": {UnlimitedQuantity()}},
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

func TestVolumeClassResourcesFits(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name        string
		current     VolumeClassResources
		incoming    VolumeClassResources
		expectedErr error
	}{
		{
			name:     "empty VolumeClassResources resources",
			current:  VolumeClassResources{},
			incoming: VolumeClassResources{},
		},
		{
			name:     "fits VolumeClassResources",
			current:  VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
			incoming: VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
		},

		{
			name:        "does not fit VolumeClassResources resources",
			current:     VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
			incoming:    VolumeClassResources{"foo": {resource.MustParse("2Mi")}},
			expectedErr: ErrExceededResources,
		},
		{
			name:     "fits VolumeClassResources resources with specified unlimited values",
			current:  VolumeClassResources{"foo": {UnlimitedQuantity()}},
			incoming: VolumeClassResources{"foo": {resource.MustParse("2Mi")}},
		},
		{
			name:     "fits VolumeClassResources with AllVolumeClasses specified but not others",
			current:  VolumeClassResources{AllVolumeClasses: {resource.MustParse("2Mi")}},
			incoming: VolumeClassResources{"foo": {resource.MustParse("2Mi")}},
		},
		{
			name:    "fits VolumeClassResources with AllVolumeClasses specified and others",
			current: VolumeClassResources{AllVolumeClasses: {resource.MustParse("1Mi")}},
			incoming: VolumeClassResources{
				"foo": {resource.MustParse("1Mi")},
				"bar": {resource.MustParse("1Mi")},
			},
		},
		{
			name:    "fits VolumeClassResources with AllVolumeClasses specified and others with unlimited set",
			current: VolumeClassResources{AllVolumeClasses: {UnlimitedQuantity()}},
			incoming: VolumeClassResources{
				"foo": {resource.MustParse("1Mi")},
				"bar": {resource.MustParse("1Mi")},
			},
		},
		{
			name:        "does not fit VolumeClassResources with AllVolumeClasses specified that is not enough",
			current:     VolumeClassResources{AllVolumeClasses: {resource.MustParse("1Mi")}},
			incoming:    VolumeClassResources{"foo": {resource.MustParse("2Mi")}},
			expectedErr: ErrExceededResources,
		},
		{
			name:    "does not fit VolumeClassResources with AllVolumeClasses if one incoming exceeds the resources",
			current: VolumeClassResources{AllVolumeClasses: {resource.MustParse("1Mi")}},
			incoming: VolumeClassResources{
				"foo": {resource.MustParse("2Mi")},
				"bar": {resource.MustParse("1Mi")},
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

func TestVolumeClassResourcesToString(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name     string
		current  VolumeClassResources
		expected string
	}{
		{
			name:     "empty VolumeClassResources",
			current:  VolumeClassResources{},
			expected: "",
		},
		{
			name:     "populated VolumeClassResources",
			current:  VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
			expected: "\"foo\": { VolumeStorage: 1Mi }",
		},
		{
			name:     "populated VolumeClassResources with unlimited values",
			current:  VolumeClassResources{"foo": {UnlimitedQuantity()}},
			expected: "\"foo\": { VolumeStorage: unlimited }"},
	}

	// Run the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.current.ToString())
		})
	}
}
