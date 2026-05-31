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
	raw := "tuic://00000000-0000-0000-0000-000000000000:pass@example.com:443#future\r\n" +
		"ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#tokyo\r\n"
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
	if cfg.ActiveProxyProfile != "tokyo" || len(cfg.ProxyProfiles) != 2 {
		t.Fatalf("Config() = %+v, want imported active exportable proxy profile", cfg)
	}
	if code := run([]string{"status", "-config", cfgPath}); code != 0 {
		t.Fatalf("run(status proxy) = %d, want 0", code)
	}
	t.Setenv("MINGSUI_MIHOMO_PATH", filepath.Join(dir, "missing-mihomo"))
	if code := run([]string{"connect", "-config", cfgPath}); code != 1 {
		t.Fatalf("run(connect proxy) = %d, want missing mihomo error", code)
	}
}

func TestTopLevelImportDoesNotSelectUnsupportedProxyProfile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "client.json")
	sourcePath := filepath.Join(dir, "airport.txt")
	raw := "tuic://00000000-0000-0000-0000-000000000000:pass@example.com:443#future\r\n"
	if err := os.WriteFile(sourcePath, []byte(base64.StdEncoding.EncodeToString([]byte(raw))), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	code := run([]string{"import", "-path", cfgPath, "-source", sourcePath})
	if code != 0 {
		t.Fatalf("run(import unsupported airport) = %d, want 0", code)
	}

	cfg, err := config.LoadClient(cfgPath)
	if err != nil {
		t.Fatalf("LoadClient() error = %v", err)
	}
	if cfg.ActiveProxyProfile != "" || len(cfg.ProxyProfiles) != 1 {
		t.Fatalf("Config() = %+v, want imported unsupported proxy without active selection", cfg)
	}
	if code := run([]string{"status", "-config", cfgPath}); code != 1 {
		t.Fatalf("run(status unsupported proxy) = %d, want 1", code)
	}
	if code := run([]string{"connect", "-config", cfgPath}); code != 1 {
		t.Fatalf("run(connect unsupported proxy) = %d, want 1", code)
	}
	if code := run([]string{"doctor", "-config", cfgPath, "-json"}); code != 1 {
		t.Fatalf("run(doctor unsupported proxy) = %d, want 1", code)
	}
	if code := run([]string{"exec", "-config", cfgPath, "-connect", "--", "sh", "-c", "true"}); code != 1 {
		t.Fatalf("run(exec -connect unsupported proxy) = %d, want 1", code)
	}
}

func TestSaveImportedSubscription(t *testing.T) {
	cfg := config.DefaultClient()
	if err := saveImportedSubscription(&cfg, "airport", "https://example.com/sub", true); err != nil {
		t.Fatalf("saveImportedSubscription() error = %v", err)
	}
	if len(cfg.Subscriptions) != 1 || cfg.Subscriptions[0].Name != "airport" || cfg.Subscriptions[0].URL != "https://example.com/sub" {
		t.Fatalf("Subscriptions = %+v, want saved airport subscription", cfg.Subscriptions)
	}
	if err := saveImportedSubscription(&cfg, "local", "/tmp/sub.txt", true); err == nil {
		t.Fatal("saveImportedSubscription(local file) error = nil, want error")
	}
}

func TestRunDoctorProxyUsesMihomo(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "client.json")
	cfg := config.DefaultClient()
	cfg.ProxyProfiles = []config.ProxyProfile{
		{Name: "tokyo", Protocol: "ss", URL: "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#tokyo"},
	}
	cfg.ActiveProxyProfile = "tokyo"
	if err := config.WriteClient(cfgPath, cfg, true); err != nil {
		t.Fatalf("WriteClient() error = %v", err)
	}
	t.Setenv("MINGSUI_MIHOMO_PATH", fakeMihomoCommand(t, "exit 0"))

	if code := run([]string{"doctor", "-config", cfgPath, "-skip-local", "-json"}); code != 0 {
		t.Fatalf("run(doctor proxy) = %d, want 0", code)
	}
}

func TestRunDoctorProxyFailsWhenMihomoMissing(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "client.json")
	cfg := config.DefaultClient()
	cfg.ProxyProfiles = []config.ProxyProfile{
		{Name: "tokyo", Protocol: "ss", URL: "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#tokyo"},
	}
	cfg.ActiveProxyProfile = "tokyo"
	if err := config.WriteClient(cfgPath, cfg, true); err != nil {
		t.Fatalf("WriteClient() error = %v", err)
	}
	t.Setenv("MINGSUI_MIHOMO_PATH", filepath.Join(dir, "missing-mihomo"))

	if code := run([]string{"doctor", "-config", cfgPath, "-skip-local", "-json"}); code != 1 {
		t.Fatalf("run(doctor proxy missing mihomo) = %d, want 1", code)
	}
}

func TestConfigProxyListSelectAndRemove(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "client.json")
	cfg := config.DefaultClient()
	cfg.ProxyProfiles = []config.ProxyProfile{
		{Name: "tokyo", Protocol: "ss", URL: "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#tokyo"},
		{Name: "osaka", Protocol: "vmess", URL: "vmess://eyJwcyI6Im9zYWthIiwiYWRkIjoiZXhhbXBsZS5jb20iLCJwb3J0IjoiNDQzIiwiaWQiOiIxMjMifQ=="},
		{Name: "future", Protocol: "tuic", URL: "tuic://00000000-0000-0000-0000-000000000000:pass@example.com:443#future"},
	}
	cfg.ActiveProxyProfile = "tokyo"
	if err := config.WriteClient(cfgPath, cfg, true); err != nil {
		t.Fatalf("WriteClient() error = %v", err)
	}

	if code := run([]string{"config", "proxy", "list", "-path", cfgPath}); code != 0 {
		t.Fatalf("run(config proxy list) = %d, want 0", code)
	}
	if code := run([]string{"config", "proxy", "list", "-path", cfgPath, "-json"}); code != 0 {
		t.Fatalf("run(config proxy list -json) = %d, want 0", code)
	}
	if code := run([]string{"config", "proxy", "select", "future", "-path", cfgPath}); code != 1 {
		t.Fatalf("run(config proxy select unsupported) = %d, want 1", code)
	}
	if code := run([]string{"config", "proxy", "select", "future", "-path", cfgPath, "-force"}); code != 0 {
		t.Fatalf("run(config proxy select unsupported -force) = %d, want 0", code)
	}
	if code := run([]string{"config", "proxy", "select", "osaka", "-path", cfgPath}); code != 0 {
		t.Fatalf("run(config proxy select) = %d, want 0", code)
	}
	got, err := config.LoadClient(cfgPath)
	if err != nil {
		t.Fatalf("LoadClient() error = %v", err)
	}
	if got.ActiveProxyProfile != "osaka" || got.ActiveProfile != "" {
		t.Fatalf("Config() = %+v, want active proxy osaka", got)
	}
	if code := run([]string{"config", "proxy", "remove", "osaka", "-path", cfgPath}); code != 0 {
		t.Fatalf("run(config proxy remove) = %d, want 0", code)
	}
	got, err = config.LoadClient(cfgPath)
	if err != nil {
		t.Fatalf("LoadClient() error = %v", err)
	}
	if got.ActiveProxyProfile != "" || len(got.ProxyProfiles) != 2 {
		t.Fatalf("Config() = %+v, want osaka removed and active proxy cleared", got)
	}
}

func TestResolveClientProxyProfileUsesFirstExportable(t *testing.T) {
	cfg := config.DefaultClient()
	cfg.ProxyProfiles = []config.ProxyProfile{
		{Name: "future", Protocol: "tuic", URL: "tuic://00000000-0000-0000-0000-000000000000:pass@example.com:443#future"},
		{Name: "tokyo", Protocol: "ss", URL: "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#tokyo"},
	}

	profile, ok := resolveClientProxyProfile(cfg, true)
	if !ok || profile.Name != "tokyo" {
		t.Fatalf("resolveClientProxyProfile() = %+v, %v, want tokyo true", profile, ok)
	}
	items := proxyProfileItems(cfg)
	if len(items) != 2 || items[0].Selected || !items[1].Selected {
		t.Fatalf("proxyProfileItems() = %+v, want only exportable tokyo selected", items)
	}
}

func TestResolveClientProxyProfileSkipsAllUnsupported(t *testing.T) {
	cfg := config.DefaultClient()
	cfg.ProxyProfiles = []config.ProxyProfile{
		{Name: "future", Protocol: "tuic", URL: "tuic://00000000-0000-0000-0000-000000000000:pass@example.com:443#future"},
	}

	if profile, ok := resolveClientProxyProfile(cfg, true); ok {
		t.Fatalf("resolveClientProxyProfile() = %+v, true, want false", profile)
	}
	items := proxyProfileItems(cfg)
	if len(items) != 1 || items[0].Selected {
		t.Fatalf("proxyProfileItems() = %+v, want unsupported node unselected", items)
	}
}

func TestProxyEnvUsesHTTPAndSOCKS(t *testing.T) {
	cfg := config.DefaultClient()
	cfg.LocalAddr = "127.0.0.1:18080"
	cfg.HTTPAddr = "127.0.0.1:18081"

	vars := proxyEnv(cfg, "localhost,127.0.0.1")
	assertEnvValue(t, vars, "HTTP_PROXY", "http://127.0.0.1:18081")
	assertEnvValue(t, vars, "HTTPS_PROXY", "http://127.0.0.1:18081")
	assertEnvValue(t, vars, "ALL_PROXY", "socks5h://127.0.0.1:18080")
	assertEnvValue(t, vars, "NO_PROXY", "localhost,127.0.0.1")
	assertEnvValue(t, vars, "MINGSUI_HTTP_PROXY", "http://127.0.0.1:18081")
	assertEnvValue(t, vars, "MINGSUI_SOCKS5_PROXY", "socks5h://127.0.0.1:18080")
}

func TestProxyEnvIncludesLocalAuth(t *testing.T) {
	cfg := config.DefaultClient()
	cfg.LocalAddr = "127.0.0.1:18080"
	cfg.HTTPAddr = "127.0.0.1:18081"
	cfg.LocalAuth = config.ClientAuthConfig{
		Enabled:  true,
		Username: "ai",
		Password: "p@ss word",
	}

	vars := proxyEnv(cfg, "")
	assertEnvValue(t, vars, "HTTP_PROXY", "http://ai:p%40ss%20word@127.0.0.1:18081")
	assertEnvValue(t, vars, "ALL_PROXY", "socks5h://ai:p%40ss%20word@127.0.0.1:18080")
	assertEnvValue(t, vars, "MINGSUI_HTTP_PROXY", "http://ai:p%40ss%20word@127.0.0.1:18081")
}

func TestSystemProxyRejectsLocalAuth(t *testing.T) {
	cfg := config.DefaultClient()
	cfg.LocalAuth = config.ClientAuthConfig{
		Enabled:  true,
		Username: "ai",
		Password: "secret",
	}
	if err := validateSystemProxyConfig(cfg); err == nil {
		t.Fatal("validateSystemProxyConfig() error = nil, want local auth rejection")
	}
}

func TestProxyEnvFallsBackToSOCKSForStandardProxyVars(t *testing.T) {
	cfg := config.DefaultClient()
	cfg.LocalAddr = "127.0.0.1:18080"
	cfg.HTTPAddr = ""

	vars := proxyEnv(cfg, "")
	assertEnvValue(t, vars, "HTTP_PROXY", "socks5h://127.0.0.1:18080")
	assertEnvValue(t, vars, "HTTPS_PROXY", "socks5h://127.0.0.1:18080")
	assertEnvValue(t, vars, "ALL_PROXY", "socks5h://127.0.0.1:18080")
}

func TestRunExecConnectStartsProxyKernel(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "client.json")
	cfg := config.DefaultClient()
	cfg.ProxyProfiles = []config.ProxyProfile{
		{Name: "tokyo", Protocol: "ss", URL: "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#tokyo"},
	}
	cfg.ActiveProxyProfile = "tokyo"
	if err := config.WriteClient(cfgPath, cfg, true); err != nil {
		t.Fatalf("WriteClient() error = %v", err)
	}
	t.Setenv("MINGSUI_MIHOMO_PATH", fakeMihomoCommand(t, "exit 0"))

	code := run([]string{"exec", "-config", cfgPath, "-connect", "-connect-timeout", "0", "--", "sh", "-c", `test "$MINGSUI_HTTP_PROXY" = "http://127.0.0.1:18081"`})
	if code != 0 {
		t.Fatalf("run(exec -connect) = %d, want 0", code)
	}
}

func TestRunExecConnectRequiresProxyProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "client.json")
	if err := config.WriteClient(path, config.DefaultClient(), true); err != nil {
		t.Fatalf("WriteClient() error = %v", err)
	}

	if code := run([]string{"exec", "-config", path, "-connect", "--", "sh", "-c", "true"}); code != 1 {
		t.Fatalf("run(exec -connect without proxy) = %d, want 1", code)
	}
}

func TestRunExecConnectStartsRelayClient(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "client.json")
	cfg := config.DefaultClient()
	cfg.LocalAddr = "127.0.0.1:18180"
	cfg.HTTPAddr = "127.0.0.1:18181"
	cfg.RelayAddr = "127.0.0.1:1"
	cfg.Token = "secret"
	if err := config.WriteClient(cfgPath, cfg, true); err != nil {
		t.Fatalf("WriteClient() error = %v", err)
	}

	command := `test "$MINGSUI_HTTP_PROXY" = "http://` + cfg.HTTPAddr + `"`
	code := run([]string{"exec", "-config", cfgPath, "-connect", "-connect-timeout", "0", "--", "sh", "-c", command})
	if code != 0 {
		t.Fatalf("run(exec -connect relay) = %d, want 0", code)
	}
}

func TestLocalProxyAddrs(t *testing.T) {
	cfg := config.DefaultClient()
	cfg.LocalAddr = "127.0.0.1:18080"
	cfg.HTTPAddr = "127.0.0.1:18081"
	got := localProxyAddrs(cfg)
	if len(got) != 2 || got[0] != cfg.LocalAddr || got[1] != cfg.HTTPAddr {
		t.Fatalf("localProxyAddrs() = %+v, want socks and http addrs", got)
	}
	cfg.HTTPAddr = ""
	got = localProxyAddrs(cfg)
	if len(got) != 1 || got[0] != cfg.LocalAddr {
		t.Fatalf("localProxyAddrs() = %+v, want only socks addr", got)
	}
}

func TestMergeEnvOverridesExistingValues(t *testing.T) {
	vars := []proxyEnvVar{
		{Name: "HTTP_PROXY", Value: "http://127.0.0.1:18081"},
		{Name: "ALL_PROXY", Value: "socks5h://127.0.0.1:18080"},
	}
	merged := mergeEnv([]string{"PATH=/bin", "HTTP_PROXY=http://old.example"}, vars)

	assertEnvString(t, merged, "PATH=/bin")
	assertEnvString(t, merged, "HTTP_PROXY=http://127.0.0.1:18081")
	assertEnvString(t, merged, "ALL_PROXY=socks5h://127.0.0.1:18080")
}

func assertEnvValue(t *testing.T, vars []proxyEnvVar, name, want string) {
	t.Helper()
	for _, item := range vars {
		if item.Name == name {
			if item.Value != want {
				t.Fatalf("%s = %q, want %q", name, item.Value, want)
			}
			return
		}
	}
	t.Fatalf("missing env var %s in %+v", name, vars)
}

func assertEnvString(t *testing.T, env []string, want string) {
	t.Helper()
	for _, item := range env {
		if item == want {
			return
		}
	}
	t.Fatalf("missing env item %q in %+v", want, env)
}

func fakeMihomoCommand(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "mihomo")
	data := "#!/bin/sh\n" + body + "\n"
	if err := os.WriteFile(path, []byte(data), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

func TestKernelExportMihomo(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "client.json")
	outputPath := filepath.Join(dir, "mihomo.yaml")
	cfg := config.DefaultClient()
	cfg.HTTPAddr = "127.0.0.1:18081"
	cfg.ProxyProfiles = []config.ProxyProfile{
		{Name: "tokyo", Protocol: "ss", URL: "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#tokyo"},
	}
	cfg.ActiveProxyProfile = "tokyo"
	if err := config.WriteClient(cfgPath, cfg, true); err != nil {
		t.Fatalf("WriteClient() error = %v", err)
	}

	code := run([]string{"kernel", "export", "-config", cfgPath, "-output", outputPath})
	if code != 0 {
		t.Fatalf("run(kernel export) = %d, want 0", code)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), `type: "ss"`) || !strings.Contains(string(data), `server: "example.com"`) {
		t.Fatalf("mihomo config = %s, want ss example.com", data)
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

func TestSyncSubscriptionDataRelayProfiles(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "client.json")
	cfg := config.DefaultClient()

	code := syncSubscriptionData(cfg, cfgPath, "team", []byte(`[{"name":"tokyo","relay_addr":"tokyo.example.com:9443","token":"secret"}]`), true, "", true)
	if code != 0 {
		t.Fatalf("syncSubscriptionData(relay) = %d, want 0", code)
	}
	got, err := config.LoadClient(cfgPath)
	if err != nil {
		t.Fatalf("LoadClient() error = %v", err)
	}
	if got.ActiveProfile != "tokyo" || len(got.Profiles) != 1 {
		t.Fatalf("Config() = %+v, want synced active relay profile", got)
	}
}

func TestSyncSubscriptionDataProxyProfiles(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "client.json")
	cfg := config.DefaultClient()
	raw := "tuic://00000000-0000-0000-0000-000000000000:pass@example.com:443#future\r\n" +
		"ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#tokyo\r\n"

	code := syncSubscriptionData(cfg, cfgPath, "airport", []byte(base64.StdEncoding.EncodeToString([]byte(raw))), true, "", true)
	if code != 0 {
		t.Fatalf("syncSubscriptionData(proxy) = %d, want 0", code)
	}
	got, err := config.LoadClient(cfgPath)
	if err != nil {
		t.Fatalf("LoadClient() error = %v", err)
	}
	if got.ActiveProxyProfile != "tokyo" || len(got.ProxyProfiles) != 2 {
		t.Fatalf("Config() = %+v, want synced active exportable proxy profile", got)
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
