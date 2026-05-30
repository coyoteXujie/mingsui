package main

import (
	"path/filepath"
	"testing"

	"github.com/coyoteXujie/mingsui/internal/config"
)

func TestLocalProxyMayBeExposed(t *testing.T) {
	cfg := config.DefaultClient()
	cfg.LocalAddr = "127.0.0.1:18080"
	cfg.HTTPAddr = "127.0.0.1:18081"
	if localProxyMayBeExposed(cfg) {
		t.Fatal("localProxyMayBeExposed() = true, want false for loopback listeners")
	}

	cfg.LocalAddr = "0.0.0.0:18080"
	if !localProxyMayBeExposed(cfg) {
		t.Fatal("localProxyMayBeExposed() = false, want true for exposed listener")
	}

	cfg.LocalAuth.Enabled = true
	if localProxyMayBeExposed(cfg) {
		t.Fatal("localProxyMayBeExposed() = true, want false when local auth is enabled")
	}
}

func TestListenAddrIsLoopback(t *testing.T) {
	tests := []struct {
		addr string
		want bool
	}{
		{addr: "127.0.0.1:18080", want: true},
		{addr: "localhost:18080", want: true},
		{addr: "[::1]:18080", want: true},
		{addr: "0.0.0.0:18080", want: false},
		{addr: "192.168.1.2:18080", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			if got := listenAddrIsLoopback(tt.addr); got != tt.want {
				t.Fatalf("listenAddrIsLoopback() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckClientProfileRequiresName(t *testing.T) {
	if code := run([]string{"config", "profile", "check"}); code != 2 {
		t.Fatalf("run(config profile check) = %d, want 2", code)
	}
}

func TestCheckClientProfileRejectsMissingProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "client.json")
	if err := config.WriteClient(path, config.DefaultClient(), true); err != nil {
		t.Fatalf("WriteClient() error = %v", err)
	}

	code := run([]string{"config", "profile", "check", "missing", "-path", path})
	if code != 1 {
		t.Fatalf("run(config profile check missing) = %d, want 1", code)
	}
}
