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

func TestUpsertRelayProfile(t *testing.T) {
	cfg := config.DefaultClient()
	profile := config.RelayProfile{
		Name:      "tokyo",
		RelayAddr: "tokyo.example.com:9443",
		Token:     "secret",
	}

	if err := upsertRelayProfile(&cfg, profile, false); err != nil {
		t.Fatalf("upsertRelayProfile() error = %v", err)
	}
	if len(cfg.Profiles) != 1 {
		t.Fatalf("profiles length = %d, want 1", len(cfg.Profiles))
	}
	if err := upsertRelayProfile(&cfg, profile, false); err == nil {
		t.Fatal("upsertRelayProfile() duplicate error = nil, want error")
	}

	profile.RelayAddr = "tokyo2.example.com:9443"
	if err := upsertRelayProfile(&cfg, profile, true); err != nil {
		t.Fatalf("upsertRelayProfile(replace) error = %v", err)
	}
	if cfg.Profiles[0].RelayAddr != "tokyo2.example.com:9443" {
		t.Fatalf("RelayAddr = %q, want replaced relay", cfg.Profiles[0].RelayAddr)
	}
}

func TestSelectRelayProfile(t *testing.T) {
	cfg := config.DefaultClient()
	cfg.Profiles = []config.RelayProfile{
		{Name: "tokyo", RelayAddr: "tokyo.example.com:9443", Token: "secret"},
	}

	if err := selectRelayProfile(&cfg, "tokyo"); err != nil {
		t.Fatalf("selectRelayProfile() error = %v", err)
	}
	if cfg.ActiveProfile != "tokyo" {
		t.Fatalf("ActiveProfile = %q, want tokyo", cfg.ActiveProfile)
	}
	if err := selectRelayProfile(&cfg, "missing"); err == nil {
		t.Fatal("selectRelayProfile(missing) error = nil, want error")
	}
}

func TestRemoveRelayProfile(t *testing.T) {
	cfg := config.DefaultClient()
	cfg.ActiveProfile = "tokyo"
	cfg.Profiles = []config.RelayProfile{
		{Name: "tokyo", RelayAddr: "tokyo.example.com:9443", Token: "secret"},
		{Name: "osaka", RelayAddr: "osaka.example.com:9443", Token: "secret"},
	}

	if err := removeRelayProfile(&cfg, "tokyo"); err != nil {
		t.Fatalf("removeRelayProfile() error = %v", err)
	}
	if cfg.ActiveProfile != "" {
		t.Fatalf("ActiveProfile = %q, want cleared", cfg.ActiveProfile)
	}
	if len(cfg.Profiles) != 1 || cfg.Profiles[0].Name != "osaka" {
		t.Fatalf("Profiles = %+v, want only osaka", cfg.Profiles)
	}
	if err := removeRelayProfile(&cfg, "missing"); err == nil {
		t.Fatal("removeRelayProfile(missing) error = nil, want error")
	}
}

func TestRenameRelayProfile(t *testing.T) {
	cfg := config.DefaultClient()
	cfg.ActiveProfile = "tokyo"
	cfg.Profiles = []config.RelayProfile{
		{Name: "tokyo", RelayAddr: "tokyo.example.com:9443", Token: "secret"},
		{Name: "osaka", RelayAddr: "osaka.example.com:9443", Token: "secret"},
	}

	if err := renameRelayProfile(&cfg, "tokyo", "jp-tokyo"); err != nil {
		t.Fatalf("renameRelayProfile() error = %v", err)
	}
	if cfg.ActiveProfile != "jp-tokyo" {
		t.Fatalf("ActiveProfile = %q, want jp-tokyo", cfg.ActiveProfile)
	}
	if cfg.Profiles[0].Name != "jp-tokyo" {
		t.Fatalf("profile name = %q, want jp-tokyo", cfg.Profiles[0].Name)
	}
	if err := renameRelayProfile(&cfg, "jp-tokyo", "osaka"); err == nil {
		t.Fatal("renameRelayProfile(duplicate) error = nil, want error")
	}
}
