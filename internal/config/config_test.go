package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndLoadClient(t *testing.T) {
	path := filepath.Join(t.TempDir(), "client.json")
	want := DefaultClient()
	want.Token = "secret"

	if err := WriteClient(path, want, false); err != nil {
		t.Fatalf("WriteClient() error = %v", err)
	}

	got, err := LoadClient(path)
	if err != nil {
		t.Fatalf("LoadClient() error = %v", err)
	}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestWriteClientDoesNotOverwriteByDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "client.json")
	cfg := DefaultClient()

	if err := WriteClient(path, cfg, false); err != nil {
		t.Fatalf("WriteClient() error = %v", err)
	}
	if err := WriteClient(path, cfg, false); err == nil {
		t.Fatal("WriteClient() second write error = nil, want exists error")
	}
}

func TestLoadClientRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "client.json")
	data := []byte(`{"local_addr":"127.0.0.1:18080","relay_addr":"127.0.0.1:9443","token":"secret","unknown":true}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := LoadClient(path); err == nil {
		t.Fatal("LoadClient() error = nil, want unknown field error")
	}
}

func TestClientConfigValidatesHTTPAddrWhenPresent(t *testing.T) {
	cfg := DefaultClient()
	cfg.HTTPAddr = "127.0.0.1"

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want invalid http_addr error")
	}

	cfg.HTTPAddr = "127.0.0.1:18081"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestClientConfigValidatesLocalAuth(t *testing.T) {
	cfg := DefaultClient()
	cfg.LocalAuth.Enabled = true

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want missing local auth error")
	}

	cfg.LocalAuth.Username = "user"
	cfg.LocalAuth.Password = "pass"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestClientConfigRedacted(t *testing.T) {
	cfg := DefaultClient()
	cfg.Token = "secret"
	cfg.LocalAuth.Enabled = true
	cfg.LocalAuth.Username = "user"
	cfg.LocalAuth.Password = "pass"

	got := cfg.Redacted()
	if got.Token != RedactedValue {
		t.Fatalf("Token = %q, want redacted", got.Token)
	}
	if got.LocalAuth.Username != "user" {
		t.Fatalf("Username = %q, want unchanged", got.LocalAuth.Username)
	}
	if got.LocalAuth.Password != RedactedValue {
		t.Fatalf("Password = %q, want redacted", got.LocalAuth.Password)
	}
	if cfg.Token != "secret" || cfg.LocalAuth.Password != "pass" {
		t.Fatalf("Redacted() mutated original config: %+v", cfg)
	}
}

func TestRelayConfigRedacted(t *testing.T) {
	cfg := DefaultRelay()
	cfg.Token = "secret"

	got := cfg.Redacted()
	if got.Token != RedactedValue {
		t.Fatalf("Token = %q, want redacted", got.Token)
	}
	if cfg.Token != "secret" {
		t.Fatalf("Redacted() mutated original config: %+v", cfg)
	}
}
