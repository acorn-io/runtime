package manager

import (
	"fmt"
	"strings"
)

func isLocal(address string) bool {
	return strings.HasPrefix(address, "localhost") || strings.HasPrefix(address, "127")
}

func toDiscoverURL(address string) string {
	return fmt.Sprintf("%s://%s/apis/manager.acorn.io/v1", scheme(address), address)
}

func toProjectMembershipURL(address string) string {
	return fmt.Sprintf("%s://%s/apis/manager.acorn.io/v1/projectmemberships", scheme(address), address)
}

func toAccountURL(address, account string) string {
	return fmt.Sprintf("%s://%s/apis/manager.acorn.io/v1/accounts/%s", scheme(address), address, account)
}

func scheme(address string) string {
	if isLocal(address) {
		return "http"
	}
	return "https"
}

func toLoginURL(address, password string) string {
	return fmt.Sprintf("%s://%s/auth/cli?p=%s", scheme(address), address, password)
}

func toTokenRequestURL(address, password string) string {
	return fmt.Sprintf("%s://%s/apis/manager.acorn.io/v1/tokenrequests/%s", scheme(address), address, password)
}
