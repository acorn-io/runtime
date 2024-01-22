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
			name:     "add to empty BaseResources resources",
			current:  BaseResources{},
			incoming: BaseResources{Apps: 1},
			expected: BaseResources{Apps: 1},
		},
		{
			name:    "add to existing BaseResources resources",
			current: BaseResources{Apps: 1},
			incoming: BaseResources{
				Apps:   1,
				Images: 1,
			},
			expected: BaseResources{
				Apps:   2,
				Images: 1,
			},
		},
		{
			name:     "add where current has a resource specified with unlimited",
			current:  BaseResources{Apps: Unlimited},
			incoming: BaseResources{Apps: 1},
			expected: BaseResources{Apps: Unlimited},
		},
		{
			name:     "add where incoming has a resource specified with unlimited",
			current:  BaseResources{Apps: 1},
			incoming: BaseResources{Apps: Unlimited},
			expected: BaseResources{Apps: Unlimited},
		},
		{
			name:     "add where current and incoming have a resource specified with unlimited",
			current:  BaseResources{Apps: Unlimited},
			incoming: BaseResources{Apps: Unlimited},
			expected: BaseResources{Apps: Unlimited},
		},
		{
			name: "add where current and incoming have ComputeClasses and VolumeClasses",
			current: BaseResources{
				Apps: 1, Containers: 1,
				ComputeClasses: ComputeClassResources{"foo": {
					Memory: resource.MustParse("1Mi"),
					CPU:    resource.MustParse("1m"),
				}},
				VolumeClasses: VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
			},
			incoming: BaseResources{
				Apps: 1, Containers: 1,
				ComputeClasses: ComputeClassResources{"foo": {
					Memory: resource.MustParse("1Mi"),
					CPU:    resource.MustParse("1m"),
				}},
				VolumeClasses: VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
			},
			expected: BaseResources{
				Apps: 2, Containers: 2,
				ComputeClasses: ComputeClassResources{"foo": {
					Memory: resource.MustParse("2Mi"),
					CPU:    resource.MustParse("2m"),
				}},
				VolumeClasses: VolumeClassResources{"foo": {resource.MustParse("2Mi")}},
			},
		},
		{
			name:    "add where current is empty and incoming has ComputeClasses and VolumeClasses",
			current: BaseResources{},
			incoming: BaseResources{
				Apps: 1, Containers: 1,
				ComputeClasses: ComputeClassResources{"foo": {
					Memory: resource.MustParse("1Mi"),
					CPU:    resource.MustParse("1m"),
				}},
				VolumeClasses: VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
			},
			expected: BaseResources{
				Apps: 1, Containers: 1,
				ComputeClasses: ComputeClassResources{"foo": {
					Memory: resource.MustParse("1Mi"),
					CPU:    resource.MustParse("1m"),
				}},
				VolumeClasses: VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
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
			name:     "remove from empty BaseResources resources",
			current:  BaseResources{},
			incoming: BaseResources{Apps: 1},
			expected: BaseResources{},
		},
		{
			name:     "remove from existing BaseResources resources",
			current:  BaseResources{Apps: 1},
			incoming: BaseResources{Apps: 1},
			expected: BaseResources{},
		},
		{
			name:     "should never get negative values",
			all:      true,
			current:  BaseResources{Apps: 1},
			incoming: BaseResources{Apps: 2},
			expected: BaseResources{},
		},
		{
			name:     "remove where current has a resource specified with unlimited",
			current:  BaseResources{Apps: Unlimited},
			incoming: BaseResources{Apps: 1},
			expected: BaseResources{Apps: Unlimited},
		},
		{
			name:     "remove where incoming has a resource specified with unlimited",
			current:  BaseResources{Apps: 1},
			incoming: BaseResources{Apps: Unlimited},
			expected: BaseResources{Apps: 1},
		},
		{
			name:     "remove where current and incoming have a resource specified with unlimited",
			current:  BaseResources{Apps: Unlimited},
			incoming: BaseResources{Apps: Unlimited},
			expected: BaseResources{Apps: Unlimited},
		},
		{
			name: "remove where current and incoming have ComputeClasses and VolumeClasses",
			current: BaseResources{
				Apps: 1, Containers: 1,
				ComputeClasses: ComputeClassResources{"foo": {
					Memory: resource.MustParse("1Mi"),
					CPU:    resource.MustParse("1m"),
				}},
				VolumeClasses: VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
			},
			incoming: BaseResources{
				Apps: 1, Containers: 1,
				ComputeClasses: ComputeClassResources{"foo": {
					Memory: resource.MustParse("1Mi"),
					CPU:    resource.MustParse("1m"),
				}},
				VolumeClasses: VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
			},
			all:      true,
			expected: BaseResources{},
		},
		{
			name: "does not remove volume storage when all is false",
			expected: BaseResources{
				VolumeClasses: VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
			},
			current: BaseResources{
				VolumeClasses: VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
			},
			incoming: BaseResources{
				VolumeClasses: VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
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
			name:     "equal BaseResources resources",
			current:  BaseResources{Apps: 1},
			incoming: BaseResources{Apps: 1},
			expected: true,
		},
		{
			name:     "unequal BaseResources resources",
			current:  BaseResources{Apps: 1},
			incoming: BaseResources{Apps: 2},
			expected: false,
		},
		{
			name:     "equal BaseResources resources with unlimited values",
			current:  BaseResources{Apps: Unlimited},
			incoming: BaseResources{Apps: Unlimited},
			expected: true,
		},
		{
			name: "equal BaseResources with ComputeClasses and VolumeClasses",
			current: BaseResources{
				Apps: 1, Containers: 1,
				ComputeClasses: ComputeClassResources{"foo": {
					Memory: resource.MustParse("1Mi"),
					CPU:    resource.MustParse("1m"),
				}},
				VolumeClasses: VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
			},
			incoming: BaseResources{
				Apps: 1, Containers: 1,
				ComputeClasses: ComputeClassResources{"foo": {
					Memory: resource.MustParse("1Mi"),
					CPU:    resource.MustParse("1m"),
				}},
				VolumeClasses: VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
			},
			expected: true,
		},
		{
			name: "unequal BaseResources with ComputeClasses and VolumeClasses",
			current: BaseResources{
				Apps: 1, Containers: 1,
				ComputeClasses: ComputeClassResources{"foo": {
					Memory: resource.MustParse("1Mi"),
					CPU:    resource.MustParse("1m"),
				}},
				VolumeClasses: VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
			},
			incoming: BaseResources{
				Apps: 1, Containers: 1,
				ComputeClasses: ComputeClassResources{"foo": {
					Memory: resource.MustParse("2Mi"),
					CPU:    resource.MustParse("1m"),
				}},
				VolumeClasses: VolumeClassResources{"foo": {resource.MustParse("2Mi")}},
			},
			expected: false,
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
			name:     "fits BaseResources",
			current:  BaseResources{Apps: 1},
			incoming: BaseResources{Apps: 1},
		},

		{
			name:        "does not fit BaseResources resources",
			current:     BaseResources{Apps: 1},
			incoming:    BaseResources{Apps: 2},
			expectedErr: ErrExceededResources,
		},
		{
			name:     "fits BaseResources resources with specified unlimited values",
			current:  BaseResources{Apps: Unlimited},
			incoming: BaseResources{Apps: 2},
		},
		{
			name:        "fits count BaseResources resources with specified unlimited values but not others",
			current:     BaseResources{Jobs: 0, Apps: Unlimited},
			incoming:    BaseResources{Jobs: 2, Apps: 2},
			expectedErr: ErrExceededResources,
		},
		{
			name: "fits BaseResources with ComputeClasses and VolumeClasses",
			current: BaseResources{
				Apps: 1, Containers: 1,
				ComputeClasses: ComputeClassResources{"foo": {
					Memory: resource.MustParse("1Mi"),
					CPU:    resource.MustParse("1m"),
				}},
				VolumeClasses: VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
			},
			incoming: BaseResources{
				Apps: 1, Containers: 1,
				ComputeClasses: ComputeClassResources{"foo": {
					Memory: resource.MustParse("1Mi"),
					CPU:    resource.MustParse("1m"),
				}},
				VolumeClasses: VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
			},
		},
		{
			name: "does not fit exceeding ComputeClasses and VolumeClasses",
			current: BaseResources{
				Apps: 1, Containers: 1,
				ComputeClasses: ComputeClassResources{"foo": {
					Memory: resource.MustParse("1Mi"),
					CPU:    resource.MustParse("1m"),
				}},
				VolumeClasses: VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
			},
			incoming: BaseResources{
				Apps: 1, Containers: 1,
				ComputeClasses: ComputeClassResources{"foo": {
					Memory: resource.MustParse("2Mi"),
					CPU:    resource.MustParse("2m"),
				}},
				VolumeClasses: VolumeClassResources{"foo": {resource.MustParse("2Mi")}},
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
			name:     "populated BaseResources",
			current:  BaseResources{Apps: 1, Containers: 1},
			expected: "Apps: 1, Containers: 1",
		},
		{
			name:     "populated BaseResources with unlimited values",
			current:  BaseResources{Apps: Unlimited, Containers: 1},
			expected: "Apps: unlimited, Containers: 1",
		},
		{
			name: "populated with ComputeClasses and VolumeClasses",
			current: BaseResources{
				Apps: 1, Containers: 1,
				ComputeClasses: ComputeClassResources{"foo": {
					Memory: resource.MustParse("1Mi"),
					CPU:    resource.MustParse("1m"),
				}},
				VolumeClasses: VolumeClassResources{"foo": {resource.MustParse("1Mi")}},
			},
			expected: "Apps: 1, Containers: 1, ComputeClasses: \"foo\": { Memory: 1Mi, CPU: 1m }, VolumeClasses: \"foo\": { VolumeStorage: 1Mi }",
		},
		{
			name: "populated with multiple ComputeClasses and VolumeClasses",
			current: BaseResources{
				Apps: 1, Containers: 1,
				ComputeClasses: ComputeClassResources{
					"foo": {
						Memory: resource.MustParse("1Mi"),
						CPU:    resource.MustParse("1m"),
					},
					"bar": {
						Memory: resource.MustParse("2Mi"),
						CPU:    resource.MustParse("2m"),
					},
				},
				VolumeClasses: VolumeClassResources{
					"foo": {resource.MustParse("1Mi")},
					"bar": {resource.MustParse("2Mi")},
				},
			},
			expected: "Apps: 1, Containers: 1, ComputeClasses: \"bar\": { Memory: 2Mi, CPU: 2m }, \"foo\": { Memory: 1Mi, CPU: 1m }, VolumeClasses: \"bar\": { VolumeStorage: 2Mi }, \"foo\": { VolumeStorage: 1Mi }",
		},
	}

	// Run the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.current.ToString())
		})
	}
}
