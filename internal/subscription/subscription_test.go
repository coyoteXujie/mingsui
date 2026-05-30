package subscription

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseRelayProfilesDocument(t *testing.T) {
	data := []byte(`{
  "version": 1,
  "profiles": [
    {
      "name": "tokyo",
      "relay_addr": "tokyo.example.com:9443",
      "token": "secret",
      "tls": {"enabled": true, "server_name": "tokyo.example.com"}
    }
  ]
}`)

	profiles, err := ParseRelayProfiles(data)
	if err != nil {
		t.Fatalf("ParseRelayProfiles() error = %v", err)
	}
	if len(profiles) != 1 || profiles[0].Name != "tokyo" {
		t.Fatalf("profiles = %+v, want tokyo", profiles)
	}
	if !profiles[0].TLS.Enabled || profiles[0].TLS.ServerName != "tokyo.example.com" {
		t.Fatalf("TLS = %+v, want enabled profile TLS", profiles[0].TLS)
	}
}

func TestParseRelayProfilesArray(t *testing.T) {
	data := []byte(`[{"name":"tokyo","relay_addr":"tokyo.example.com:9443","token":"secret"}]`)

	profiles, err := ParseRelayProfiles(data)
	if err != nil {
		t.Fatalf("ParseRelayProfiles() error = %v", err)
	}
	if len(profiles) != 1 || profiles[0].RelayAddr != "tokyo.example.com:9443" {
		t.Fatalf("profiles = %+v, want imported profile", profiles)
	}
}

func TestParseRelayProfilesRejectsDuplicateNames(t *testing.T) {
	data := []byte(`[
  {"name":"tokyo","relay_addr":"tokyo.example.com:9443","token":"secret"},
  {"name":"tokyo","relay_addr":"tokyo2.example.com:9443","token":"secret"}
]`)

	if _, err := ParseRelayProfiles(data); err == nil {
		t.Fatal("ParseRelayProfiles() error = nil, want duplicate error")
	}
}

func TestLoaderReadsFileAndStdin(t *testing.T) {
	data := `[{"name":"tokyo","relay_addr":"tokyo.example.com:9443","token":"secret"}]`
	path := filepath.Join(t.TempDir(), "nodes.json")
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	loader := Loader{}
	if profiles, err := loader.LoadRelayProfiles(context.Background(), path, nil); err != nil || len(profiles) != 1 {
		t.Fatalf("LoadRelayProfiles(file) profiles = %+v, error = %v", profiles, err)
	}
	if profiles, err := loader.LoadRelayProfiles(context.Background(), "-", strings.NewReader(data)); err != nil || len(profiles) != 1 {
		t.Fatalf("LoadRelayProfiles(stdin) profiles = %+v, error = %v", profiles, err)
	}
}

func TestLoaderReadsHTTP(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[{"name":"tokyo","relay_addr":"tokyo.example.com:9443","token":"secret"}]`)),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		}),
	}
	loader := Loader{HTTPClient: client}

	profiles, err := loader.LoadRelayProfiles(context.Background(), "https://example.com/nodes.json", nil)
	if err != nil {
		t.Fatalf("LoadRelayProfiles(http) error = %v", err)
	}
	if len(profiles) != 1 || profiles[0].Name != "tokyo" {
		t.Fatalf("profiles = %+v, want tokyo", profiles)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
