package client

import (
	"encoding/base64"
	"net/http/httptest"
	"testing"

	"github.com/coyoteXujie/mingsui/internal/config"
)

func TestCheckHTTPProxyAuthDisabled(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com", nil)
	if !checkHTTPProxyAuth(req, config.ClientAuthConfig{}) {
		t.Fatal("checkHTTPProxyAuth() = false, want true when auth disabled")
	}
}

func TestCheckHTTPProxyAuth(t *testing.T) {
	auth := config.ClientAuthConfig{
		Enabled:  true,
		Username: "user",
		Password: "pass",
	}
	req := httptest.NewRequest("GET", "http://example.com", nil)
	req.Header.Set("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("user:pass")))

	if !checkHTTPProxyAuth(req, auth) {
		t.Fatal("checkHTTPProxyAuth() = false, want true")
	}
}

func TestCheckHTTPProxyAuthRejectsBadPassword(t *testing.T) {
	auth := config.ClientAuthConfig{
		Enabled:  true,
		Username: "user",
		Password: "pass",
	}
	req := httptest.NewRequest("GET", "http://example.com", nil)
	req.Header.Set("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("user:bad")))

	if checkHTTPProxyAuth(req, auth) {
		t.Fatal("checkHTTPProxyAuth() = true, want false")
	}
}
