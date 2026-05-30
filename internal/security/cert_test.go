package security

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGenerateSelfSignedCertificate(t *testing.T) {
	certPEM, keyPEM, err := GenerateSelfSignedCertificate(CertificateOptions{
		Hosts:    []string{"example.com", "127.0.0.1"},
		ValidFor: time.Hour,
	})
	if err != nil {
		t.Fatalf("GenerateSelfSignedCertificate() error = %v", err)
	}
	if len(certPEM) == 0 {
		t.Fatal("certPEM is empty")
	}
	if len(keyPEM) == 0 {
		t.Fatal("keyPEM is empty")
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		t.Fatal("Decode(certPEM) returned nil")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("ParseCertificate() error = %v", err)
	}
	if err := cert.VerifyHostname("example.com"); err != nil {
		t.Fatalf("VerifyHostname(example.com) error = %v", err)
	}
	if err := cert.VerifyHostname("127.0.0.1"); err != nil {
		t.Fatalf("VerifyHostname(127.0.0.1) error = %v", err)
	}
}

func TestGenerateSelfSignedCertificateRequiresHost(t *testing.T) {
	if _, _, err := GenerateSelfSignedCertificate(CertificateOptions{}); err == nil {
		t.Fatal("GenerateSelfSignedCertificate() error = nil, want host error")
	}
}

func TestWriteCertificateFilesDoesNotOverwriteByDefault(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "relay.crt")
	keyPath := filepath.Join(dir, "relay.key")

	if err := WriteCertificateFiles(certPath, keyPath, []byte("cert"), []byte("key"), false); err != nil {
		t.Fatalf("WriteCertificateFiles() error = %v", err)
	}
	if err := WriteCertificateFiles(certPath, keyPath, []byte("cert2"), []byte("key2"), false); err == nil {
		t.Fatal("WriteCertificateFiles() second write error = nil, want exists error")
	}

	keyInfo, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("Stat(keyPath) error = %v", err)
	}
	if keyInfo.Mode().Perm() != 0o600 {
		t.Fatalf("key mode = %o, want 0600", keyInfo.Mode().Perm())
	}
}
