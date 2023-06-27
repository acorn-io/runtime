package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/sirupsen/logrus"
)

type tokenRequest struct {
	Spec   tokenRequestSpec   `json:"spec,omitempty"`
	Status tokenRequestStatus `json:"status,omitempty"`
}

type tokenRequestSpec struct {
	AccountName string `json:"accountName,omitempty"`
}

type tokenRequestStatus struct {
	Token   string `json:"token,omitempty"`
	Expired bool   `json:"expired,omitempty"`
}

type membershipList struct {
	Items []membership `json:"items,omitempty"`
}

type membership struct {
	AccountName        string `json:"accountName,omitempty"`
	ProjectName        string `json:"projectName,omitempty"`
	AccountEndpointURL string `json:"accountEndpointURL,omitempty"`
}

type account struct {
	Status accountStatus `json:"status,omitempty"`
}

type accountStatus struct {
	EndpointURL string `json:"endpointURL,omitempty"`
}

func httpDelete(ctx context.Context, url, token string) {
	logrus.Debugf("Delete %s", url)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return
	}
	_ = resp.Body.Close()
}

func httpGet(ctx context.Context, url, token string, into interface{}) error {
	logrus.Debugf("Looking up %s", url)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	if token != "" {
		req.Header.Add("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid status code: %v", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("can't read response %w", err)
	}

	logrus.Debugf("Response code: %v. Response body: %s", resp.StatusCode, body)

	return json.Unmarshal(body, into)
}
