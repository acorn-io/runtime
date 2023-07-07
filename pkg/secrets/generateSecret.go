package secrets

import (
	"crypto/rand"
	"math/big"
)

// GenerateRandomSecret generates a random secret with the specified length
// using a mix of uppercase letters, lowercase letters, numbers, and special characters.
func GenerateRandomSecret(length int) (string, error) {
	const (
		uppercase = "ABCDEFGHJKLMNPQRSTUVWXYZ"
		lowercase = "abcdefghijkmnopqrstuvwxyz"
		numbers   = "23456789"
		special   = "!@#$%^&*_-=+"
	)

	// Create a pool of characters to choose from
	pool := uppercase + lowercase + numbers + special

	// Generate a random secret by randomly selecting characters from the pool
	secret := make([]byte, length)
	for i := 0; i < length; i++ {
		index, err := rand.Int(rand.Reader, big.NewInt(int64(len(pool))))
		if err != nil {
			return "", err
		}
		secret[i] = pool[index.Int64()]
	}

	return string(secret), nil
}
