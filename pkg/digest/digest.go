package digest

import (
	"crypto/sha256"
	"encoding/hex"
)

func SHA256(parts ...string) string {
	d := sha256.New()
	for _, part := range parts {
		d.Write([]byte(part))
		d.Write([]byte{'\x00'})
	}
	hash := d.Sum(nil)
	return hex.EncodeToString(hash[:])
}
