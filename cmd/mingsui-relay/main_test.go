package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coyoteXujie/mingsui/internal/config"
	"github.com/coyoteXujie/mingsui/internal/security"
)

func TestCheckTLSAcceptsValidCertificate(t *testing.T) {
	certPath, keyPath := writeTestCertificate(t)
	cfg := config.RelayTLSConfig{
		Enabled:  true,
		CertFile: certPath,
		KeyFile:  keyPath,
	}

	got := checkTLS(cfg, time.Now())
	if !got.OK {
		t.Fatalf("checkTLS().OK = false, error = %q", got.Error)
	}
	if got.Certificate == nil {
		t.Fatal("checkTLS().Certificate = nil, want certificate summary")
	}
}

func TestCheckTLSRejectsExpiredCertificate(t *testing.T) {
	certPath, keyPath := writeTestCertificate(t)
	cfg := config.RelayTLSConfig{
		Enabled:  true,
		CertFile: certPath,
		KeyFile:  keyPath,
	}

	got := checkTLS(cfg, time.Now().Add(2*time.Hour))
	if got.OK {
		t.Fatal("checkTLS().OK = true, want false")
	}
	if got.Error != "TLS 证书已过期" {
		t.Fatalf("checkTLS().Error = %q, want expired error", got.Error)
	}
}

func TestFormatRemaining(t *testing.T) {
	tests := []struct {
		name string
		in   time.Duration
		want string
	}{
		{name: "less than minute", in: 30 * time.Second, want: "不足 1 分钟"},
		{name: "minutes", in: 12 * time.Minute, want: "12 分钟"},
		{name: "hours", in: 3*time.Hour + 20*time.Minute, want: "3 小时"},
		{name: "days", in: 49 * time.Hour, want: "2 天"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatRemaining(tt.in); got != tt.want {
				t.Fatalf("formatRemaining() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestApplyRelayOverridesMaxConnections(t *testing.T) {
	cfg := config.DefaultRelay()
	applyRelayOverrides(&cfg, "", "", false, -1)
	if cfg.MaxConnections != 0 {
		t.Fatalf("MaxConnections = %d, want unchanged 0", cfg.MaxConnections)
	}

	applyRelayOverrides(&cfg, "", "", false, 32)
	if cfg.MaxConnections != 32 {
		t.Fatalf("MaxConnections = %d, want 32", cfg.MaxConnections)
	}
}

func TestRelayWarnings(t *testing.T) {
	cfg := config.DefaultRelay()
	cfg.ListenAddr = "0.0.0.0:9443"
	cfg.Token = "change-me"
	cfg.AllowPrivateNetworks = true
	cfg.MaxConnections = 0

	warnings := relayWarnings(cfg)
	assertHasWarning(t, warnings, "当前使用默认 token")
	assertHasWarning(t, warnings, "允许访问私有和本地目标网络")
	assertHasWarning(t, warnings, "未设置最大活跃连接数")
	assertHasWarning(t, warnings, "未启用 TLS")
}

func TestRelayWarningsAcceptsLocalLimitedRelay(t *testing.T) {
	cfg := config.DefaultRelay()
	cfg.ListenAddr = "127.0.0.1:9443"
	cfg.Token = "secret"
	cfg.MaxConnections = 32

	if warnings := relayWarnings(cfg); len(warnings) != 0 {
		t.Fatalf("relayWarnings() = %v, want no warnings", warnings)
	}
}

func TestListenAddrIsLoopback(t *testing.T) {
	tests := []struct {
		addr string
		want bool
	}{
		{addr: "127.0.0.1:9443", want: true},
		{addr: "localhost:9443", want: true},
		{addr: "[::1]:9443", want: true},
		{addr: "0.0.0.0:9443", want: false},
		{addr: "10.0.0.2:9443", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			if got := listenAddrIsLoopback(tt.addr); got != tt.want {
				t.Fatalf("listenAddrIsLoopback() = %v, want %v", got, tt.want)
			}
		})
	}
}

func writeTestCertificate(t *testing.T) (string, string) {
	t.Helper()

	certPEM, keyPEM, err := security.GenerateSelfSignedCertificate(security.CertificateOptions{
		Hosts:    []string{"localhost", "127.0.0.1"},
		ValidFor: time.Hour,
	})
	if err != nil {
		t.Fatalf("GenerateSelfSignedCertificate() error = %v", err)
	}

	dir := t.TempDir()
	certPath := filepath.Join(dir, "relay.crt")
	keyPath := filepath.Join(dir, "relay.key")
	if err := security.WriteCertificateFiles(certPath, keyPath, certPEM, keyPEM, false); err != nil {
		t.Fatalf("WriteCertificateFiles() error = %v", err)
	}
	return certPath, keyPath
}

func assertHasWarning(t *testing.T, warnings []string, want string) {
	t.Helper()

	for _, warning := range warnings {
		if strings.Contains(warning, want) {
			return
		}
	}
	t.Fatalf("warnings %v do not contain %q", warnings, want)
}
