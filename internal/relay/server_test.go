package relay

import (
	"log"
	"net"
	"strings"
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

func TestHandleHealthReturnsMetrics(t *testing.T) {
	server, err := NewServer(config.DefaultRelay(), log.Default())
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	server.metrics.OpenConnection()
	server.metrics.AddUploadBytes(10)
	server.metrics.AddDownloadBytes(20)

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		server.handle(serverConn)
	}()

	req := protocol.ConnectRequest{
		Version: protocol.Version,
		Command: protocol.CommandHealth,
		Token:   "change-me",
	}
	if err := protocol.WriteJSON(clientConn, req); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	var resp protocol.ConnectResponse
	if err := protocol.ReadJSON(clientConn, &resp); err != nil {
		t.Fatalf("ReadJSON() error = %v", err)
	}
	<-done

	if !resp.OK {
		t.Fatalf("OK = false, error = %q", resp.Error)
	}
	if resp.Metrics == nil {
		t.Fatal("Metrics = nil, want metrics")
	}
	if resp.Metrics.ActiveConnections != 1 || resp.Metrics.TotalConnections != 1 {
		t.Fatalf("connection metrics = %+v, want active=1 total=1", *resp.Metrics)
	}
	if resp.Metrics.UploadBytes != 10 || resp.Metrics.DownloadBytes != 20 {
		t.Fatalf("traffic metrics = %+v, want upload=10 download=20", *resp.Metrics)
	}
}

func TestHandleRejectsWhenMaxConnectionsReached(t *testing.T) {
	cfg := config.DefaultRelay()
	cfg.AllowPrivateNetworks = true
	cfg.MaxConnections = 1
	server, err := NewServer(cfg, log.Default())
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	if !server.metrics.ReserveConnection(cfg.MaxConnections) {
		t.Fatal("ReserveConnection() = false, want true")
	}
	defer server.metrics.CloseConnection()

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		server.handle(serverConn)
	}()

	req := protocol.ConnectRequest{
		Version: protocol.Version,
		Command: protocol.CommandConnect,
		Token:   "change-me",
		Network: "tcp",
		Address: "127.0.0.1:1",
	}
	if err := protocol.WriteJSON(clientConn, req); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	var resp protocol.ConnectResponse
	if err := protocol.ReadJSON(clientConn, &resp); err != nil {
		t.Fatalf("ReadJSON() error = %v", err)
	}
	<-done

	if resp.OK {
		t.Fatal("OK = true, want connection limit rejection")
	}
	if !strings.Contains(resp.Error, "connection limit") {
		t.Fatalf("Error = %q, want connection limit", resp.Error)
	}
	got := server.Metrics()
	if got.ActiveConnections != 1 || got.TotalConnections != 0 {
		t.Fatalf("Metrics = %+v, want active=1 total=0", got)
	}
}
