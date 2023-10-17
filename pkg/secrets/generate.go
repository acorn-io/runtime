package secrets

import (
	"crypto/rand"
	"math/big"
	"sort"
	"unicode"

	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	defaultLength       = 54
	defaultCharacterSet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!#$%^&*_-=+"
)

// GenerateRandomSecret generates a random secret with the specified length and character set.
// If the length is less than 1, a default value of 54 will be used.
// If the character set is empty, a default value of "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!#$%^&*_-=+"
// will be used.
func GenerateRandomSecret(length int, characterSet string) (string, error) {
	if length < 1 {
		length = defaultLength
	}
	if characterSet == "" {
		characterSet = defaultCharacterSet
	} else {
		characterSet = inflateRanges(characterSet)
	}

	// Generate a random secret by randomly selecting characters from the given character set.
	secret := make([]byte, length)
	for i := 0; i < length; i++ {
		index, err := rand.Int(rand.Reader, big.NewInt(int64(len(characterSet))))
		if err != nil {
			return "", err
		}
		secret[i] = characterSet[index.Int64()]
	}

	return string(secret), nil
}

// inflateRanges inflates a character set by expanding any ranges (e.g. `A-z`) into the full set of characters they represent.
func inflateRanges(characterSet string) string {
	var (
		runeSet  = []rune(characterSet)
		inflated = sets.New[rune]()
	)
	for i := 0; i < len(runeSet); i++ {
		cur := runeSet[i]
		// Alphanumeric character detected
		if alphanumeric(cur) && (i+2 < len(runeSet) && runeSet[i+1] == '-' && alphanumeric(runeSet[i+2])) {
			// Range detected, convert to full set of characters
			start, end := cur, runeSet[i+2]
			if start <= end && !mixedRange(start, end) {
				for c := start; c <= end; c++ {
					if alphanumeric(c) {
						inflated.Insert(c)
					}
				}

				// Skip the next two characters since they were part of the range
				i += 2
				continue
			}
		}

		inflated.Insert(cur)
	}

	sorted := inflated.UnsortedList()
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	return string(sorted)
}

// alphanumeric returns true IFF the given rune is alphanumeric; e.g. [A-z0-9] .
func alphanumeric(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}

// mixedRange returns true IFF the given runes don't have the same casing or aren't both numbers.
func mixedRange(a, b rune) bool {
	switch {
	case unicode.IsNumber(a):
		return !unicode.IsNumber(b)
	case unicode.IsUpper(a):
		return !unicode.IsUpper(b)
	case unicode.IsLower(a):
		return !unicode.IsLower(b)
	}

	return true
}
