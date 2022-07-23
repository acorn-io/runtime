package dns

import "fmt"

const (
	RecordTypeA     RecordType = "A"
	RecordTypeAAAA  RecordType = "AAAA"
	RecordTypeCname RecordType = "CNAME"
	RecordTypeTxt   RecordType = "TXT"
)

type RecordType string

func (rt RecordType) IsValid() error {
	switch rt {
	case RecordTypeA, RecordTypeAAAA, RecordTypeCname, RecordTypeTxt:
		return nil
	}

	return fmt.Errorf("invalid record type")
}

type DomainResponse struct {
	Name  string `json:"name,omitempty"`
	Token string `json:"token,omitempty"`
}

type RenewRequest struct {
	Records []RecordRequest `json:"records,omitempty"`
	Version string          `json:"version,omitempty"`
}

type RenewResponse struct {
	Name             string         `json:"name,omitempty"`
	OutOfSyncRecords []FQDNTypePair `json:"outOfSyncRecords,omitempty"`
}

type RecordRequest struct {
	Name   string     `json:"name,omitempty"`
	Type   RecordType `json:"type,omitempty"`
	Values []string   `json:"values,omitempty"`
}

type RecordResponse struct {
	RecordRequest
	FQDN string `json:"fqdn,omitempty"`
}

type FQDNTypePair struct {
	FQDN string `json:"fqdn,omitempty"`
	Type string `json:"type,omitempty"`
}

type AuthErrorResponse struct {
	Status  int           `json:"status,omitempty"`
	Message string        `json:"msg,omitempty"`
	Data    authErrorData `json:"data,omitempty"`
}

type authErrorData struct {
	NoDomain bool `json:"noDomain,omitempty"`
}
