package v1

import (
	"errors"
	"testing"

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
				Counts:     Counts{Apps: 1},
				Quantities: Quantities{VolumeStorage: resource.MustParse("1Mi")},
			},
			expected: Resources{
				Counts:     Counts{Apps: 1},
				Quantities: Quantities{VolumeStorage: resource.MustParse("1Mi")},
			},
		},
		{
			name: "add to existing resources",
			current: Resources{
				Counts:     Counts{Apps: 1},
				Quantities: Quantities{VolumeStorage: resource.MustParse("1Mi")},
			},
			incoming: Resources{
				Counts: Counts{
					Apps:   1,
					Images: 1,
				},
				Quantities: Quantities{
					VolumeStorage: resource.MustParse("1Mi"),
					CPU:           resource.MustParse("20m"),
				},
			},
			expected: Resources{
				Counts: Counts{
					Apps:   2,
					Images: 1,
				},
				Quantities: Quantities{
					VolumeStorage: resource.MustParse("2Mi"),
					CPU:           resource.MustParse("20m"),
				},
			},
		},
		{
			name:    "does not change flags",
			current: Resources{},
			incoming: Resources{
				Flags:      Flags{Unlimited: true},
				Counts:     Counts{Apps: 1},
				Quantities: Quantities{VolumeStorage: resource.MustParse("1Mi")},
			},
			expected: Resources{
				Counts:     Counts{Apps: 1},
				Quantities: Quantities{VolumeStorage: resource.MustParse("1Mi")},
			},
		},
	}

	// Run the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			current := NewResources()
			current.Add(tc.current)

			incoming := tc.incoming
			current.Add(incoming)
			current.Equals(tc.expected)
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
				Counts:     Counts{Apps: 1},
				Quantities: Quantities{VolumeStorage: resource.MustParse("1Mi")},
			},
			expected: Resources{},
		},
		{
			name: "remove from existing resources",
			current: Resources{
				Counts:     Counts{Apps: 1},
				Quantities: Quantities{VolumeStorage: resource.MustParse("1Mi")},
			},
			incoming: Resources{
				Counts:     Counts{Apps: 1},
				Quantities: Quantities{VolumeStorage: resource.MustParse("1Mi")},
			},
			expected: Resources{},
		},
		{
			name: "remove persistent counts with all",
			current: Resources{
				PersistentCounts: PersistentCounts{Secrets: 1},
			},
			incoming: Resources{
				PersistentCounts: PersistentCounts{Secrets: 1},
			},
			all:      true,
			expected: Resources{},
		},
		{
			name: "does not remove persistent counts without all",
			current: Resources{
				PersistentCounts: PersistentCounts{Secrets: 1},
			},
			incoming: Resources{
				PersistentCounts: PersistentCounts{Secrets: 1},
			},
			expected: Resources{
				PersistentCounts: PersistentCounts{Secrets: 1},
			},
		},
	}

	// Run the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			current := NewResources()
			current.Add(tc.current)

			incoming := tc.incoming
			current.Remove(incoming, tc.all)
			current.Equals(tc.expected)
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
				Counts:           Counts{Apps: 1},
				PersistentCounts: PersistentCounts{Secrets: 1},
				Quantities:       Quantities{VolumeStorage: resource.MustParse("1Mi")},
			},
			incoming: Resources{
				Counts:           Counts{Apps: 1},
				PersistentCounts: PersistentCounts{Secrets: 1},
				Quantities:       Quantities{VolumeStorage: resource.MustParse("1Mi")},
			},
			expected: true,
		},
		{
			name: "unequal resources",
			current: Resources{
				Counts:           Counts{Apps: 1},
				PersistentCounts: PersistentCounts{Secrets: 1},
				Quantities:       Quantities{VolumeStorage: resource.MustParse("1Mi")},
			},
			incoming: Resources{
				Counts:           Counts{Apps: 2},
				PersistentCounts: PersistentCounts{Secrets: 2},
				Quantities:       Quantities{VolumeStorage: resource.MustParse("1Mi")},
			},
			expected: false,
		},
	}

	// Run the test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			current := NewResources()
			current.Add(tc.current)

			if current.Equals(tc.incoming) != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, !tc.expected)
			}
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
				Counts:           Counts{Apps: 1},
				PersistentCounts: PersistentCounts{Secrets: 1},
				Quantities:       Quantities{VolumeStorage: resource.MustParse("1Mi")},
			},
			incoming: Resources{
				Counts:           Counts{Apps: 1},
				PersistentCounts: PersistentCounts{Secrets: 1},
				Quantities:       Quantities{VolumeStorage: resource.MustParse("1Mi")},
			},
		},

		{
			name: "does not fit resources",
			current: Resources{
				Counts:           Counts{Apps: 1},
				PersistentCounts: PersistentCounts{Secrets: 1},
				Quantities:       Quantities{VolumeStorage: resource.MustParse("1Mi")},
			},
			incoming: Resources{
				Counts:           Counts{Apps: 2},
				PersistentCounts: PersistentCounts{Secrets: 2},
				Quantities:       Quantities{VolumeStorage: resource.MustParse("1Mi")},
			},
			expectedErr: ErrDoesNotFit,
		},
		{
			name: "fits resources with unlimited flag set",
			current: Resources{
				Flags:            Flags{Unlimited: true},
				Counts:           Counts{Apps: 1},
				PersistentCounts: PersistentCounts{Secrets: 1},
				Quantities:       Quantities{VolumeStorage: resource.MustParse("1Mi")},
			},
			incoming: Resources{
				Counts:           Counts{Apps: 2},
				PersistentCounts: PersistentCounts{Secrets: 2},
				Quantities:       Quantities{VolumeStorage: resource.MustParse("2Mi")},
			},
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
