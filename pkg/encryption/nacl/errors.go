package nacl

import "errors"

type ErrKeyNotFound struct {
	NamespaceKeyNotFound bool
}

type ErrUnableToDecrypt struct{}

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
	return "Unable to decrypt values"
}

func (d *ErrDecryptionKeyNotAvailable) Error() string {
	return "Decryption Key Not Available on this Cluster"
}
