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

func TestGenerateTrojanConfig(t *testing.T) {
	cfg := config.DefaultClient()
	cfg.ProxyProfiles = []config.ProxyProfile{
		{
			Name:     "trojan-hk",
			Protocol: "trojan",
			URL:      "trojan://secret@example.com:443?security=tls&sni=sni.example.com&type=ws&host=edge.example.com&path=%2Fws&allowInsecure=1#trojan-hk",
		},
	}

	data, err := Generate(cfg, Options{})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	got := string(data)
	for _, want := range []string{
		`type: "trojan"`,
		`server: "example.com"`,
		`port: 443`,
		`password: "secret"`,
		`udp: true`,
		`tls: true`,
		`servername: "sni.example.com"`,
		`skip-cert-verify: true`,
		`network: "ws"`,
		`path: "/ws"`,
		`Host: "edge.example.com"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("config =\n%s\nwant contains %q", got, want)
		}
	}
}

func TestGenerateVLESSRealityConfig(t *testing.T) {
	cfg := config.DefaultClient()
	cfg.ProxyProfiles = []config.ProxyProfile{
		{
			Name:     "vless-reality",
			Protocol: "vless",
			URL:      "vless://00000000-0000-0000-0000-000000000000@example.com:443?security=reality&sni=www.example.com&fp=chrome&flow=xtls-rprx-vision&pbk=public-key&sid=short&type=grpc&serviceName=mingsui#vless-reality",
		},
	}

	data, err := Generate(cfg, Options{})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	got := string(data)
	for _, want := range []string{
		`type: "vless"`,
		`uuid: "00000000-0000-0000-0000-000000000000"`,
		`flow: "xtls-rprx-vision"`,
		`tls: true`,
		`servername: "www.example.com"`,
		`client-fingerprint: "chrome"`,
		`public-key: "public-key"`,
		`short-id: "short"`,
		`network: "grpc"`,
		`grpc-service-name: "mingsui"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("config =\n%s\nwant contains %q", got, want)
		}
	}
}

func TestGenerateHysteria2Config(t *testing.T) {
	cfg := config.DefaultClient()
	cfg.ProxyProfiles = []config.ProxyProfile{
		{
			Name:     "hy2-sg",
			Protocol: "hysteria2",
			URL:      "hysteria2://pass@example.com:8443?sni=sni.example.com&insecure=1&obfs=salamander&obfs-password=obfs-pass#hy2-sg",
		},
	}

	data, err := Generate(cfg, Options{})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	got := string(data)
	for _, want := range []string{
		`type: "hysteria2"`,
		`server: "example.com"`,
		`port: 8443`,
		`password: "pass"`,
		`sni: "sni.example.com"`,
		`skip-cert-verify: true`,
		`obfs: "salamander"`,
		`obfs-password: "obfs-pass"`,
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
			Protocol: "tuic",
			URL:      "tuic://00000000-0000-0000-0000-000000000000:pass@example.com:443#future",
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
			Protocol: "tuic",
			URL:      "tuic://00000000-0000-0000-0000-000000000000:pass@example.com:443#future",
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
			Protocol: "tuic",
			URL:      "tuic://00000000-0000-0000-0000-000000000000:pass@example.com:443#future",
		},
		{
			Name:     "vless-reality",
			Protocol: "vless",
			URL:      "vless://00000000-0000-0000-0000-000000000000@example.com:443?security=reality&pbk=public-key&sid=short#vless-reality",
		},
	}
	got, ok := FirstExportableProfileName(profiles)
	if !ok || got != "vless-reality" {
		t.Fatalf("FirstExportableProfileName() = %q, %v, want vless-reality true", got, ok)
	}
}

func TestFirstAutoSelectableProfileNameSkipsMainlandChina(t *testing.T) {
	profiles := []config.ProxyProfile{
		{
			Name:     "中国大陆 01",
			Protocol: "ss",
			URL:      "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#cn",
		},
		{
			Name:     "日本 CN2",
			Protocol: "ss",
			URL:      "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#jp",
		},
	}
	got, ok := FirstAutoSelectableProfileName(profiles)
	if !ok || got != "日本 CN2" {
		t.Fatalf("FirstAutoSelectableProfileName() = %q, %v, want 日本 CN2 true", got, ok)
	}
}

func TestFirstAutoSelectableProfileNameRejectsAllMainlandChina(t *testing.T) {
	profiles := []config.ProxyProfile{
		{
			Name:     "上海",
			Protocol: "ss",
			URL:      "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#shanghai",
		},
	}
	if got, ok := FirstAutoSelectableProfileName(profiles); ok {
		t.Fatalf("FirstAutoSelectableProfileName() = %q, true, want false", got)
	}
}

func TestGenerateAutoSelectsForeignProxy(t *testing.T) {
	cfg := config.DefaultClient()
	cfg.ProxyProfiles = []config.ProxyProfile{
		{
			Name:     "中国大陆",
			Protocol: "ss",
			URL:      "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#cn",
		},
		{
			Name:     "日本",
			Protocol: "ss",
			URL:      "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#jp",
		},
	}

	data, err := Generate(cfg, Options{})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "      - \"日本\"\n      - \"中国大陆\"") {
		t.Fatalf("config =\n%s\nwant 日本 before 中国大陆 in proxy group", got)
	}
}

func TestGenerateRejectsOnlyMainlandChinaWithoutSelection(t *testing.T) {
	cfg := config.DefaultClient()
	cfg.ProxyProfiles = []config.ProxyProfile{
		{
			Name:     "中国大陆",
			Protocol: "ss",
			URL:      "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#cn",
		},
	}

	if _, err := Generate(cfg, Options{}); err == nil {
		t.Fatal("Generate() error = nil, want no auto-selectable foreign node error")
	}
}

func TestLikelyMainlandChinaProfileAllowsHongKong(t *testing.T) {
	profile := config.ProxyProfile{
		Name:     "中国香港",
		Protocol: "ss",
		URL:      "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#hk",
	}
	if LikelyMainlandChinaProfile(profile) {
		t.Fatal("LikelyMainlandChinaProfile(中国香港) = true, want false")
	}
}

func TestLikelyMainlandChinaProfileRejectsReturnToChina(t *testing.T) {
	profile := config.ProxyProfile{
		Name:     "香港回国",
		Protocol: "ss",
		URL:      "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#hk-cn",
	}
	if !LikelyMainlandChinaProfile(profile) {
		t.Fatal("LikelyMainlandChinaProfile(香港回国) = false, want true")
	}
}

func TestGenerateRejectsMissingProxyProfiles(t *testing.T) {
	if _, err := Generate(config.DefaultClient(), Options{}); err == nil {
		t.Fatal("Generate() error = nil, want missing proxy profiles error")
	}
}
