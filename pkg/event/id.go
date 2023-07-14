package event

import (
	"encoding/hex"
	"hash/fnv"
	"strconv"
	"strings"

	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
)

// ContentID returns a deterministic ID based on the content of a given event.
// The returned ID is a valid kubernetes resource name (metadata.name).
func ContentID(e *apiv1.Event) (string, error) {
	fieldSet := strings.Join([]string{
		e.Type,
		string(e.Severity),
		e.Source.String(),
		e.Description,
		strconv.FormatInt(e.Observed.UnixMicro(), 10),
	}, ",")

	h := fnv.New128a()
	if _, err := h.Write([]byte(fieldSet)); err != nil {
		return "", err
	}

	digest := h.Sum(nil)
	encoded := hex.EncodeToString(digest)

	// Trim to 63 characters
	// Note: This can't happen with a hex-encoded 128 bit hash, but let's be defensive
	// in case we switch to a hash >= 256 bits.
	if runes := []rune(encoded); len(runes) > 63 {
		encoded = string(runes[:63])
	}

	return encoded, nil
}
