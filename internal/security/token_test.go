package security

import (
	"regexp"
	"testing"
)

func TestGenerateToken(t *testing.T) {
	token, err := GenerateToken(DefaultTokenBytes)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if len(token) < 32 {
		t.Fatalf("token length = %d, want at least 32", len(token))
	}

	ok, err := regexp.MatchString(`^[A-Za-z0-9_-]+$`, token)
	if err != nil {
		t.Fatalf("MatchString() error = %v", err)
	}
	if !ok {
		t.Fatalf("token contains characters outside URL-safe base64: %q", token)
	}
}

func TestGenerateTokenRejectsWeakLength(t *testing.T) {
	if _, err := GenerateToken(8); err == nil {
		t.Fatal("GenerateToken() error = nil, want weak length error")
	}
}

func TestGenerateTokenRejectsExcessiveLength(t *testing.T) {
	if _, err := GenerateToken(256); err == nil {
		t.Fatal("GenerateToken() error = nil, want excessive length error")
	}
}
