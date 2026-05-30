package main

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
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

func TestImportClientProfilesFromFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "client.json")
	sourcePath := filepath.Join(dir, "nodes.json")
	data := []byte(`[{"name":"tokyo","relay_addr":"tokyo.example.com:9443","token":"secret"}]`)
	if err := os.WriteFile(sourcePath, data, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	code := run([]string{"config", "profile", "import", "-path", cfgPath, "-source", sourcePath, "-select", "tokyo"})
	if code != 0 {
		t.Fatalf("run(config profile import) = %d, want 0", code)
	}

	cfg, err := config.LoadClient(cfgPath)
	if err != nil {
		t.Fatalf("LoadClient() error = %v", err)
	}
	if len(cfg.Profiles) != 1 || cfg.Profiles[0].Name != "tokyo" {
		t.Fatalf("Profiles = %+v, want tokyo", cfg.Profiles)
	}
	if cfg.ActiveProfile != "tokyo" {
		t.Fatalf("ActiveProfile = %q, want tokyo", cfg.ActiveProfile)
	}
}

func TestTopLevelImportSelectsFirstProfile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "client.json")
	sourcePath := filepath.Join(dir, "nodes.json")
	data := []byte(`[{"name":"tokyo","relay_addr":"tokyo.example.com:9443","token":"secret"}]`)
	if err := os.WriteFile(sourcePath, data, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	code := run([]string{"import", "-path", cfgPath, "-source", sourcePath})
	if code != 0 {
		t.Fatalf("run(import) = %d, want 0", code)
	}
	code = run([]string{"import", "-path", cfgPath, "-source", sourcePath})
	if code != 0 {
		t.Fatalf("run(import duplicate) = %d, want 0 because top-level import overwrites by default", code)
	}

	cfg, err := config.LoadClient(cfgPath)
	if err != nil {
		t.Fatalf("LoadClient() error = %v", err)
	}
	if cfg.ActiveProfile != "tokyo" {
		t.Fatalf("ActiveProfile = %q, want tokyo", cfg.ActiveProfile)
	}
}

func TestTopLevelImportStoresProxyProfiles(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "client.json")
	sourcePath := filepath.Join(dir, "airport.txt")
	raw := "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#tokyo\r\n"
	if err := os.WriteFile(sourcePath, []byte(base64.StdEncoding.EncodeToString([]byte(raw))), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	code := run([]string{"import", "-path", cfgPath, "-source", sourcePath})
	if code != 0 {
		t.Fatalf("run(import airport) = %d, want 0", code)
	}

	cfg, err := config.LoadClient(cfgPath)
	if err != nil {
		t.Fatalf("LoadClient() error = %v", err)
	}
	if cfg.ActiveProxyProfile != "tokyo" || len(cfg.ProxyProfiles) != 1 {
		t.Fatalf("Config() = %+v, want imported active proxy profile", cfg)
	}
	if code := run([]string{"status", "-config", cfgPath}); code != 0 {
		t.Fatalf("run(status proxy) = %d, want 0", code)
	}
	if code := run([]string{"connect", "-config", cfgPath}); code != 1 {
		t.Fatalf("run(connect proxy) = %d, want pending proxy engine error", code)
	}
}

func TestResolveClientProfileAutoSelectsFirst(t *testing.T) {
	cfg := config.DefaultClient()
	cfg.Profiles = []config.RelayProfile{
		{Name: "tokyo", RelayAddr: "tokyo.example.com:9443", Token: "secret"},
		{Name: "osaka", RelayAddr: "osaka.example.com:9443", Token: "osaka-secret"},
	}

	got, selected, err := resolveClientProfile(cfg, "", true)
	if err != nil {
		t.Fatalf("resolveClientProfile() error = %v", err)
	}
	if selected != "tokyo" {
		t.Fatalf("selected = %q, want tokyo", selected)
	}
	if got.RelayAddr != "tokyo.example.com:9443" || got.Token != "secret" {
		t.Fatalf("resolved = %+v, want tokyo relay and token", got)
	}
}

func TestRunStatusDoesNotRequireRelayDial(t *testing.T) {
	path := filepath.Join(t.TempDir(), "client.json")
	cfg := config.DefaultClient()
	cfg.RelayAddr = "127.0.0.1:1"
	if err := config.WriteClient(path, cfg, true); err != nil {
		t.Fatalf("WriteClient() error = %v", err)
	}

	if code := run([]string{"status", "-config", path}); code != 0 {
		t.Fatalf("run(status) = %d, want 0", code)
	}
}

func TestConfigSubscriptionAddAndRemove(t *testing.T) {
	path := filepath.Join(t.TempDir(), "client.json")

	code := run([]string{"config", "subscription", "add", "team", "-path", path, "-url", "https://example.com/nodes.json"})
	if code != 0 {
		t.Fatalf("run(config subscription add) = %d, want 0", code)
	}
	cfg, err := config.LoadClient(path)
	if err != nil {
		t.Fatalf("LoadClient() error = %v", err)
	}
	if len(cfg.Subscriptions) != 1 || cfg.Subscriptions[0].Name != "team" {
		t.Fatalf("Subscriptions = %+v, want team", cfg.Subscriptions)
	}

	code = run([]string{"config", "subscription", "remove", "team", "-path", path})
	if code != 0 {
		t.Fatalf("run(config subscription remove) = %d, want 0", code)
	}
	cfg, err = config.LoadClient(path)
	if err != nil {
		t.Fatalf("LoadClient() error = %v", err)
	}
	if len(cfg.Subscriptions) != 0 {
		t.Fatalf("Subscriptions = %+v, want empty", cfg.Subscriptions)
	}
}

func TestExportClientProfiles(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "client.json")
	outputPath := filepath.Join(dir, "nodes.json")
	cfg := config.DefaultClient()
	cfg.Profiles = []config.RelayProfile{
		{Name: "tokyo", RelayAddr: "tokyo.example.com:9443", Token: "secret"},
		{Name: "osaka", RelayAddr: "osaka.example.com:9443", Token: "osaka-secret"},
	}
	if err := config.WriteClient(cfgPath, cfg, true); err != nil {
		t.Fatalf("WriteClient() error = %v", err)
	}

	code := run([]string{"config", "profile", "export", "-path", cfgPath, "-output", outputPath, "tokyo"})
	if code != 0 {
		t.Fatalf("run(config profile export) = %d, want 0", code)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), `"name": "tokyo"`) {
		t.Fatalf("export = %s, want tokyo", data)
	}
	if strings.Contains(string(data), "secret") {
		t.Fatalf("export = %s, want redacted token by default", data)
	}

	code = run([]string{"config", "profile", "export", "-path", cfgPath, "-output", outputPath, "-secrets", "tokyo"})
	if code != 0 {
		t.Fatalf("run(config profile export -secrets) = %d, want 0", code)
	}
	data, err = os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), `"token": "secret"`) {
		t.Fatalf("export = %s, want real token with -secrets", data)
	}
}
