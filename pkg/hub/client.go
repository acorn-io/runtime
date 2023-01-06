package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

type accountList struct {
	Items []account `json:"items,omitempty"`
}

type account struct {
	Metadata metav1.ObjectMeta `json:"metadata,omitempty"`
	Status   accountStatus     `json:"status,omitempty"`
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

	return json.NewDecoder(resp.Body).Decode(into)
}
