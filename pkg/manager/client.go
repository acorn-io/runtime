package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"

	"github.com/acorn-io/runtime/pkg/version"
	"github.com/sirupsen/logrus"
)

var (
	ErrTokenNotFound = fmt.Errorf("token not found")
	ErrForbidden     = fmt.Errorf("forbidden")

	userAgent = fmt.Sprintf("acorn/%s (%s; %s)", version.Get().String(), runtime.GOOS, runtime.GOARCH)
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
	req, err := newRequest(url, http.MethodDelete, token)
	if err != nil {
		return
	}

	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return
	}
	_ = resp.Body.Close()
}

func httpGet(ctx context.Context, url, token string, into interface{}) error {
	logrus.Debugf("Looking up %s", url)
	req, err := newRequest(url, http.MethodGet, token)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		break
	case http.StatusNotFound:
		return fmt.Errorf("%w: %v", ErrTokenNotFound, resp.StatusCode)
	case http.StatusForbidden:
		return ErrForbidden
	default:
		return fmt.Errorf("invalid status code: %v", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("can't read response %w", err)
	}

	logrus.Debugf("Response code: %v. Response body: %s", resp.StatusCode, body)

	return json.Unmarshal(body, into)
}

func newRequest(url, method, token string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	if token != "" {
		req.Header.Add("Authorization", "Bearer "+token)
	}
	req.Header.Add("User-Agent", userAgent)

	return req, nil
}
