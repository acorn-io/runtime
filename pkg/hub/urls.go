package hub

import (
	"fmt"
	"strings"
)

func isLocal(address string) bool {
	return strings.HasPrefix(address, "localhost") || strings.HasPrefix(address, "127")
}

func toDiscoverURL(address string) string {
	return fmt.Sprintf("%s://%s/apis/hub.acorn.io/v1", scheme(address), address)
}

func toAccountsURL(address string) string {
	return fmt.Sprintf("%s://%s/apis/hub.acorn.io/v1/accounts", scheme(address), address)
}

func toProjectsURL(endpointAddress string) string {
	return fmt.Sprintf("%s/apis/api.acorn.io/v1/projects", endpointAddress)
}

func toAccountURL(address, account string) string {
	return fmt.Sprintf("%s://%s/apis/hub.acorn.io/v1/accounts/%s", scheme(address), address, account)
}

func scheme(address string) string {
	if isLocal(address) {
		return "http"
	}
	return "https"
}

func toLoginURL(address, password string) string {
	return fmt.Sprintf("%s://%s/auth/login?p=%s", scheme(address), address, password)
}

func toTokenRequestURL(address, password string) string {
	return fmt.Sprintf("%s://%s/apis/hub.acorn.io/v1/tokenrequests/%s", scheme(address), address, password)
}
