package event

import (
	"encoding/hex"
	"hash/fnv"
	"strings"
	"time"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
)

// ContentID returns a deterministic ID based on the content of a given event.
// The returned ID is a valid kubernetes resource name (metadata.name).
func ContentID(e *apiv1.Event) (string, error) {
	// TODO: Reduce the field set used to generate when composite events are added.
	// TODO: Find a better way of selecting and encoding field sets. Maybe a multi-layered io.Writer.
	fieldSet := strings.Join([]string{
		e.Type,
		string(e.Severity),
		e.Actor,
		e.Source.String(),
		e.Description,
		e.Observed.UTC().Format(time.RFC3339),
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
