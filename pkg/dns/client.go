package dns

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

// Client handles interactions with the AcornDNS API service and Acorn.
type Client interface {

	// ReserveDomain calls AcornDNS to reserve a new domain. It returns the domain, a token for authentication,
	// and an error
	ReserveDomain(endpoint string) (string, string, error)

	// CreateRecords calls AcornDNS to create dns records based on the supplied RecordRequests for the specified domain
	CreateRecords(endpoint, domain, token string, records []RecordRequest) error

	// Renew calls AcornDNS to renew the domain and the records specified in the renewRequest. The response will contain
	// "out of sync" records, which are records that AcornDNS either doesn't know about or has different values for
	Renew(endpoint, domain, token string, renew RenewRequest) (RenewResponse, error)

	// DeleteRecord calls AcornDNS to delete the record(s) associated with the supplied prefix
	DeleteRecord(endpoint, domain, recordPrefix, token string) error

	// PurgeRecords calls AcornDNS to purge all records for the given domain, but doesn't delete the domain itself
	PurgeRecords(endpoint, domain, token string) error
}

// AuthFailedNoDomainError indicates that a request failed authentication because the domain was not found. If encountered,
// we'll need to reserve a new domain.
type AuthFailedNoDomainError struct{}

// Error implements the Error interface
func (e AuthFailedNoDomainError) Error() string {
	return "the supplied domain failed authentication"
}

// IsDomainAuthError checks if the error is a DomainAuthError
func IsDomainAuthError(err error) bool {
	return errors.Is(err, AuthFailedNoDomainError{})
}

// NewClient creates a new AcornDNS client
func NewClient() Client {
	return &client{
		c: http.DefaultClient,
	}
}

type client struct {
	c *http.Client
}

func (c *client) CreateRecords(endpoint, domain, token string, records []RecordRequest) error {
	url := fmt.Sprintf("%s/domains/%s/records", endpoint, domain)

	for _, recordRequest := range records {
		body, err := jsonBody(recordRequest)
		if err != nil {
			return err
		}

		req, err := c.request(http.MethodPost, url, body, token)
		if err != nil {
			return err
		}

		err = c.do(req, &RecordResponse{}, &authedRateLimit)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *client) Renew(endpoint, domain, token string, renew RenewRequest) (RenewResponse, error) {
	url := fmt.Sprintf("%v/domains/%v/renew", endpoint, domain)
	body, err := jsonBody(renew)
	if err != nil {
		return RenewResponse{}, err
	}

	req, err := c.request(http.MethodPost, url, body, token)
	if err != nil {
		return RenewResponse{}, err
	}

	resp := RenewResponse{}
	err = c.do(req, &resp, &authedRateLimit)
	if err != nil {
		return RenewResponse{}, fmt.Errorf("failed to execute renew request, error: %w", err)
	}
	return resp, nil
}

func (c *client) ReserveDomain(endpoint string) (string, string, error) {
	url := fmt.Sprintf("%s/%s", endpoint, "domains")

	req, err := c.request(http.MethodPost, url, nil, "")
	if err != nil {
		return "", "", err
	}

	resp := &DomainResponse{}
	err = c.do(req, resp, &unauthedRateLimit)
	if err != nil {
		return "", "", fmt.Errorf("failed to reserve domain, error: %w", err)
	}

	domain := resp.Name
	if !strings.HasPrefix(domain, ".") {
		domain = "." + domain
	}
	return domain, resp.Token, err
}

func (c *client) DeleteRecord(endpoint, domain, prefix, token string) error {
	url := fmt.Sprintf("%v/domains/%v/records/%v", endpoint, domain, prefix)

	req, err := c.request(http.MethodDelete, url, nil, token)
	if err != nil {
		return err
	}

	err = c.do(req, nil, &authedRateLimit)
	if err != nil {
		return fmt.Errorf("failed to execute delete request, error: %w", err)
	}
	return nil
}

func (c *client) PurgeRecords(endpoint, domain, token string) error {
	url := fmt.Sprintf("%v/domains/%v/purgerecords", endpoint, domain)

	req, err := c.request(http.MethodPost, url, nil, token)
	if err != nil {
		return err
	}

	err = c.do(req, nil, &authedRateLimit)
	if err != nil {
		return fmt.Errorf("failed to execute delete request, error: %w", err)
	}
	return nil
}

func (c *client) request(method string, url string, body io.Reader, token string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	if token != "" {
		bearer := "Bearer " + token
		req.Header.Add("Authorization", bearer)
	}

	return req, nil
}

func (c *client) do(req *http.Request, responseBody any, rateLimit *rl) error {
	logrus.Debugf("Making DNS request %v %v", req.Method, req.URL)
	if rateLimit != nil {
		if err := checkRateLimit(rateLimit); err != nil {
			return err
		}
	}

	resp, err := c.c.Do(req)
	if err != nil {
		return err
	}

	logrus.Debugf("Resposne code %v for DNS request %v %v", resp.StatusCode, req.Method, req.URL)

	if resp.StatusCode == http.StatusTooManyRequests {
		rlErrMsg, err := setRateLimited(resp, rateLimit)
		if err != nil {
			return fmt.Errorf("encountered rate limit, but encountered problem processing response: %w", err)
		}
		return fmt.Errorf(rlErrMsg)
	}

	// when err is nil, resp contains a non-nil resp.Body which must be closed
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body, error: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		var authError AuthErrorResponse

		err = json.Unmarshal(body, &authError)
		if err != nil {
			return fmt.Errorf("failed to unmarshal error response, error: %w", err)
		}

		if authError.Data.NoDomain {
			return AuthFailedNoDomainError{}
		}

		return fmt.Errorf("authentication failed")
	}

	if code := resp.StatusCode; code < 200 || code > 300 {
		return fmt.Errorf("unexpected response status code: %v", code)
	}

	if responseBody != nil {
		err = json.Unmarshal(body, responseBody)
		if err != nil {
			return fmt.Errorf("failed to unmarshal response body (%v), error: %w", string(body), err)
		}
	}

	return nil
}

func jsonBody(payload any) (io.Reader, error) {
	buf := &bytes.Buffer{}
	err := json.NewEncoder(buf).Encode(payload)
	if err != nil {
		return nil, err
	}
	return buf, nil
}
