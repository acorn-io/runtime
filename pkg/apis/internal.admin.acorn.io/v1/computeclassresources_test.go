package v1

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestComputeClassResourcesAdd(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name     string
		current  ComputeClassResources
		incoming ComputeClassResources
		expected ComputeClassResources
	}{
		{
			name:    "add to empty ComputeClassResources resources",
			current: ComputeClassResources{},
			incoming: ComputeClassResources{"foo": {
				Memory: resource.MustParse("1Mi"),
			}},
			expected: ComputeClassResources{"foo": {
				Memory: resource.MustParse("1Mi"),
			}},
		},
		{
			name: "add to existing ComputeClassResources resources",
			current: ComputeClassResources{"foo": {
				Memory: resource.MustParse("1Mi"),
			}},
			incoming: ComputeClassResources{"foo": {
				Memory: resource.MustParse("1Mi"),
			}},
			expected: ComputeClassResources{"foo": {
				Memory: resource.MustParse("2Mi"),
			}},
		},
		{
			name: "add where current has a resource specified with unlimited",
			current: ComputeClassResources{"foo": {
				Memory: UnlimitedQuantity(),
			}},
			incoming: ComputeClassResources{"foo": {
				Memory: resource.MustParse("1Mi"),
			}},
			expected: ComputeClassResources{"foo": {
				Memory: UnlimitedQuantity(),
			}},
		},
		{
			name: "add where incoming has a resource specified with unlimited",
			current: ComputeClassResources{"foo": {
				Memory: resource.MustParse("1Mi"),
			}},
			incoming: ComputeClassResources{"foo": {
				Memory: UnlimitedQuantity(),
			}},
			expected: ComputeClassResources{"foo": {
				Memory: UnlimitedQuantity(),
			}},
		},
		{
			name: "add where current and incoming have a resource specified with unlimited",
			current: ComputeClassResources{"foo": {
				Memory: UnlimitedQuantity(),
			}},
			incoming: ComputeClassResources{"foo": {
				Memory: UnlimitedQuantity(),
			}},
			expected: ComputeClassResources{"foo": {
				Memory: UnlimitedQuantity(),
			}},
		},
		{
			name: "add where current and incoming have AllComputeClasses specified at non-unlimited values",
			current: ComputeClassResources{AllComputeClasses: {
				Memory: resource.MustParse("1Mi"),
			}},
			incoming: ComputeClassResources{AllComputeClasses: {
				Memory: resource.MustParse("1Mi"),
			}},
			expected: ComputeClassResources{AllComputeClasses: {
				Memory: resource.MustParse("2Mi"),
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

func TestComputeClassResourcesRemove(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name     string
		current  ComputeClassResources
		incoming ComputeClassResources
		expected ComputeClassResources
	}{
		{
			name:    "remove from empty ComputeClassResources resources",
			current: ComputeClassResources{},
			incoming: ComputeClassResources{"foo": {
				Memory: resource.MustParse("1Mi"),
			}},
			expected: ComputeClassResources{},
		},
		{
			name: "resulting empty does not remove other non-empty ComputeClassResources resources",
			current: ComputeClassResources{
				"foo": {Memory: resource.MustParse("1Mi")},
				"bar": {Memory: resource.MustParse("2Mi")},
			},
			incoming: ComputeClassResources{
				"foo": {Memory: resource.MustParse("1Mi")},
				"bar": {Memory: resource.MustParse("1Mi")},
			},
			expected: ComputeClassResources{
				"bar": {Memory: resource.MustParse("1Mi")},
			},
		},
		{
			name: "remove from existing ComputeClassResources resources",
			current: ComputeClassResources{"foo": {
				Memory: resource.MustParse("1Mi"),
			}},
			incoming: ComputeClassResources{"foo": {
				Memory: resource.MustParse("1Mi"),
			}},
			expected: ComputeClassResources{},
		},
		{
			name: "should never get negative values",
			current: ComputeClassResources{"foo": {
				Memory: resource.MustParse("1Mi"),
			}},
			incoming: ComputeClassResources{"foo": {
				Memory: resource.MustParse("2Mi"),
			}},
			expected: ComputeClassResources{},
		},
		{
			name: "remove where current has a resource specified with unlimited",
			current: ComputeClassResources{"foo": {
				Memory: UnlimitedQuantity(),
			}},
			incoming: ComputeClassResources{"foo": {
				Memory: resource.MustParse("1Mi"),
			}},
			expected: ComputeClassResources{"foo": {
				Memory: UnlimitedQuantity(),
			}},
		},
		{
			name: "remove where incoming has a resource specified with unlimited",
			current: ComputeClassResources{"foo": {
				Memory: resource.MustParse("1Mi"),
			}},
			incoming: ComputeClassResources{"foo": {
				Memory: UnlimitedQuantity(),
			}},
			expected: ComputeClassResources{"foo": {
				Memory: resource.MustParse("1Mi"),
			}},
		},
		{
			name: "remove where current and incoming have a resource specified with unlimited",
			current: ComputeClassResources{"foo": {
				Memory: UnlimitedQuantity(),
			}},
			incoming: ComputeClassResources{"foo": {
				Memory: UnlimitedQuantity(),
			}},
			expected: ComputeClassResources{"foo": {
				Memory: UnlimitedQuantity(),
			}},
		},
		{
			name: "remove where current and incoming have a AllComputeClasses specified with non-unlimited values",
			current: ComputeClassResources{AllComputeClasses: {
				Memory: resource.MustParse("2Mi"),
			}},
			incoming: ComputeClassResources{AllComputeClasses: {
				Memory: resource.MustParse("1Mi"),
			}},
			expected: ComputeClassResources{AllComputeClasses: {
				Memory: resource.MustParse("1Mi"),
			}},
		},
		{
			name: "remove where current has two ComputeClasses and incoming has one",
			current: ComputeClassResources{
				"foo": {Memory: resource.MustParse("2Mi")},
				"bar": {Memory: resource.MustParse("2Mi")},
			},
			incoming: ComputeClassResources{
				"foo": {Memory: resource.MustParse("2Mi")},
			},
			expected: ComputeClassResources{
				"bar": {Memory: resource.MustParse("2Mi")},
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

func TestComputeClassResourcesEquals(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name     string
		current  ComputeClassResources
		incoming ComputeClassResources
		expected bool
	}{
		{
			name:     "empty ComputeClassResources resources",
			current:  ComputeClassResources{},
			incoming: ComputeClassResources{},
			expected: true,
		},
		{
			name: "equal ComputeClassResources resources",
			current: ComputeClassResources{"foo": {
				Memory: resource.MustParse("1Mi"),
			}},
			incoming: ComputeClassResources{"foo": {
				Memory: resource.MustParse("1Mi"),
			}},
			expected: true,
		},
		{
			name: "unequal ComputeClassResources resources",
			current: ComputeClassResources{"foo": {
				Memory: resource.MustParse("1Mi"),
			}},
			incoming: ComputeClassResources{"foo": {
				Memory: resource.MustParse("2Mi"),
			}},
			expected: false,
		},
		{
			name: "equal ComputeClassResources resources with unlimited values",
			current: ComputeClassResources{"foo": {
				Memory: UnlimitedQuantity(),
			}},
			incoming: ComputeClassResources{"foo": {
				Memory: UnlimitedQuantity(),
			}},
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

func TestComputeClassResourcesFits(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name        string
		current     ComputeClassResources
		incoming    ComputeClassResources
		expectedErr error
	}{
		{
			name:     "empty ComputeClassResources resources",
			current:  ComputeClassResources{},
			incoming: ComputeClassResources{},
		},
		{
			name: "fits ComputeClassResources",
			current: ComputeClassResources{"foo": {
				Memory: resource.MustParse("1Mi"),
			}},
			incoming: ComputeClassResources{"foo": {
				Memory: resource.MustParse("1Mi"),
			}},
		},
		{
			name: "fits when incoming is empty",
			current: ComputeClassResources{"foo": {
				Memory: resource.MustParse("1Mi"),
			}},
		},
		{
			name: "does not fit when current is empty",
			incoming: ComputeClassResources{"foo": {
				Memory: resource.MustParse("1Mi"),
			}},
			expectedErr: ErrExceededResources,
		},
		{
			name: "does not fit ComputeClassResources resources",
			current: ComputeClassResources{"foo": {
				Memory: resource.MustParse("1Mi"),
			}},
			incoming: ComputeClassResources{"foo": {
				Memory: resource.MustParse("2Mi"),
			}},
			expectedErr: ErrExceededResources,
		},
		{
			name: "fits ComputeClassResources resources with specified unlimited values",
			current: ComputeClassResources{"foo": {
				Memory: UnlimitedQuantity(),
			}},
			incoming: ComputeClassResources{"foo": {
				Memory: resource.MustParse("2Mi"),
			}},
		},
		{
			name: "fits quantity ComputeClassResources resources with specified unlimited values but not others",
			current: ComputeClassResources{"foo": {
				Memory: UnlimitedQuantity(),
				CPU:    resource.MustParse("1m"),
			}},
			incoming: ComputeClassResources{"foo": {
				Memory: resource.MustParse("2Mi"),
				CPU:    resource.MustParse("2m"),
			}},
			expectedErr: ErrExceededResources,
		},
		{
			name: "fits ComputeClassResources with AllComputeClasses specified but not others",
			current: ComputeClassResources{AllComputeClasses: {
				Memory: resource.MustParse("2Mi"),
			}},
			incoming: ComputeClassResources{"foo": {
				Memory: resource.MustParse("2Mi"),
			}},
		},
		{
			name: "fits ComputeClassResources with AllComputeClasses specified and others",
			current: ComputeClassResources{AllComputeClasses: {
				Memory: resource.MustParse("1Mi"),
			}},
			incoming: ComputeClassResources{
				"foo": {
					Memory: resource.MustParse("1Mi"),
				},
				"bar": {
					Memory: resource.MustParse("1Mi"),
				},
			},
		},
		{
			name: "fits ComputeClassResources with AllComputeClasses specified and others with unlimited set",
			current: ComputeClassResources{AllComputeClasses: {
				Memory: UnlimitedQuantity(),
			}},
			incoming: ComputeClassResources{
				"foo": {
					Memory: resource.MustParse("1Mi"),
				},
				"bar": {
					Memory: resource.MustParse("1Mi"),
				},
			},
		},
		{
			name: "does not fit ComputeClassResources with AllComputeClasses specified that is not enough",
			current: ComputeClassResources{AllComputeClasses: {
				Memory: resource.MustParse("1Mi"),
			}},
			incoming: ComputeClassResources{"foo": {
				Memory: resource.MustParse("2Mi"),
			}},
			expectedErr: ErrExceededResources,
		},
		{
			name: "does not fit ComputeClassResources with AllComputeClasses if one incoming exceeds the resources",
			current: ComputeClassResources{AllComputeClasses: {
				Memory: resource.MustParse("1Mi"),
			}},
			incoming: ComputeClassResources{
				"foo": {
					Memory: resource.MustParse("2Mi"),
				},
				"bar": {
					Memory: resource.MustParse("1Mi"),
				},
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

func TestComputeClassResourcesToString(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name     string
		current  ComputeClassResources
		expected string
	}{
		{
			name:     "empty ComputeClassResources",
			current:  ComputeClassResources{},
			expected: "",
		},
		{
			name: "populated ComputeClassResources",
			current: ComputeClassResources{"foo": {
				Memory: resource.MustParse("1Mi"),
				CPU:    resource.MustParse("1m"),
			}},
			expected: "\"foo\": { Memory: 1Mi, CPU: 1m }",
		},
		{
			name: "populated ComputeClassResources with unlimited values",
			current: ComputeClassResources{"foo": {
				Memory: UnlimitedQuantity(),
				CPU:    UnlimitedQuantity(),
			}},
			expected: "\"foo\": { Memory: unlimited, CPU: unlimited }",
		},
		{
			name: "multiple populated ComputeClassResources",
			current: ComputeClassResources{
				"foo": {
					Memory: resource.MustParse("1Mi"),
					CPU:    resource.MustParse("1m"),
				},
				"bar": {
					Memory: resource.MustParse("2Mi"),
					CPU:    resource.MustParse("2m"),
				},
			},
			expected: "\"bar\": { Memory: 2Mi, CPU: 2m }, \"foo\": { Memory: 1Mi, CPU: 1m }",
		},
	}

	// Run the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.current.ToString())
		})
	}
}
