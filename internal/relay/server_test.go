package relay

import "testing"

func TestValidateTargetRejectsPrivateAddress(t *testing.T) {
	err := validateTarget("127.0.0.1:80", false)
	if err == nil {
		t.Fatal("validateTarget() error = nil, want private address error")
	}
}

func TestValidateTargetAllowsPrivateAddressWhenConfigured(t *testing.T) {
	err := validateTarget("127.0.0.1:80", true)
	if err != nil {
		t.Fatalf("validateTarget() error = %v", err)
	}
}
