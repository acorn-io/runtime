package secrets

import (
	"crypto/rand"
	"math/big"
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
