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
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d (%s) from GitHub API: %s", resp.StatusCode, resp.Status, string(body))
	}

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
