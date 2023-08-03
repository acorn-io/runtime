package cosign

import "fmt"

type ErrNoSupportedKeys struct {
	Username string
}

func (e ErrNoSupportedKeys) Error() string {
	return fmt.Sprintf("no supported keys found for GitHub user %s", e.Username)
}
