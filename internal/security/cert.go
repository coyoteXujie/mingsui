package security

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type CertificateOptions struct {
	Hosts      []string
	ValidFor   time.Duration
	RSAKeyBits int
}

func GenerateSelfSignedCertificate(options CertificateOptions) (certPEM, keyPEM []byte, err error) {
	hosts := cleanHosts(options.Hosts)
	if len(hosts) == 0 {
		return nil, nil, fmt.Errorf("至少需要一个证书主机名或 IP")
	}
	validFor := options.ValidFor
	if validFor <= 0 {
		validFor = 365 * 24 * time.Hour
	}
	keyBits := options.RSAKeyBits
	if keyBits == 0 {
		keyBits = 2048
	}
	if keyBits < 2048 {
		return nil, nil, fmt.Errorf("RSA 密钥长度不能小于 2048")
	}

	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return nil, nil, err
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, keyBits)
	if err != nil {
		return nil, nil, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: hosts[0],
		},
		NotBefore:             time.Now().Add(-5 * time.Minute),
		NotAfter:              time.Now().Add(validFor),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	for _, host := range hosts {
		if ip := net.ParseIP(host); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, host)
		}
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, err
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	return certPEM, keyPEM, nil
}

func WriteCertificateFiles(certPath, keyPath string, certPEM, keyPEM []byte, force bool) error {
	if strings.TrimSpace(certPath) == "" {
		return fmt.Errorf("证书路径不能为空")
	}
	if strings.TrimSpace(keyPath) == "" {
		return fmt.Errorf("私钥路径不能为空")
	}
	if err := writeFileExclusive(certPath, certPEM, 0o644, force); err != nil {
		return err
	}
	if err := writeFileExclusive(keyPath, keyPEM, 0o600, force); err != nil {
		return err
	}
	return nil
}

func cleanHosts(hosts []string) []string {
	seen := make(map[string]struct{}, len(hosts))
	cleaned := make([]string, 0, len(hosts))
	for _, host := range hosts {
		host = strings.TrimSpace(host)
		host = strings.Trim(host, "[]")
		if host == "" {
			continue
		}
		if _, ok := seen[host]; ok {
			continue
		}
		seen[host] = struct{}{}
		cleaned = append(cleaned, host)
	}
	return cleaned
}

func writeFileExclusive(path string, data []byte, perm os.FileMode, force bool) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	flag := os.O_WRONLY | os.O_CREATE
	if force {
		flag |= os.O_TRUNC
	} else {
		flag |= os.O_EXCL
	}
	file, err := os.OpenFile(path, flag, perm)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return err
	}
	return nil
}
