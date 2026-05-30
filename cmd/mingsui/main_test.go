package main

import (
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
