package main

import (
	"path/filepath"
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
