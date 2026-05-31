package mihomo

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/coyoteXujie/mingsui/internal/config"
)

func TestGenerateSSConfig(t *testing.T) {
	cfg := config.DefaultClient()
	cfg.LocalAddr = "127.0.0.1:18080"
	cfg.HTTPAddr = "127.0.0.1:18081"
	cfg.ProxyProfiles = []config.ProxyProfile{
		{
			Name:     "日本 1",
			Protocol: "ss",
			URL:      "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#%E6%97%A5%E6%9C%AC%201",
		},
	}
	cfg.ActiveProxyProfile = "日本 1"

	data, err := Generate(cfg, Options{})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	got := string(data)
	for _, want := range []string{
		`port: 18081`,
		`socks-port: 18080`,
		`name: "日本 1"`,
		`type: "ss"`,
		`server: "example.com"`,
		`port: 8388`,
		`cipher: "aes-256-gcm"`,
		`password: "pass"`,
		`MATCH,明隧`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("config =\n%s\nwant contains %q", got, want)
		}
	}
}

func TestGenerateVMessConfig(t *testing.T) {
	raw := `{"ps":"tokyo","add":"vmess.example.com","port":"443","id":"00000000-0000-0000-0000-000000000000","aid":"0","scy":"auto","tls":"tls","net":"ws","host":"edge.example.com","path":"/ws","sni":"sni.example.com"}`
	cfg := config.DefaultClient()
	cfg.ProxyProfiles = []config.ProxyProfile{
		{
			Name:     "tokyo",
			Protocol: "vmess",
			URL:      "vmess://" + base64.StdEncoding.EncodeToString([]byte(raw)),
		},
	}

	data, err := Generate(cfg, Options{})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	got := string(data)
	for _, want := range []string{
		`type: "vmess"`,
		`server: "vmess.example.com"`,
		`uuid: "00000000-0000-0000-0000-000000000000"`,
		`tls: true`,
		`network: "ws"`,
		`servername: "sni.example.com"`,
		`Host: "edge.example.com"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("config =\n%s\nwant contains %q", got, want)
		}
	}
}

func TestGenerateSkipsUnsupportedUnselectedProxy(t *testing.T) {
	cfg := config.DefaultClient()
	cfg.ProxyProfiles = []config.ProxyProfile{
		{
			Name:     "future",
			Protocol: "vless",
			URL:      "vless://00000000-0000-0000-0000-000000000000@example.com:443#future",
		},
		{
			Name:     "tokyo",
			Protocol: "ss",
			URL:      "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#tokyo",
		},
	}
	cfg.ActiveProxyProfile = "tokyo"

	data, err := Generate(cfg, Options{})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	got := string(data)
	if strings.Contains(got, `name: "future"`) {
		t.Fatalf("config =\n%s\nwant unsupported unselected proxy skipped", got)
	}
	if !strings.Contains(got, `name: "tokyo"`) {
		t.Fatalf("config =\n%s\nwant selected supported proxy", got)
	}
}

func TestGenerateRejectsUnsupportedSelectedProxy(t *testing.T) {
	cfg := config.DefaultClient()
	cfg.ProxyProfiles = []config.ProxyProfile{
		{
			Name:     "future",
			Protocol: "vless",
			URL:      "vless://00000000-0000-0000-0000-000000000000@example.com:443#future",
		},
		{
			Name:     "tokyo",
			Protocol: "ss",
			URL:      "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#tokyo",
		},
	}
	cfg.ActiveProxyProfile = "future"

	if _, err := Generate(cfg, Options{}); err == nil {
		t.Fatal("Generate() error = nil, want unsupported selected proxy error")
	}
}

func TestFirstExportableProfileName(t *testing.T) {
	profiles := []config.ProxyProfile{
		{
			Name:     "future",
			Protocol: "vless",
			URL:      "vless://00000000-0000-0000-0000-000000000000@example.com:443#future",
		},
		{
			Name:     "tokyo",
			Protocol: "ss",
			URL:      "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#tokyo",
		},
	}
	got, ok := FirstExportableProfileName(profiles)
	if !ok || got != "tokyo" {
		t.Fatalf("FirstExportableProfileName() = %q, %v, want tokyo true", got, ok)
	}
}

func TestGenerateRejectsMissingProxyProfiles(t *testing.T) {
	if _, err := Generate(config.DefaultClient(), Options{}); err == nil {
		t.Fatal("Generate() error = nil, want missing proxy profiles error")
	}
}
