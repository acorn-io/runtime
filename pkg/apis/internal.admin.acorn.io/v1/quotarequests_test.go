package v1

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestQuotaRequestResourcesAdd(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name     string
		current  QuotaRequestResources
		incoming QuotaRequestResources
		expected QuotaRequestResources
	}{
		{
			name:    "add to empty QuotaRequestResources",
			current: QuotaRequestResources{},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:          1,
					VolumeStorage: resource.MustParse("1Mi"),
				},
				Secrets: 1,
			},
			expected: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:          1,
					VolumeStorage: resource.MustParse("1Mi"),
				},
				Secrets: 1,
			},
		},
		{
			name: "add to existing QuotaRequestResources",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:          1,
					VolumeStorage: resource.MustParse("1Mi"),
				},
				Secrets: 1,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:          1,
					Images:        1,
					VolumeStorage: resource.MustParse("1Mi"),
					CPU:           resource.MustParse("20m"),
				},
				Secrets: 1,
			},
			expected: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:          2,
					Images:        1,
					VolumeStorage: resource.MustParse("2Mi"),
					CPU:           resource.MustParse("20m"),
				},
				Secrets: 2,
			},
		},
		{
			name: "add where current has a resource specified with unlimited",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:   Unlimited,
					Memory: UnlimitedQuantity(),
				},
				Secrets: Unlimited,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:   1,
					Memory: resource.MustParse("1Mi"),
				},
				Secrets: 1,
			},
			expected: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:   Unlimited,
					Memory: UnlimitedQuantity(),
				},
				Secrets: Unlimited,
			},
		},
		{
			name: "add where incoming has a resource specified with unlimited",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:   1,
					Memory: resource.MustParse("1Mi"),
				},
				Secrets: 1,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:   Unlimited,
					Memory: UnlimitedQuantity(),
				},
				Secrets: Unlimited,
			},
			expected: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:   Unlimited,
					Memory: UnlimitedQuantity(),
				},
				Secrets: Unlimited,
			},
		},
		{
			name: "add where current and incoming have a resource specified with unlimited",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:   Unlimited,
					Memory: UnlimitedQuantity(),
				},
				Secrets: Unlimited,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:   Unlimited,
					Memory: UnlimitedQuantity(),
				},
				Secrets: Unlimited,
			},
			expected: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:   Unlimited,
					Memory: UnlimitedQuantity(),
				},
				Secrets: Unlimited,
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
func TestQuotaRequestResourcesRemove(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name     string
		current  QuotaRequestResources
		incoming QuotaRequestResources
		all      bool
		expected QuotaRequestResources
	}{
		{
			name:    "remove from empty QuotaRequestResources",
			current: QuotaRequestResources{},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:   1,
					Memory: resource.MustParse("1Mi"),
				},
				Secrets: 1,
			},
			expected: QuotaRequestResources{},
		},
		{
			name: "should never get negative values",
			all:  true,
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:          1,
					Memory:        resource.MustParse("1Mi"),
					VolumeStorage: resource.MustParse("1Mi"),
				},
				Secrets: 1,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:          2,
					Memory:        resource.MustParse("2Mi"),
					VolumeStorage: resource.MustParse("2Mi"),
				},
				Secrets: 2,
			},
			expected: QuotaRequestResources{
				BaseResources: BaseResources{},
			},
		},
		{
			name: "remove resources counts with all",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					VolumeStorage: resource.MustParse("1Mi"),
				},
				Secrets: 1,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					VolumeStorage: resource.MustParse("1Mi"),
				},
				Secrets: 1,
			},
			all: true,
			expected: QuotaRequestResources{
				BaseResources: BaseResources{},
			},
		},
		{
			name: "does not remove resources counts without all",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					VolumeStorage: resource.MustParse("1Mi"),
				},
				Secrets: 1,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					VolumeStorage: resource.MustParse("1Mi"),
				},
				Secrets: 1,
			},
			expected: QuotaRequestResources{
				BaseResources: BaseResources{
					VolumeStorage: resource.MustParse("1Mi"),
				},
				Secrets: 1,
			},
		},
		{
			name: "remove where current has a resource specified with unlimited",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:   Unlimited,
					Memory: UnlimitedQuantity(),
				},
				Secrets: Unlimited,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:   1,
					Memory: resource.MustParse("1Mi"),
				},
				Secrets: 1,
			},
			expected: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:   Unlimited,
					Memory: UnlimitedQuantity(),
				},
				Secrets: Unlimited,
			},
		},
		{
			name: "remove where incoming has a resource specified with unlimited",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:   1,
					Memory: resource.MustParse("1Mi"),
				},
				Secrets: 1,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:   Unlimited,
					Memory: UnlimitedQuantity(),
				},
				Secrets: Unlimited,
			},
			expected: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:   1,
					Memory: resource.MustParse("1Mi"),
				},
				Secrets: 1,
			},
		},
		{
			name: "remove where current and incoming have a resource specified with unlimited",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:   Unlimited,
					Memory: UnlimitedQuantity(),
				},
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:   Unlimited,
					Memory: UnlimitedQuantity(),
				},
			},
			expected: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:   Unlimited,
					Memory: UnlimitedQuantity(),
				},
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
func TestQuotaRequestResourcesEquals(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name     string
		current  QuotaRequestResources
		incoming QuotaRequestResources
		expected bool
	}{
		{
			name:     "empty QuotaRequestResources",
			current:  QuotaRequestResources{},
			incoming: QuotaRequestResources{},
			expected: true,
		},
		{
			name: "equal QuotaRequestResources",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:          1,
					VolumeStorage: resource.MustParse("1Mi"),
				},
				Secrets: 1,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:          1,
					VolumeStorage: resource.MustParse("1Mi"),
				},
				Secrets: 1,
			},
			expected: true,
		},
		{
			name: "unequal QuotaRequestResources only",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:          1,
					VolumeStorage: resource.MustParse("1Mi"),
				},
				Secrets: 1,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:          1,
					VolumeStorage: resource.MustParse("1Mi"),
				},
			},
			expected: false,
		},
		{
			name: "unequal base resources only",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:          1,
					Containers:    1,
					VolumeStorage: resource.MustParse("1Mi"),
				},
				Secrets: 1,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:          1,
					VolumeStorage: resource.MustParse("1Mi"),
				},
				Secrets: 1,
			},
			expected: false,
		},
		{
			name: "unequal QuotaRequestResources",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:          1,
					VolumeStorage: resource.MustParse("1Mi"),
				},
				Secrets: 1,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:          2,
					VolumeStorage: resource.MustParse("1Mi"),
				},
				Secrets: 2,
			},
			expected: false,
		},
		{
			name: "equal QuotaRequestResources with unlimited values",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:          Unlimited,
					VolumeStorage: UnlimitedQuantity(),
				},
				Secrets: Unlimited,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:          Unlimited,
					VolumeStorage: UnlimitedQuantity(),
				},
				Secrets: Unlimited,
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
func TestQuotaRequestResourcesFits(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name        string
		current     QuotaRequestResources
		incoming    QuotaRequestResources
		expectedErr error
	}{
		{
			name:     "empty QuotaRequestResources",
			current:  QuotaRequestResources{},
			incoming: QuotaRequestResources{},
		},
		{
			name: "fits BaseResources",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:          1,
					VolumeStorage: resource.MustParse("1Mi"),
				},
				Secrets: 1,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:          1,
					VolumeStorage: resource.MustParse("1Mi"),
				},
				Secrets: 1,
			},
		},
		{
			name: "does not fit QuotaRequestResources",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:          1,
					VolumeStorage: resource.MustParse("1Mi"),
				},
				Secrets: 1,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:          2,
					VolumeStorage: resource.MustParse("1Mi"),
				},
				Secrets: 2,
			},
			expectedErr: ErrExceededResources,
		},
		{
			name: "false as expected with only QuotaRequestResources",
			current: QuotaRequestResources{
				Secrets: 1,
			},
			incoming: QuotaRequestResources{
				Secrets: 2,
			},
			expectedErr: ErrExceededResources,
		},
		{
			name: "false as expected with only base resources",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:          1,
					VolumeStorage: resource.MustParse("1Mi"),
				},
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:          2,
					VolumeStorage: resource.MustParse("1Mi"),
				},
			},
			expectedErr: ErrExceededResources,
		},
		{
			name: "fits QuotaRequestResources with specified unlimited values",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:          Unlimited,
					VolumeStorage: UnlimitedQuantity(),
				},
				Secrets: Unlimited,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:          2,
					VolumeStorage: resource.MustParse("2Mi"),
				},
				Secrets: 2,
			},
		},
		{
			name: "fits count QuotaRequestResources with specified unlimited values but not others",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Jobs: 0,
					Apps: Unlimited,
				},
				Secrets: Unlimited,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Jobs: 2,
					Apps: 2,
				},
				Secrets: 2,
			},
			expectedErr: ErrExceededResources,
		},
		{
			name: "fits quantity QuotaRequestResources with specified unlimited values but not others",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					VolumeStorage: UnlimitedQuantity(),
				},
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					CPU:           resource.MustParse("100m"),
					VolumeStorage: resource.MustParse("2Mi"),
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

func TestQuotaRequestResourcesToString(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name     string
		current  QuotaRequestResources
		expected string
	}{
		{
			name:     "empty BaseResources",
			current:  QuotaRequestResources{},
			expected: "",
		},
		{
			name: "populated BaseResources",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:          1,
					VolumeStorage: resource.MustParse("1Mi"),
				},
				Secrets: 1,
			},
			expected: "Secrets: 1, Apps: 1, VolumeStorage: 1Mi",
		},
		{
			name: "populated BaseResources with unlimited values",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:          Unlimited,
					VolumeStorage: UnlimitedQuantity(),
				},
				Secrets: Unlimited,
			},
			expected: "Secrets: unlimited, Apps: unlimited, VolumeStorage: unlimited",
		},
	}

	// Run the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.current.ToString())
		})
	}
}
