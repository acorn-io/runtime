package cosign

import "fmt"

type ErrNoSupportedKeys struct {
	Username string
}

func (e ErrNoSupportedKeys) Error() string {
	return fmt.Sprintf("no supported keys found for GitHub user %s", e.Username)
}

// Verification Errors similar to Cosign's types, but with exported fields

func NewVerificationFailure(err error) *VerificationFailure {
	return &VerificationFailure{Err: err}
}

type VerificationFailure struct {
	Err error
}

func (e *VerificationFailure) Error() string {
	return e.Err.Error()
}

func (e *VerificationFailure) Unwrap() error {
	return e.Err
}

type ErrNoSignaturesFound struct {
	Err error
}

func (e *ErrNoSignaturesFound) Error() string {
	return e.Err.Error()
}

func (e *ErrNoSignaturesFound) Unwrap() error {
	return e.Err
}

type ErrNoMatchingSignatures struct {
	Err error
}

func (e *ErrNoMatchingSignatures) Error() string {
	return e.Err.Error()
}

func (e *ErrNoMatchingSignatures) Unwrap() error {
	return e.Err
}
