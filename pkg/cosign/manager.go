package cosign

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

type AcornPublicKey struct {
	ID          string `json:"id,omitempty"`
	Key         string `json:"key,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

func getAcornPublicKeys(username string) ([]AcornPublicKey, error) {
	logrus.Debugf("Getting public keys for Acorn user %s", username)

	parts := strings.Split(username, "@")
	if len(parts) > 2 {
		return nil, fmt.Errorf("invalid username: %s", username)
	}

	host := "https://acorn.io"

	if len(parts) == 2 {
		username = parts[0]
		host = parts[1]
	}

	url := fmt.Sprintf("%s/v1/manager.acorn.io.publickeys/%s/", host, username)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// TODO: maybe use token if user is logged in, so that we could rate-limit anonymous requests?
	// token := "g4xc25zkb6b2dkld5fjwvqf6p9lns6lp2h9rbctrqtqf7cb6fr75rd"
	// req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 400 {
		return nil, fmt.Errorf("failed to get public keys for user %s: %d (%s)", username, resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data struct {
		Data []AcornPublicKey `json:"data"`
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	keys := data.Data

	if len(keys) == 0 {
		return nil, fmt.Errorf("no keys found for user %s", username)
	}

	logrus.Debugf("Found %d keys for user %s in Acorn Manager (%s)", len(keys), username, url)

	return keys, nil
}
