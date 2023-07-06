package cosign

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/sirupsen/logrus"
)

type GitHubPublicKey struct {
	ID  int    `json:"id,omitempty"`
	Key string `json:"key,omitempty"`
}

func getGitHubPublicKeys(username string) ([]GitHubPublicKey, error) {
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
		return nil, fmt.Errorf("no keys found for user %s", username)
	}

	return keys, nil
}
