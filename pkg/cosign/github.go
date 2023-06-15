package cosign

import (
	"crypto"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

var (
	supportedKeyTypes = map[string]interface{}{
		ssh.KeyAlgoRSA:      nil,
		ssh.KeyAlgoED25519:  nil,
		ssh.KeyAlgoECDSA256: nil,
		ssh.KeyAlgoECDSA384: nil,
		ssh.KeyAlgoECDSA521: nil,
	}
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
		keyBytes, err := base64.StdEncoding.DecodeString(keyData)
		if err != nil {
			logrus.Warnf("Failed to decode public key data for GitHub user %s - Key ID #%d: %v", username, key.ID, err)
			continue
		}

		parsedKey, err := ssh.ParsePublicKey(keyBytes)
		if err != nil {
			logrus.Warnf("Failed to parse public key for GitHub user %s - Key ID #%d: %v", username, key.ID, err)
			continue
		}

		if _, ok := supportedKeyTypes[parsedKey.Type()]; !ok {
			logrus.Debugf("Unsupported key type '%s' for GitHub user %s - Key ID #%d: %v", parsedKey.Type(), username, key.ID, err)
			continue
		}

		parsedCryptoKey := parsedKey.(ssh.CryptoPublicKey).CryptoPublicKey()

		validKeys = append(validKeys, parsedCryptoKey)
	}

	if len(validKeys) == 0 {
		return nil, ErrNoSupportedKeys{Username: username}
	}

	return validKeys, nil
}
