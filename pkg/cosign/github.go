package cosign

import (
	"crypto"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

type ErrNoSupportedKeys struct {
	Username string
}

func (e ErrNoSupportedKeys) Error() string {
	return fmt.Sprintf("no supported keys found for GitHub user %s", e.Username)
}

type GitHubPublicKey struct {
	ID  int    `json:"id,omitempty"`
	Key string `json:"key,omitempty"`
}

func getGitHubPublicKeys(username string) ([]crypto.PublicKey, error) {
	logrus.Debugf("Getting public keys for GitHub user %s", username)
	url := fmt.Sprintf("https://api.github.com/users/%s/keys", username)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var keys []GitHubPublicKey
	err = json.Unmarshal(body, &keys)
	if err != nil {
		return nil, err
	}

	if len(keys) == 0 {
		return nil, err
	}

	var validKeys []crypto.PublicKey

	for _, key := range keys {
		keyData := strings.Fields(key.Key)[1]
		parsedCryptoKey, err := ParsePublicKey(keyData)
		if err != nil {
			logrus.Warnf("Failed to parse public key for GitHub user %s - Key ID #%d: %v", username, key.ID, err)
			continue
		}

		validKeys = append(validKeys, parsedCryptoKey)
	}

	if len(validKeys) == 0 {
		return nil, ErrNoSupportedKeys{Username: username}
	}

	return validKeys, nil
}
