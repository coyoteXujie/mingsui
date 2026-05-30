package config

import (
	"os"
	"path/filepath"
	"reflect"
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
	if !reflect.DeepEqual(got, want) {
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

func TestClientConfigProfiles(t *testing.T) {
	cfg := DefaultClient()
	cfg.Profiles = []RelayProfile{
		{
			Name:      "tokyo",
			RelayAddr: "tokyo.example.com:9443",
			Token:     "tokyo-secret",
			TLS: ClientTLSConfig{
				Enabled:    true,
				ServerName: "tokyo.example.com",
			},
		},
	}
	cfg.ActiveProfile = "tokyo"

	resolved, err := cfg.ResolveProfile("")
	if err != nil {
		t.Fatalf("ResolveProfile() error = %v", err)
	}
	if resolved.RelayAddr != "tokyo.example.com:9443" {
		t.Fatalf("RelayAddr = %q, want tokyo relay", resolved.RelayAddr)
	}
	if resolved.Token != "tokyo-secret" {
		t.Fatalf("Token = %q, want profile token", resolved.Token)
	}
	if !resolved.TLS.Enabled || resolved.TLS.ServerName != "tokyo.example.com" {
		t.Fatalf("TLS = %+v, want profile TLS", resolved.TLS)
	}
}

func TestClientConfigCloneCopiesProfiles(t *testing.T) {
	cfg := DefaultClient()
	cfg.Profiles = []RelayProfile{
		{Name: "tokyo", RelayAddr: "tokyo.example.com:9443", Token: "secret"},
	}
	cfg.Subscriptions = []RelaySubscription{
		{Name: "team", URL: "https://example.com/nodes.json"},
	}

	got := cfg.Clone()
	got.Profiles[0].Token = "changed"
	got.Subscriptions[0].URL = "https://changed.example.com/nodes.json"
	if cfg.Profiles[0].Token != "secret" {
		t.Fatalf("Clone() shared profile slice: %+v", cfg.Profiles[0])
	}
	if cfg.Subscriptions[0].URL != "https://example.com/nodes.json" {
		t.Fatalf("Clone() shared subscription slice: %+v", cfg.Subscriptions[0])
	}
}

func TestClientConfigUpsertRelayProfile(t *testing.T) {
	cfg := DefaultClient()
	profile := RelayProfile{
		Name:      "tokyo",
		RelayAddr: "tokyo.example.com:9443",
		Token:     "secret",
	}

	if err := cfg.UpsertRelayProfile(profile, false); err != nil {
		t.Fatalf("UpsertRelayProfile() error = %v", err)
	}
	if len(cfg.Profiles) != 1 {
		t.Fatalf("profiles length = %d, want 1", len(cfg.Profiles))
	}
	if err := cfg.UpsertRelayProfile(profile, false); err == nil {
		t.Fatal("UpsertRelayProfile() duplicate error = nil, want error")
	}

	profile.RelayAddr = "tokyo2.example.com:9443"
	if err := cfg.UpsertRelayProfile(profile, true); err != nil {
		t.Fatalf("UpsertRelayProfile(replace) error = %v", err)
	}
	if cfg.Profiles[0].RelayAddr != "tokyo2.example.com:9443" {
		t.Fatalf("RelayAddr = %q, want replaced relay", cfg.Profiles[0].RelayAddr)
	}
}

func TestClientConfigImportRelayProfilesIsAtomic(t *testing.T) {
	cfg := DefaultClient()
	cfg.Profiles = []RelayProfile{
		{Name: "tokyo", RelayAddr: "tokyo.example.com:9443", Token: "secret"},
	}

	err := cfg.ImportRelayProfiles([]RelayProfile{
		{Name: "osaka", RelayAddr: "osaka.example.com:9443", Token: "secret"},
		{Name: "tokyo", RelayAddr: "tokyo2.example.com:9443", Token: "secret"},
	}, false)
	if err == nil {
		t.Fatal("ImportRelayProfiles() error = nil, want duplicate error")
	}
	if len(cfg.Profiles) != 1 || cfg.Profiles[0].Name != "tokyo" {
		t.Fatalf("Profiles = %+v, want original unchanged", cfg.Profiles)
	}
}

func TestClientConfigRelaySubscriptions(t *testing.T) {
	cfg := DefaultClient()
	sub := RelaySubscription{Name: "team", URL: "https://example.com/nodes.json"}

	if err := cfg.UpsertRelaySubscription(sub, false); err != nil {
		t.Fatalf("UpsertRelaySubscription() error = %v", err)
	}
	if _, ok := cfg.RelaySubscription("team"); !ok {
		t.Fatal("RelaySubscription(team) ok = false, want true")
	}
	if err := cfg.UpsertRelaySubscription(sub, false); err == nil {
		t.Fatal("UpsertRelaySubscription(duplicate) error = nil, want error")
	}

	sub.URL = "https://example.com/updated.json"
	if err := cfg.UpsertRelaySubscription(sub, true); err != nil {
		t.Fatalf("UpsertRelaySubscription(replace) error = %v", err)
	}
	if cfg.Subscriptions[0].URL != "https://example.com/updated.json" {
		t.Fatalf("URL = %q, want updated", cfg.Subscriptions[0].URL)
	}

	if err := cfg.RemoveRelaySubscription("team"); err != nil {
		t.Fatalf("RemoveRelaySubscription() error = %v", err)
	}
	if len(cfg.Subscriptions) != 0 {
		t.Fatalf("Subscriptions = %+v, want empty", cfg.Subscriptions)
	}
}

func TestClientConfigRejectsInvalidRelaySubscription(t *testing.T) {
	cfg := DefaultClient()
	cfg.Subscriptions = []RelaySubscription{
		{Name: "team", URL: "file:///tmp/nodes.json"},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want invalid subscription URL error")
	}
}

func TestClientConfigRedactsSubscriptions(t *testing.T) {
	cfg := DefaultClient()
	cfg.Subscriptions = []RelaySubscription{
		{Name: "team", URL: "https://token@example.com/nodes.json"},
	}

	got := cfg.Redacted()
	if got.Subscriptions[0].URL != RedactedValue {
		t.Fatalf("subscription URL = %q, want redacted", got.Subscriptions[0].URL)
	}
	if cfg.Subscriptions[0].URL != "https://token@example.com/nodes.json" {
		t.Fatalf("Redacted() mutated original subscription: %+v", cfg.Subscriptions[0])
	}
}

func TestClientConfigSelectRelayProfile(t *testing.T) {
	cfg := DefaultClient()
	cfg.Profiles = []RelayProfile{
		{Name: "tokyo", RelayAddr: "tokyo.example.com:9443", Token: "secret"},
	}

	if err := cfg.SelectRelayProfile("tokyo"); err != nil {
		t.Fatalf("SelectRelayProfile() error = %v", err)
	}
	if cfg.ActiveProfile != "tokyo" {
		t.Fatalf("ActiveProfile = %q, want tokyo", cfg.ActiveProfile)
	}
	if err := cfg.SelectRelayProfile("missing"); err == nil {
		t.Fatal("SelectRelayProfile(missing) error = nil, want error")
	}
}

func TestClientConfigRemoveRelayProfile(t *testing.T) {
	cfg := DefaultClient()
	cfg.ActiveProfile = "tokyo"
	cfg.Profiles = []RelayProfile{
		{Name: "tokyo", RelayAddr: "tokyo.example.com:9443", Token: "secret"},
		{Name: "osaka", RelayAddr: "osaka.example.com:9443", Token: "secret"},
	}

	if err := cfg.RemoveRelayProfile("tokyo"); err != nil {
		t.Fatalf("RemoveRelayProfile() error = %v", err)
	}
	if cfg.ActiveProfile != "" {
		t.Fatalf("ActiveProfile = %q, want cleared", cfg.ActiveProfile)
	}
	if len(cfg.Profiles) != 1 || cfg.Profiles[0].Name != "osaka" {
		t.Fatalf("Profiles = %+v, want only osaka", cfg.Profiles)
	}
	if err := cfg.RemoveRelayProfile("missing"); err == nil {
		t.Fatal("RemoveRelayProfile(missing) error = nil, want error")
	}
}

func TestClientConfigRenameRelayProfile(t *testing.T) {
	cfg := DefaultClient()
	cfg.ActiveProfile = "tokyo"
	cfg.Profiles = []RelayProfile{
		{Name: "tokyo", RelayAddr: "tokyo.example.com:9443", Token: "secret"},
		{Name: "osaka", RelayAddr: "osaka.example.com:9443", Token: "secret"},
	}

	if err := cfg.RenameRelayProfile("tokyo", "jp-tokyo"); err != nil {
		t.Fatalf("RenameRelayProfile() error = %v", err)
	}
	if cfg.ActiveProfile != "jp-tokyo" {
		t.Fatalf("ActiveProfile = %q, want jp-tokyo", cfg.ActiveProfile)
	}
	if cfg.Profiles[0].Name != "jp-tokyo" {
		t.Fatalf("profile name = %q, want jp-tokyo", cfg.Profiles[0].Name)
	}
	if err := cfg.RenameRelayProfile("jp-tokyo", "osaka"); err == nil {
		t.Fatal("RenameRelayProfile(duplicate) error = nil, want error")
	}
}

func TestClientConfigResolveProfileMissing(t *testing.T) {
	cfg := DefaultClient()
	cfg.Profiles = []RelayProfile{
		{Name: "tokyo", RelayAddr: "tokyo.example.com:9443", Token: "secret"},
	}

	if _, err := cfg.ResolveProfile("missing"); err == nil {
		t.Fatal("ResolveProfile() error = nil, want missing profile error")
	}
}

func TestClientConfigRejectsDuplicateProfiles(t *testing.T) {
	cfg := DefaultClient()
	cfg.Profiles = []RelayProfile{
		{Name: "tokyo", RelayAddr: "tokyo.example.com:9443", Token: "secret"},
		{Name: "tokyo", RelayAddr: "tokyo2.example.com:9443", Token: "secret"},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want duplicate profile error")
	}
}

func TestClientConfigRedactsProfiles(t *testing.T) {
	cfg := DefaultClient()
	cfg.Profiles = []RelayProfile{
		{Name: "tokyo", RelayAddr: "tokyo.example.com:9443", Token: "secret"},
	}

	got := cfg.Redacted()
	if got.Profiles[0].Token != RedactedValue {
		t.Fatalf("profile token = %q, want redacted", got.Profiles[0].Token)
	}
	if cfg.Profiles[0].Token != "secret" {
		t.Fatalf("Redacted() mutated original profile: %+v", cfg.Profiles[0])
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

func TestRelayConfigValidatesMaxConnections(t *testing.T) {
	cfg := DefaultRelay()
	cfg.MaxConnections = -1

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want negative max_connections error")
	}

	cfg.MaxConnections = 10
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}
