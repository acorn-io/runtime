package helper

import (
	"testing"
)

// Subset returns true if the first array is completely
// contained in the second array. There must be at least
// the same number of duplicate values in second as there
// are in first.
func Subset[V, W any, Z comparable](t *testing.T, first []V, second []W, firstLookup func(V) Z, secondLookup func(W) Z) bool {
	t.Helper()

	set := make(map[Z]int)
	for _, value := range second {
		set[secondLookup(value)]++
	}

	for _, value := range first {
		parsedValue := firstLookup(value)
		if set[parsedValue] < 1 {
			return false
		}
		set[parsedValue]--
	}

	return true
}
