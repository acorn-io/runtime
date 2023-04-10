package digest

import (
	"crypto/sha256"
	"encoding/hex"
)

func SHA256(parts ...string) string {
	d := sha256.New()
	for i, part := range parts {
		if i > 0 {
			d.Write([]byte{'\x00'})
		}
		d.Write([]byte(part))
	}
	hash := d.Sum(nil)
	return hex.EncodeToString(hash[:])
}
