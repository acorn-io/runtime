package externalid

import (
	"strings"

	"github.com/acorn-io/baaah/pkg/name"
	"github.com/acorn-io/runtime/pkg/digest"
)

func normalizeString(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, ".", "-")
	return strings.ReplaceAll(s, "/", "-")
}

func ExternalID(accountID, projectName, acornName string) string {
	return name.SafeConcatName(
		"acorn",
		normalizeString(accountID),
		normalizeString(projectName),
		normalizeString(acornName),
		digest.SHA256(accountID, projectName, acornName))
}
