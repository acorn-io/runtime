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
					Apps:           1,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: resource.MustParse("1Mi")}},
					VolumeClasses:  VolumeClassResources{"volume-class": {resource.MustParse("1Mi")}},
				},
				Secrets: 1,
			},
			expected: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           1,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: resource.MustParse("1Mi")}},
					VolumeClasses:  VolumeClassResources{"volume-class": {resource.MustParse("1Mi")}},
				},
				Secrets: 1,
			},
		},
		{
			name: "add to existing QuotaRequestResources",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           1,
					ComputeClasses: ComputeClassResources{"compute-class": {CPU: resource.MustParse("20m")}},
					VolumeClasses:  VolumeClassResources{"volume-class": {resource.MustParse("1Mi")}},
				},
				Secrets: 1,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           1,
					Images:         1,
					ComputeClasses: ComputeClassResources{"compute-class": {CPU: resource.MustParse("20m")}},
					VolumeClasses:  VolumeClassResources{"volume-class": {resource.MustParse("1Mi")}},
				},
				Secrets: 1,
			},
			expected: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           2,
					Images:         1,
					ComputeClasses: ComputeClassResources{"compute-class": {CPU: resource.MustParse("40m")}},
					VolumeClasses:  VolumeClassResources{"volume-class": {resource.MustParse("2Mi")}},
				},
				Secrets: 2,
			},
		},
		{
			name: "add where current has a resource specified with unlimited",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           Unlimited,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: UnlimitedQuantity()}},
					VolumeClasses:  VolumeClassResources{"volume-class": {UnlimitedQuantity()}},
				},
				Secrets: Unlimited,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           1,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: resource.MustParse("1Mi")}},
					VolumeClasses:  VolumeClassResources{"volume-class": {resource.MustParse("1Mi")}},
				},
				Secrets: 1,
			},
			expected: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           Unlimited,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: UnlimitedQuantity()}},
					VolumeClasses:  VolumeClassResources{"volume-class": {UnlimitedQuantity()}},
				},
				Secrets: Unlimited,
			},
		},
		{
			name: "add where incoming has a resource specified with unlimited",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           1,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: resource.MustParse("1Mi")}},
					VolumeClasses:  VolumeClassResources{"volume-class": {resource.MustParse("1Mi")}},
				},
				Secrets: 1,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           Unlimited,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: UnlimitedQuantity()}},
					VolumeClasses:  VolumeClassResources{"volume-class": {UnlimitedQuantity()}},
				},
				Secrets: Unlimited,
			},
			expected: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           Unlimited,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: UnlimitedQuantity()}},
					VolumeClasses:  VolumeClassResources{"volume-class": {UnlimitedQuantity()}},
				},
				Secrets: Unlimited,
			},
		},
		{
			name: "add where current and incoming have a resource specified with unlimited",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           Unlimited,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: UnlimitedQuantity()}},
					VolumeClasses:  VolumeClassResources{"volume-class": {UnlimitedQuantity()}},
				},
				Secrets: Unlimited,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           Unlimited,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: UnlimitedQuantity()}},
					VolumeClasses:  VolumeClassResources{"volume-class": {UnlimitedQuantity()}},
				},
				Secrets: Unlimited,
			},
			expected: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           Unlimited,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: UnlimitedQuantity()}},
					VolumeClasses:  VolumeClassResources{"volume-class": {UnlimitedQuantity()}},
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
					Apps:           1,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: resource.MustParse("1Mi")}},
					VolumeClasses:  VolumeClassResources{"volume-class": {resource.MustParse("1Mi")}},
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
					Apps:           1,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: resource.MustParse("1Mi")}},
					VolumeClasses:  VolumeClassResources{"volume-class": {resource.MustParse("1Mi")}},
				},
				Secrets: 1,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           2,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: resource.MustParse("2Mi")}},
					VolumeClasses:  VolumeClassResources{"volume-class": {resource.MustParse("2Mi")}},
				},
				Secrets: 2,
			},
			expected: QuotaRequestResources{
				BaseResources: BaseResources{},
			},
		},
		{
			name: "removes persistent resources with all",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: resource.MustParse("1Mi")}},
					VolumeClasses:  VolumeClassResources{"volume-class": {resource.MustParse("1Mi")}},
				},
				Secrets: 1,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: resource.MustParse("1Mi")}},
					VolumeClasses:  VolumeClassResources{"volume-class": {resource.MustParse("1Mi")}},
				},
				Secrets: 1,
			},
			all: true,
			expected: QuotaRequestResources{
				BaseResources: BaseResources{},
			},
		},
		{
			name: "does not remove persistent resources without all",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: resource.MustParse("1Mi")}},
					VolumeClasses:  VolumeClassResources{"volume-class": {resource.MustParse("1Mi")}},
				},
				Secrets: 1,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: resource.MustParse("1Mi")}},
					VolumeClasses:  VolumeClassResources{"volume-class": {resource.MustParse("1Mi")}},
				},
				Secrets: 1,
			},
			expected: QuotaRequestResources{
				BaseResources: BaseResources{
					VolumeClasses: VolumeClassResources{"volume-class": {resource.MustParse("1Mi")}},
				},
				Secrets: 1,
			},
		},
		{
			name: "remove where current has a resource specified with unlimited",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           Unlimited,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: UnlimitedQuantity()}},
					VolumeClasses:  VolumeClassResources{"volume-class": {UnlimitedQuantity()}},
				},
				Secrets: Unlimited,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           1,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: UnlimitedQuantity()}},
					VolumeClasses:  VolumeClassResources{"volume-class": {UnlimitedQuantity()}},
				},
				Secrets: 1,
			},
			expected: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           Unlimited,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: UnlimitedQuantity()}},
					VolumeClasses:  VolumeClassResources{"volume-class": {UnlimitedQuantity()}},
				},
				Secrets: Unlimited,
			},
		},
		{
			name: "remove where incoming has a resource specified with unlimited",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           1,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: resource.MustParse("1Mi")}},
					VolumeClasses:  VolumeClassResources{"volume-class": {resource.MustParse("1Mi")}},
				},
				Secrets: 1,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           Unlimited,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: UnlimitedQuantity()}},
					VolumeClasses:  VolumeClassResources{"volume-class": {UnlimitedQuantity()}},
				},
				Secrets: Unlimited,
			},
			expected: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           1,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: resource.MustParse("1Mi")}},
					VolumeClasses:  VolumeClassResources{"volume-class": {resource.MustParse("1Mi")}},
				},
				Secrets: 1,
			},
		},
		{
			name: "remove where current and incoming have a resource specified with unlimited",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           Unlimited,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: UnlimitedQuantity()}},
					VolumeClasses:  VolumeClassResources{"volume-class": {UnlimitedQuantity()}},
				},
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           Unlimited,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: UnlimitedQuantity()}},
					VolumeClasses:  VolumeClassResources{"volume-class": {UnlimitedQuantity()}},
				},
			},
			expected: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           Unlimited,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: UnlimitedQuantity()}},
					VolumeClasses:  VolumeClassResources{"volume-class": {UnlimitedQuantity()}},
				},
			},
		},
		{
			name: "remove where current has two computeclasses and volumeclasses where incoming has one",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					ComputeClasses: ComputeClassResources{
						"compute-class-1": {Memory: resource.MustParse("1Mi")},
						"compute-class-2": {Memory: resource.MustParse("1Mi")},
					},
					VolumeClasses: VolumeClassResources{
						"volume-class-1": {resource.MustParse("1Mi")},
						"volume-class-2": {resource.MustParse("1Mi")},
					},
				},
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					ComputeClasses: ComputeClassResources{
						"compute-class-1": {Memory: resource.MustParse("1Mi")},
					},
					VolumeClasses: VolumeClassResources{
						"volume-class-1": {resource.MustParse("1Mi")},
					},
				},
			},
			all: true,
			expected: QuotaRequestResources{
				BaseResources: BaseResources{
					ComputeClasses: ComputeClassResources{
						"compute-class-2": {Memory: resource.MustParse("1Mi")},
					},
					VolumeClasses: VolumeClassResources{
						"volume-class-2": {resource.MustParse("1Mi")},
					},
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
					Apps:           1,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: resource.MustParse("1Mi")}},
					VolumeClasses:  VolumeClassResources{"volume-class": {resource.MustParse("1Mi")}},
				},
				Secrets: 1,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           1,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: resource.MustParse("1Mi")}},
					VolumeClasses:  VolumeClassResources{"volume-class": {resource.MustParse("1Mi")}},
				},
				Secrets: 1,
			},
			expected: true,
		},
		{
			name: "unequal QuotaRequestResources only",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           1,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: resource.MustParse("1Mi")}},
					VolumeClasses:  VolumeClassResources{"volume-class": {resource.MustParse("1Mi")}},
				},
				Secrets: 1,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           1,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: resource.MustParse("1Mi")}},
					VolumeClasses:  VolumeClassResources{"volume-class": {resource.MustParse("1Mi")}},
				},
			},
			expected: false,
		},
		{
			name: "unequal base resources only",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           1,
					Containers:     1,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: resource.MustParse("1Mi")}},
					VolumeClasses:  VolumeClassResources{"volume-class": {resource.MustParse("1Mi")}},
				},
				Secrets: 1,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           1,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: resource.MustParse("1Mi")}},
					VolumeClasses:  VolumeClassResources{"volume-class": {resource.MustParse("1Mi")}},
				},
				Secrets: 1,
			},
			expected: false,
		},
		{
			name: "unequal QuotaRequestResources",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           1,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: resource.MustParse("1Mi")}},
					VolumeClasses:  VolumeClassResources{"volume-class": {resource.MustParse("1Mi")}},
				},
				Secrets: 1,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           2,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: resource.MustParse("1Mi")}},
					VolumeClasses:  VolumeClassResources{"volume-class": {resource.MustParse("1Mi")}},
				},
				Secrets: 2,
			},
			expected: false,
		},
		{
			name: "equal QuotaRequestResources with unlimited values",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           Unlimited,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: UnlimitedQuantity()}},
					VolumeClasses:  VolumeClassResources{"volume-class": {UnlimitedQuantity()}},
				},
				Secrets: Unlimited,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           Unlimited,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: UnlimitedQuantity()}},
					VolumeClasses:  VolumeClassResources{"volume-class": {UnlimitedQuantity()}},
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
					Apps:           1,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: resource.MustParse("1Mi")}},
					VolumeClasses:  VolumeClassResources{"volume-class": {resource.MustParse("1Mi")}},
				},
				Secrets: 1,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           1,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: resource.MustParse("1Mi")}},
					VolumeClasses:  VolumeClassResources{"volume-class": {resource.MustParse("1Mi")}},
				},
				Secrets: 1,
			},
		},
		{
			name: "does not fit QuotaRequestResources",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           1,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: resource.MustParse("1Mi")}},
					VolumeClasses:  VolumeClassResources{"volume-class": {resource.MustParse("1Mi")}},
				},
				Secrets: 1,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           2,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: resource.MustParse("1Mi")}},
					VolumeClasses:  VolumeClassResources{"volume-class": {resource.MustParse("1Mi")}},
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
					Apps:           1,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: resource.MustParse("1Mi")}},
					VolumeClasses:  VolumeClassResources{"volume-class": {resource.MustParse("1Mi")}},
				},
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           2,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: resource.MustParse("1Mi")}},
					VolumeClasses:  VolumeClassResources{"volume-class": {resource.MustParse("1Mi")}},
				},
			},
			expectedErr: ErrExceededResources,
		},
		{
			name: "fits QuotaRequestResources with specified unlimited values",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           Unlimited,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: UnlimitedQuantity()}},
					VolumeClasses:  VolumeClassResources{"volume-class": {UnlimitedQuantity()}},
				},
				Secrets: Unlimited,
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps:           2,
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: resource.MustParse("2Mi")}},
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
					ComputeClasses: ComputeClassResources{"compute-class": {Memory: UnlimitedQuantity()}},
					VolumeClasses:  VolumeClassResources{"volume-class": {UnlimitedQuantity()}},
				},
			},
			incoming: QuotaRequestResources{
				BaseResources: BaseResources{
					ComputeClasses: ComputeClassResources{"compute-class": {CPU: resource.MustParse("100m")}},
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
					Apps: 1,
					ComputeClasses: ComputeClassResources{"compute-class": {
						Memory: resource.MustParse("1Mi"),
						CPU:    resource.MustParse("1Mi"),
					}},
					VolumeClasses: VolumeClassResources{"volume-class": {resource.MustParse("1Mi")}},
				},
				Secrets: 1,
			},
			expected: "Secrets: 1, Apps: 1, ComputeClasses: \"compute-class\": { Memory: 1Mi, CPU: 1Mi }, VolumeClasses: \"volume-class\": { VolumeStorage: 1Mi }",
		},
		{
			name: "populated BaseResources with unlimited values",
			current: QuotaRequestResources{
				BaseResources: BaseResources{
					Apps: Unlimited,
					ComputeClasses: ComputeClassResources{"compute-class": {
						Memory: UnlimitedQuantity(),
						CPU:    UnlimitedQuantity(),
					}},
					VolumeClasses: VolumeClassResources{"volume-class": {UnlimitedQuantity()}},
				},
				Secrets: Unlimited,
			},
			expected: "Secrets: unlimited, Apps: unlimited, ComputeClasses: \"compute-class\": { Memory: unlimited, CPU: unlimited }, VolumeClasses: \"volume-class\": { VolumeStorage: unlimited }",
		},
	}

	// Run the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.current.ToString())
		})
	}
}
