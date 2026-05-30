package relay

import (
	"log"
	"testing"

	"github.com/coyoteXujie/mingsui/internal/config"
	"github.com/coyoteXujie/mingsui/internal/protocol"
)

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

func TestValidateRequestAllowsHealthCommand(t *testing.T) {
	server, err := NewServer(config.DefaultRelay(), log.Default())
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	err = server.validateRequest(protocol.ConnectRequest{
		Version: protocol.Version,
		Command: protocol.CommandHealth,
		Token:   "change-me",
	})
	if err != nil {
		t.Fatalf("validateRequest() error = %v", err)
	}
}

func TestValidateRequestRejectsUnknownCommand(t *testing.T) {
	server, err := NewServer(config.DefaultRelay(), log.Default())
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	err = server.validateRequest(protocol.ConnectRequest{
		Version: protocol.Version,
		Command: "unknown",
		Token:   "change-me",
	})
	if err == nil {
		t.Fatal("validateRequest() error = nil, want unknown command error")
	}
}
