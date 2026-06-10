package desktop

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/coyoteXujie/mingsui/internal/config"
	"github.com/coyoteXujie/mingsui/internal/proxycheck"
)

func TestHTTPHandlerState(t *testing.T) {
	app, err := NewApp(filepath.Join(t.TempDir(), "client.json"), testLogger())
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	handler, err := NewHTTPHandler(app)
	if err != nil {
		t.Fatalf("NewHTTPHandler() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var got stateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got.ConfigPath == "" {
		t.Fatal("ConfigPath is empty")
	}
	if got.Config.Token != config.RedactedValue {
		t.Fatalf("Token = %q, want redacted", got.Config.Token)
	}
}

func TestHTTPHandlerStateProxyCapabilities(t *testing.T) {
	app, err := NewApp(filepath.Join(t.TempDir(), "client.json"), testLogger())
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	cfg := config.DefaultClient()
	cfg.ProxyProfiles = []config.ProxyProfile{
		{Name: "tokyo", Protocol: "ss", URL: "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#tokyo"},
		{Name: "future", Protocol: "tuic", URL: "tuic://00000000-0000-0000-0000-000000000000:pass@example.com:443#future"},
		{Name: "中国大陆", Protocol: "ss", URL: "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#cn"},
	}
	if err := app.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	handler, err := NewHTTPHandler(app)
	if err != nil {
		t.Fatalf("NewHTTPHandler() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got stateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(got.ProxyCapabilities) != 3 {
		t.Fatalf("ProxyCapabilities = %+v, want 3 items", got.ProxyCapabilities)
	}
	if !got.ProxyCapabilities[0].Exportable || got.ProxyCapabilities[1].Exportable {
		t.Fatalf("ProxyCapabilities = %+v, want ss exportable and tuic unsupported", got.ProxyCapabilities)
	}
	if got.ProxyCapabilities[2].AutoSelectable {
		t.Fatalf("ProxyCapabilities = %+v, want mainland node not auto selectable", got.ProxyCapabilities)
	}
}

func TestHTTPHandlerImportProfiles(t *testing.T) {
	app, err := NewApp(filepath.Join(t.TempDir(), "client.json"), testLogger())
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	handler, err := NewHTTPHandler(app)
	if err != nil {
		t.Fatalf("NewHTTPHandler() error = %v", err)
	}

	body := []byte(`{
  "content": "[{\"name\":\"tokyo\",\"relay_addr\":\"tokyo.example.com:9443\",\"token\":\"secret\"}]",
  "replace": false,
  "select": "tokyo"
}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/profiles/import", bytes.NewReader(body))
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	cfg := app.Config()
	if cfg.ActiveProfile != "tokyo" || len(cfg.Profiles) != 1 {
		t.Fatalf("Config() = %+v, want imported profile", cfg)
	}
}

func TestHTTPHandlerImportAndSelectProxyProfile(t *testing.T) {
	app, err := NewApp(filepath.Join(t.TempDir(), "client.json"), testLogger())
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	handler, err := NewHTTPHandler(app)
	if err != nil {
		t.Fatalf("NewHTTPHandler() error = %v", err)
	}
	raw := "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#tokyo\r\n"
	payload, err := json.Marshal(importProfilesRequest{
		Content: base64.StdEncoding.EncodeToString([]byte(raw)),
		Replace: false,
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/profiles/import", bytes.NewReader(payload))
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("import status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if app.Config().ActiveProxyProfile != "tokyo" {
		t.Fatalf("ActiveProxyProfile = %q, want tokyo", app.Config().ActiveProxyProfile)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/proxy/select", bytes.NewReader([]byte(`{"name":"tokyo"}`)))
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("select status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestHTTPHandlerSyncSubscriptionReturnsReport(t *testing.T) {
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := "tuic://00000000-0000-0000-0000-000000000000:pass@example.com:443#future\r\n" +
			"ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#tokyo\r\n"
		_, _ = w.Write([]byte(base64.StdEncoding.EncodeToString([]byte(raw))))
	}))
	defer source.Close()

	app, err := NewApp(filepath.Join(t.TempDir(), "client.json"), testLogger())
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	if err := app.UpsertRelaySubscription(config.RelaySubscription{Name: "airport", URL: source.URL}, false); err != nil {
		t.Fatalf("UpsertRelaySubscription() error = %v", err)
	}
	handler, err := NewHTTPHandler(app)
	if err != nil {
		t.Fatalf("NewHTTPHandler() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/subscription/sync", bytes.NewReader([]byte(`{"name":"airport","replace":true}`)))
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var got messageResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got.SyncReport == nil || got.SyncReport.Imported != 2 || got.SyncReport.ImportedExportableProxyProfiles != 1 {
		t.Fatalf("response = %+v, want sync report with imported proxy counts", got)
	}
	if got.Count != 2 || app.Config().ActiveProxyProfile != "tokyo" {
		t.Fatalf("response/config = %+v / %+v, want count and selected proxy", got, app.Config())
	}
}

func TestHTTPHandlerRejectsUnsupportedProxySelection(t *testing.T) {
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
	handler, err := NewHTTPHandler(app)
	if err != nil {
		t.Fatalf("NewHTTPHandler() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/proxy/select", bytes.NewReader([]byte(`{"name":"future"}`)))
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("select status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestHTTPHandlerCheckProxyProfilesSelectsBest(t *testing.T) {
	app, err := NewApp(filepath.Join(t.TempDir(), "client.json"), testLogger())
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	cfg := config.DefaultClient()
	cfg.ProxyProfiles = []config.ProxyProfile{
		{Name: "tokyo", Protocol: "ss", URL: "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#tokyo"},
		{Name: "osaka", Protocol: "ss", URL: "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#osaka"},
	}
	if err := app.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	oldRunner := proxyCheckRunner
	proxyCheckRunner = func(ctx context.Context, cfg config.ClientConfig, opts proxycheck.Options) (proxycheck.Report, error) {
		return proxycheck.Report{Results: []proxycheck.Result{
			{Name: "tokyo", Protocol: "ss", Tested: true, OK: true, LatencyMS: 120},
			{Name: "osaka", Protocol: "ss", Tested: true, OK: true, LatencyMS: 60},
		}}, nil
	}
	defer func() {
		proxyCheckRunner = oldRunner
	}()
	handler, err := NewHTTPHandler(app)
	if err != nil {
		t.Fatalf("NewHTTPHandler() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/proxy/check", bytes.NewReader([]byte(`{"select_best":true}`)))
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if app.Config().ActiveProxyProfile != "osaka" {
		t.Fatalf("ActiveProxyProfile = %q, want osaka", app.Config().ActiveProxyProfile)
	}
}

func TestHTTPHandlerCheckSingleProxyProfile(t *testing.T) {
	app, err := NewApp(filepath.Join(t.TempDir(), "client.json"), testLogger())
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	cfg := config.DefaultClient()
	cfg.ProxyProfiles = []config.ProxyProfile{
		{Name: "tokyo", Protocol: "ss", URL: "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#tokyo"},
		{Name: "中国大陆", Protocol: "ss", URL: "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#cn"},
	}
	if err := app.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	oldRunner := proxyCheckRunner
	proxyCheckRunner = func(ctx context.Context, cfg config.ClientConfig, opts proxycheck.Options) (proxycheck.Report, error) {
		if len(cfg.ProxyProfiles) != 1 || cfg.ProxyProfiles[0].Name != "中国大陆" {
			t.Fatalf("ProxyProfiles = %+v, want only requested node", cfg.ProxyProfiles)
		}
		if !opts.IncludeNonAutoSelectable {
			t.Fatal("IncludeNonAutoSelectable = false, want true")
		}
		return proxycheck.Report{Results: []proxycheck.Result{
			{Name: "中国大陆", Protocol: "ss", Tested: true, OK: true, LatencyMS: 90},
		}}, nil
	}
	defer func() {
		proxyCheckRunner = oldRunner
	}()
	handler, err := NewHTTPHandler(app)
	if err != nil {
		t.Fatalf("NewHTTPHandler() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/proxy/check", bytes.NewReader([]byte(`{"name":"中国大陆"}`)))
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got messageResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got.ProxyCheck == nil || got.ProxyCheck.Results[0].Name != "中国大陆" {
		t.Fatalf("response = %+v, want single proxy check report", got)
	}
}

func TestHTTPHandlerDeleteProxyProfile(t *testing.T) {
	app, err := NewApp(filepath.Join(t.TempDir(), "client.json"), testLogger())
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	cfg := config.DefaultClient()
	cfg.ProxyProfiles = []config.ProxyProfile{
		{Name: "tokyo", Protocol: "ss", URL: "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#tokyo"},
		{Name: "osaka", Protocol: "ss", URL: "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#osaka"},
	}
	cfg.ActiveProxyProfile = "osaka"
	if err := app.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	handler, err := NewHTTPHandler(app)
	if err != nil {
		t.Fatalf("NewHTTPHandler() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/proxy/delete", bytes.NewReader([]byte(`{"name":"osaka"}`)))
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	got := app.Config()
	if got.ActiveProxyProfile != "" || len(got.ProxyProfiles) != 1 || got.ProxyProfiles[0].Name != "tokyo" {
		t.Fatalf("Config() = %+v, want osaka deleted and active selection cleared", got)
	}
}

func TestHTTPHandlerCheckProxyProfile(t *testing.T) {
	app, err := NewApp(filepath.Join(t.TempDir(), "client.json"), testLogger())
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	cfg := config.DefaultClient()
	cfg.ProxyProfiles = []config.ProxyProfile{
		{Name: "tokyo", Protocol: "ss", URL: "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#tokyo"},
	}
	cfg.ActiveProxyProfile = "tokyo"
	if err := app.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	t.Setenv("MINGSUI_MIHOMO_PATH", fakeDesktopMihomo(t))
	handler, err := NewHTTPHandler(app)
	if err != nil {
		t.Fatalf("NewHTTPHandler() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/check", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got messageResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got.Mode != "proxy" || !got.OK {
		t.Fatalf("response = %+v, want proxy ok", got)
	}
}

func TestHTTPHandlerSaveAndDeleteProfile(t *testing.T) {
	app, err := NewApp(filepath.Join(t.TempDir(), "client.json"), testLogger())
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	cfg := config.DefaultClient()
	cfg.Profiles = []config.RelayProfile{
		{Name: "tokyo", RelayAddr: "tokyo.example.com:9443", Token: "profile-secret"},
	}
	if err := app.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	handler, err := NewHTTPHandler(app)
	if err != nil {
		t.Fatalf("NewHTTPHandler() error = %v", err)
	}

	saveBody := []byte(`{"name":"tokyo","relay_addr":"tokyo2.example.com:9443","token":"******","replace":true,"tls":{"enabled":false}}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/profile", bytes.NewReader(saveBody))
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("save status = %d, body = %s", rec.Code, rec.Body.String())
	}
	got := app.Config().Profiles[0]
	if got.RelayAddr != "tokyo2.example.com:9443" || got.Token != "profile-secret" {
		t.Fatalf("profile = %+v, want updated relay and preserved token", got)
	}

	deleteBody := []byte(`{"name":"tokyo"}`)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/profile/delete", bytes.NewReader(deleteBody))
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(app.Config().Profiles) != 0 {
		t.Fatalf("Profiles = %+v, want empty", app.Config().Profiles)
	}
}

func TestHTTPHandlerSaveConfigPreservesRedactedSecrets(t *testing.T) {
	app, err := NewApp(filepath.Join(t.TempDir(), "client.json"), testLogger())
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	cfg := config.DefaultClient()
	cfg.Token = "secret"
	cfg.Profiles = []config.RelayProfile{
		{Name: "tokyo", RelayAddr: "tokyo.example.com:9443", Token: "profile-secret"},
	}
	cfg.Subscriptions = []config.RelaySubscription{
		{Name: "team", URL: "https://token@example.com/nodes.json"},
	}
	cfg.ProxyProfiles = []config.ProxyProfile{
		{Name: "hk", Protocol: "ss", URL: "ss://secret@example.com:8388#hk"},
	}
	if err := app.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	handler, err := NewHTTPHandler(app)
	if err != nil {
		t.Fatalf("NewHTTPHandler() error = %v", err)
	}

	redacted := cfg.Redacted()
	redacted.LocalAddr = "127.0.0.1:19080"
	payload, err := json.Marshal(redacted)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(payload))
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	got := app.Config()
	if got.Token != "secret" || got.Profiles[0].Token != "profile-secret" || got.Subscriptions[0].URL != "https://token@example.com/nodes.json" || got.ProxyProfiles[0].URL != "ss://secret@example.com:8388#hk" {
		t.Fatalf("Config() secrets = %+v, want preserved", got)
	}
	if got.LocalAddr != "127.0.0.1:19080" {
		t.Fatalf("LocalAddr = %q, want updated", got.LocalAddr)
	}
}

func TestHTTPHandlerRejectsWrongMethod(t *testing.T) {
	app, err := NewApp(filepath.Join(t.TempDir(), "client.json"), testLogger())
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	handler, err := NewHTTPHandler(app)
	if err != nil {
		t.Fatalf("NewHTTPHandler() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/start", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func fakeDesktopMihomo(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "mihomo")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
