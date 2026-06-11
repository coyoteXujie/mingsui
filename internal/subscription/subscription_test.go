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

	"github.com/coyoteXujie/mingsui/internal/config"
	"github.com/coyoteXujie/mingsui/internal/mihomo"
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

func TestParseProxyProfilesClashMihomoYAML(t *testing.T) {
	data := []byte(`
mixed-port: 7890
proxies:
  - name: "日本 SS"
    type: ss
    server: ss.example.com
    port: 8388
    cipher: aes-128-gcm
    password: "p#ass"
  - name: "香港 VLESS"
    type: vless
    server: vless.example.com
    port: 443
    uuid: 00000000-0000-0000-0000-000000000000
    tls: true
    servername: edge.example.com
    network: ws
    ws-opts:
      path: /ws
      headers:
        Host: ws.example.com
  - {name: "新加坡 Trojan", type: trojan, server: trojan.example.com, port: 443, password: pass, sni: trojan.example.com}
proxy-groups:
  - name: Auto
    type: select
    proxies:
      - "日本 SS"
`)

	profiles, err := ParseProxyProfiles(data)
	if err != nil {
		t.Fatalf("ParseProxyProfiles() error = %v", err)
	}
	if len(profiles) != 3 {
		t.Fatalf("profiles length = %d, want 3: %+v", len(profiles), profiles)
	}
	wants := []struct {
		name     string
		protocol string
		contains []string
	}{
		{name: "日本 SS", protocol: "ss", contains: []string{"ss://", "#%E6%97%A5%E6%9C%AC+SS"}},
		{name: "香港 VLESS", protocol: "vless", contains: []string{"security=tls", "sni=edge.example.com", "type=ws", "host=ws.example.com", "path=%2Fws"}},
		{name: "新加坡 Trojan", protocol: "trojan", contains: []string{"trojan://pass@trojan.example.com:443", "sni=trojan.example.com"}},
	}
	for i, want := range wants {
		if profiles[i].Name != want.name || profiles[i].Protocol != want.protocol {
			t.Fatalf("profiles[%d] = %+v, want %s/%s", i, profiles[i], want.name, want.protocol)
		}
		for _, part := range want.contains {
			if !strings.Contains(profiles[i].URL, part) {
				t.Fatalf("profiles[%d].URL = %q, want contains %q", i, profiles[i].URL, part)
			}
		}
	}

	cfg := config.DefaultClient()
	cfg.ProxyProfiles = profiles
	if _, err := mihomo.Generate(cfg, mihomo.Options{}); err != nil {
		t.Fatalf("Generate(parsed Clash profiles) error = %v", err)
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
