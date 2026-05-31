package subscription

import (
	"context"
	"encoding/base64"
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

func TestParseRelayProfilesRecognizesAirportSubscription(t *testing.T) {
	raw := "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#tokyo\r\n" +
		"vmess://eyJwcyI6InRva3lvIiwiYWRkIjoiZXhhbXBsZS5jb20iLCJwb3J0IjoiNDQzIiwiaWQiOiIxMjMifQ==\r\n"
	data := []byte(base64.StdEncoding.EncodeToString([]byte(raw)))

	_, err := ParseRelayProfiles(data)
	if err == nil {
		t.Fatal("ParseRelayProfiles() error = nil, want unsupported airport subscription error")
	}
	message := err.Error()
	for _, want := range []string{"真实机场订阅", "ss: 1", "vmess: 1", "mingsui import"} {
		if !strings.Contains(message, want) {
			t.Fatalf("error = %q, want %q", message, want)
		}
	}
}

func TestParseProxyProfilesBase64AirportSubscription(t *testing.T) {
	raw := "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#tokyo\r\n" +
		"vmess://eyJwcyI6InRva3lvIiwiYWRkIjoiZXhhbXBsZS5jb20iLCJwb3J0IjoiNDQzIiwiaWQiOiIxMjMifQ==\r\n"
	data := []byte(base64.StdEncoding.EncodeToString([]byte(raw)))

	profiles, err := ParseProxyProfiles(data)
	if err != nil {
		t.Fatalf("ParseProxyProfiles() error = %v", err)
	}
	if len(profiles) != 2 {
		t.Fatalf("profiles length = %d, want 2", len(profiles))
	}
	if profiles[0].Name != "tokyo" || profiles[0].Protocol != "ss" {
		t.Fatalf("profiles[0] = %+v, want ss tokyo", profiles[0])
	}
	if profiles[1].Name != "tokyo-2" || profiles[1].Protocol != "vmess" {
		t.Fatalf("profiles[1] = %+v, want vmess tokyo-2", profiles[1])
	}
}

func TestParseProxyProfilesSkipsAirportInfoNodes(t *testing.T) {
	raw := "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#%E5%89%A9%E4%BD%99%E6%B5%81%E9%87%8F%EF%BC%9A100GB\r\n" +
		"ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#%5B1x%5D%20%E6%97%A5%E6%9C%AC%201\r\n"

	profiles, err := ParseProxyProfiles([]byte(base64.StdEncoding.EncodeToString([]byte(raw))))
	if err != nil {
		t.Fatalf("ParseProxyProfiles() error = %v", err)
	}
	if len(profiles) != 1 || profiles[0].Name != "[1x] 日本 1" {
		t.Fatalf("profiles = %+v, want only real node", profiles)
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
