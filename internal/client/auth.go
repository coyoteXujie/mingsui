package client

import (
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/coyoteXujie/mingsui/internal/config"
)

func localAuthEnabled(auth config.ClientAuthConfig) bool {
	return auth.Enabled
}

func checkLocalCredentials(auth config.ClientAuthConfig, username, password string) bool {
	if !localAuthEnabled(auth) {
		return true
	}
	userOK := subtle.ConstantTimeCompare([]byte(username), []byte(auth.Username)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(password), []byte(auth.Password)) == 1
	return userOK && passOK
}

func checkHTTPProxyAuth(req *http.Request, auth config.ClientAuthConfig) bool {
	if !localAuthEnabled(auth) {
		return true
	}
	username, password, ok := parseProxyBasicAuth(req.Header.Get("Proxy-Authorization"))
	return ok && checkLocalCredentials(auth, username, password)
}

func parseProxyBasicAuth(header string) (username, password string, ok bool) {
	scheme, value, found := strings.Cut(strings.TrimSpace(header), " ")
	if !found || !strings.EqualFold(scheme, "Basic") {
		return "", "", false
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return "", "", false
	}
	username, password, found = strings.Cut(string(decoded), ":")
	if !found {
		return "", "", false
	}
	return username, password, true
}
