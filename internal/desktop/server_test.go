package desktop

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/coyoteXujie/mingsui/internal/config"
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
