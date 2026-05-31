package desktop

import (
	"context"
	"encoding/base64"
	"io"
	"log"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/coyoteXujie/mingsui/internal/config"
)

func TestNewAppUsesDefaultConfigWhenFileMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "client.json")
	app, err := NewApp(path, testLogger())
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	if app.ConfigPath() != path {
		t.Fatalf("ConfigPath() = %q, want %q", app.ConfigPath(), path)
	}
	if got := app.Config(); !reflect.DeepEqual(got, config.DefaultClient()) {
		t.Fatalf("Config() = %+v, want default %+v", got, config.DefaultClient())
	}
}

func TestAppSaveConfigAndReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "client.json")
	app, err := NewApp(path, testLogger())
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	want := config.DefaultClient()
	want.Token = "secret"
	want.RelayAddr = "relay.example.com:9443"
	if err := app.SaveConfig(want); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	reloaded, err := NewApp(path, testLogger())
	if err != nil {
		t.Fatalf("NewApp(reload) error = %v", err)
	}
	if got := reloaded.Config(); !reflect.DeepEqual(got, want) {
		t.Fatalf("reloaded Config() = %+v, want %+v", got, want)
	}
}

func TestAppConfigReturnsCopy(t *testing.T) {
	app, err := NewApp(filepath.Join(t.TempDir(), "client.json"), testLogger())
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	cfg := config.DefaultClient()
	cfg.Profiles = []config.RelayProfile{
		{Name: "tokyo", RelayAddr: "tokyo.example.com:9443", Token: "secret"},
	}
	if err := app.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	got := app.Config()
	got.Profiles[0].Token = "changed"
	if app.Config().Profiles[0].Token != "secret" {
		t.Fatalf("Config() returned shared profile slice: %+v", app.Config().Profiles[0])
	}
}

func TestAppManageRelayProfiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "client.json")
	app, err := NewApp(path, testLogger())
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	profile := config.RelayProfile{
		Name:      "tokyo",
		RelayAddr: "tokyo.example.com:9443",
		Token:     "secret",
	}
	if err := app.UpsertRelayProfile(profile, false); err != nil {
		t.Fatalf("UpsertRelayProfile() error = %v", err)
	}
	if err := app.SelectRelayProfile("tokyo"); err != nil {
		t.Fatalf("SelectRelayProfile() error = %v", err)
	}
	if err := app.RenameRelayProfile("tokyo", "jp-tokyo"); err != nil {
		t.Fatalf("RenameRelayProfile() error = %v", err)
	}

	cfg := app.Config()
	if cfg.ActiveProfile != "jp-tokyo" {
		t.Fatalf("ActiveProfile = %q, want jp-tokyo", cfg.ActiveProfile)
	}
	if len(cfg.Profiles) != 1 || cfg.Profiles[0].Name != "jp-tokyo" {
		t.Fatalf("Profiles = %+v, want renamed profile", cfg.Profiles)
	}

	reloaded, err := NewApp(path, testLogger())
	if err != nil {
		t.Fatalf("NewApp(reload) error = %v", err)
	}
	if reloaded.Config().ActiveProfile != "jp-tokyo" {
		t.Fatalf("reloaded ActiveProfile = %q, want jp-tokyo", reloaded.Config().ActiveProfile)
	}

	if err := app.RemoveRelayProfile("jp-tokyo"); err != nil {
		t.Fatalf("RemoveRelayProfile() error = %v", err)
	}
	if got := app.Config(); got.ActiveProfile != "" || len(got.Profiles) != 0 {
		t.Fatalf("Config() = %+v, want no active profile and no profiles", got)
	}
}

func TestAppCheckRelayProfileRejectsMissingProfile(t *testing.T) {
	app, err := NewApp(filepath.Join(t.TempDir(), "client.json"), testLogger())
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	if _, err := app.CheckRelayProfileStatus(context.Background(), "missing"); err == nil {
		t.Fatal("CheckRelayProfileStatus() error = nil, want missing profile error")
	}
}

func TestAppStatusUsesFirstProfileWhenActiveProfileEmpty(t *testing.T) {
	app, err := NewApp(filepath.Join(t.TempDir(), "client.json"), testLogger())
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	cfg := config.DefaultClient()
	cfg.Profiles = []config.RelayProfile{
		{Name: "tokyo", RelayAddr: "tokyo.example.com:9443", Token: "secret"},
	}
	if err := app.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	status := app.Status()
	if status.RelayAddr != "tokyo.example.com:9443" {
		t.Fatalf("Status().RelayAddr = %q, want first profile relay", status.RelayAddr)
	}
}

func TestAppImportRelayProfiles(t *testing.T) {
	app, err := NewApp(filepath.Join(t.TempDir(), "client.json"), testLogger())
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	count, err := app.ImportRelayProfiles([]byte(`[{"name":"tokyo","relay_addr":"tokyo.example.com:9443","token":"secret"}]`), false, "")
	if err != nil {
		t.Fatalf("ImportRelayProfiles() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	cfg := app.Config()
	if cfg.ActiveProfile != "tokyo" || len(cfg.Profiles) != 1 {
		t.Fatalf("Config() = %+v, want imported active profile", cfg)
	}
}

func TestAppImportProxyProfiles(t *testing.T) {
	app, err := NewApp(filepath.Join(t.TempDir(), "client.json"), testLogger())
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	raw := "tuic://00000000-0000-0000-0000-000000000000:pass@example.com:443#future\r\n" +
		"ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#tokyo\r\n"

	count, err := app.ImportRelayProfiles([]byte(base64.StdEncoding.EncodeToString([]byte(raw))), false, "")
	if err != nil {
		t.Fatalf("ImportRelayProfiles(proxy) error = %v", err)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
	cfg := app.Config()
	if cfg.ActiveProxyProfile != "tokyo" || len(cfg.ProxyProfiles) != 2 {
		t.Fatalf("Config() = %+v, want imported active exportable proxy profile", cfg)
	}
	t.Setenv("MINGSUI_MIHOMO_PATH", filepath.Join(t.TempDir(), "missing-mihomo"))
	if err := app.Start(context.Background()); err == nil {
		t.Fatal("Start() error = nil, want missing Mihomo error for proxy profile")
	}
}

func TestAppImportProxyProfilesWithoutExportableDoesNotSelect(t *testing.T) {
	app, err := NewApp(filepath.Join(t.TempDir(), "client.json"), testLogger())
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	raw := "tuic://00000000-0000-0000-0000-000000000000:pass@example.com:443#future\r\n"

	count, err := app.ImportRelayProfiles([]byte(base64.StdEncoding.EncodeToString([]byte(raw))), false, "")
	if err != nil {
		t.Fatalf("ImportRelayProfiles(proxy) error = %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	cfg := app.Config()
	if cfg.ActiveProxyProfile != "" || len(cfg.ProxyProfiles) != 1 {
		t.Fatalf("Config() = %+v, want unsupported proxy imported without active selection", cfg)
	}
	if profile, ok := activeProxyProfile(cfg); ok {
		t.Fatalf("activeProxyProfile() = %+v, true, want false", profile)
	}
	if err := app.Start(context.Background()); err == nil {
		t.Fatal("Start() error = nil, want unsupported proxy error")
	}
	if _, err := app.CheckRelayStatus(context.Background()); err == nil {
		t.Fatal("CheckRelayStatus() error = nil, want unsupported proxy error")
	}
}

func TestSaveImportedSubscription(t *testing.T) {
	cfg := config.DefaultClient()
	if err := saveImportedSubscription(&cfg, "https://example.com/sub"); err != nil {
		t.Fatalf("saveImportedSubscription() error = %v", err)
	}
	if len(cfg.Subscriptions) != 1 || cfg.Subscriptions[0].Name != "airport" || cfg.Subscriptions[0].URL != "https://example.com/sub" {
		t.Fatalf("Subscriptions = %+v, want saved airport subscription", cfg.Subscriptions)
	}

	cfg.Subscriptions[0].Name = "team"
	if err := saveImportedSubscription(&cfg, "https://example.com/sub"); err != nil {
		t.Fatalf("saveImportedSubscription(existing) error = %v", err)
	}
	if len(cfg.Subscriptions) != 1 || cfg.Subscriptions[0].Name != "team" {
		t.Fatalf("Subscriptions = %+v, want existing name preserved", cfg.Subscriptions)
	}
}

func TestAppRejectsUnsupportedProxySelection(t *testing.T) {
	app, err := NewApp(filepath.Join(t.TempDir(), "client.json"), testLogger())
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	cfg := config.DefaultClient()
	cfg.ProxyProfiles = []config.ProxyProfile{
		{Name: "future", Protocol: "tuic", URL: "tuic://00000000-0000-0000-0000-000000000000:pass@example.com:443#future"},
	}
	if err := app.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	if err := app.SelectProxyProfile("future"); err == nil {
		t.Fatal("SelectProxyProfile() error = nil, want unsupported proxy error")
	}
}

func TestAppEnableSystemProxyRejectsLocalAuth(t *testing.T) {
	app, err := NewApp(filepath.Join(t.TempDir(), "client.json"), testLogger())
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	cfg := config.DefaultClient()
	cfg.LocalAuth = config.ClientAuthConfig{
		Enabled:  true,
		Username: "ai",
		Password: "secret",
	}
	if err := app.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	if err := app.EnableSystemProxy(context.Background()); err == nil {
		t.Fatal("EnableSystemProxy() error = nil, want local auth rejection")
	}
}

func TestAppManageRelaySubscriptions(t *testing.T) {
	app, err := NewApp(filepath.Join(t.TempDir(), "client.json"), testLogger())
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	sub := config.RelaySubscription{Name: "team", URL: "https://example.com/nodes.json"}
	if err := app.UpsertRelaySubscription(sub, false); err != nil {
		t.Fatalf("UpsertRelaySubscription() error = %v", err)
	}
	if got := app.Config().Subscriptions; len(got) != 1 || got[0].Name != "team" {
		t.Fatalf("Subscriptions = %+v, want team", got)
	}
	if err := app.RemoveRelaySubscription("team"); err != nil {
		t.Fatalf("RemoveRelaySubscription() error = %v", err)
	}
	if got := app.Config().Subscriptions; len(got) != 0 {
		t.Fatalf("Subscriptions = %+v, want empty", got)
	}
}

func TestAppStopIsNoopWhenNotRunning(t *testing.T) {
	app, err := NewApp(filepath.Join(t.TempDir(), "client.json"), testLogger())
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	if err := app.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func testLogger() *log.Logger {
	return log.New(io.Discard, "", 0)
}
