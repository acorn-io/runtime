package nacl

import (
	"errors"

	"github.com/acorn-io/baaah/pkg/merr"
)

type ErrKeyNotFound struct {
	NamespaceKeyNotFound bool
}

type ErrUnableToDecrypt struct {
	Errs []error
}

type ErrDecryptionKeyNotAvailable struct{}

func NewErrKeyNotFound(ns bool) error {
	return &ErrKeyNotFound{
		NamespaceKeyNotFound: ns,
	}
}

func ErrNamespaceKeyNotFound(err error) bool {
	var keyNotFound *ErrKeyNotFound
	if errors.As(err, &keyNotFound) {
		return keyNotFound.NamespaceKeyNotFound
	}
	return false
}

func (k *ErrKeyNotFound) Error() string {
	if k.NamespaceKeyNotFound {
		return "No keys exist for this namespace"
	}
	return "No encryption keys were found"
}

func (utd *ErrUnableToDecrypt) Error() string {
	if utd != nil && len(utd.Errs) > 0 {
		return "Unable to decrypt values: " + merr.NewErrors(utd.Errs...).Error()
	}
	return "Unable to decrypt values"
}

func (d *ErrDecryptionKeyNotAvailable) Error() string {
	return "Decryption Key Not Available on this Cluster"
}
