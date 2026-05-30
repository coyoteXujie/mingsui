package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const RedactedValue = "******"

type ClientTLSConfig struct {
	Enabled            bool   `json:"enabled"`
	ServerName         string `json:"server_name"`
	CAFile             string `json:"ca_file"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify"`
}

type ClientAuthConfig struct {
	Enabled  bool   `json:"enabled"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type ClientConfig struct {
	LocalAddr          string           `json:"local_addr"`
	HTTPAddr           string           `json:"http_addr"`
	RelayAddr          string           `json:"relay_addr"`
	Token              string           `json:"token"`
	DialTimeoutSeconds int              `json:"dial_timeout_seconds"`
	LocalAuth          ClientAuthConfig `json:"local_auth"`
	TLS                ClientTLSConfig  `json:"tls"`
}

type RelayTLSConfig struct {
	Enabled  bool   `json:"enabled"`
	CertFile string `json:"cert_file"`
	KeyFile  string `json:"key_file"`
}

type RelayConfig struct {
	ListenAddr           string         `json:"listen_addr"`
	Token                string         `json:"token"`
	DialTimeoutSeconds   int            `json:"dial_timeout_seconds"`
	AllowPrivateNetworks bool           `json:"allow_private_networks"`
	TLS                  RelayTLSConfig `json:"tls"`
}

func DefaultClient() ClientConfig {
	return ClientConfig{
		LocalAddr:          "127.0.0.1:18080",
		RelayAddr:          "127.0.0.1:9443",
		Token:              "change-me",
		DialTimeoutSeconds: 10,
	}
}

func DefaultRelay() RelayConfig {
	return RelayConfig{
		ListenAddr:           "0.0.0.0:9443",
		Token:                "change-me",
		DialTimeoutSeconds:   10,
		AllowPrivateNetworks: false,
	}
}

func DefaultClientPath() string {
	return defaultPath("client.json")
}

func DefaultRelayPath() string {
	return defaultPath("relay.json")
}

func LoadClient(path string) (ClientConfig, error) {
	cfg := DefaultClient()
	if path == "" {
		path = DefaultClientPath()
	}
	if err := loadJSON(path, &cfg); err != nil {
		return ClientConfig{}, err
	}
	if err := cfg.Validate(); err != nil {
		return ClientConfig{}, err
	}
	return cfg, nil
}

func LoadRelay(path string) (RelayConfig, error) {
	cfg := DefaultRelay()
	if path == "" {
		path = DefaultRelayPath()
	}
	if err := loadJSON(path, &cfg); err != nil {
		return RelayConfig{}, err
	}
	if err := cfg.Validate(); err != nil {
		return RelayConfig{}, err
	}
	return cfg, nil
}

func WriteClient(path string, cfg ClientConfig, force bool) error {
	if path == "" {
		path = DefaultClientPath()
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	return writeJSON(path, cfg, force)
}

func WriteRelay(path string, cfg RelayConfig, force bool) error {
	if path == "" {
		path = DefaultRelayPath()
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	return writeJSON(path, cfg, force)
}

func (c ClientConfig) Validate() error {
	if err := validateAddr("local_addr", c.LocalAddr); err != nil {
		return err
	}
	if strings.TrimSpace(c.HTTPAddr) != "" {
		if err := validateAddr("http_addr", c.HTTPAddr); err != nil {
			return err
		}
	}
	if err := validateAddr("relay_addr", c.RelayAddr); err != nil {
		return err
	}
	if strings.TrimSpace(c.Token) == "" {
		return errors.New("token is required")
	}
	if c.DialTimeoutSeconds < 0 {
		return errors.New("dial_timeout_seconds cannot be negative")
	}
	if err := c.LocalAuth.Validate(); err != nil {
		return err
	}
	return nil
}

func (c ClientConfig) DialTimeout() time.Duration {
	if c.DialTimeoutSeconds <= 0 {
		return 10 * time.Second
	}
	return time.Duration(c.DialTimeoutSeconds) * time.Second
}

func (c ClientConfig) Redacted() ClientConfig {
	c.Token = redact(c.Token)
	c.LocalAuth.Password = redact(c.LocalAuth.Password)
	return c
}

func (a ClientAuthConfig) Validate() error {
	if !a.Enabled {
		return nil
	}
	if strings.TrimSpace(a.Username) == "" {
		return errors.New("local_auth.username is required when local_auth.enabled is true")
	}
	if a.Password == "" {
		return errors.New("local_auth.password is required when local_auth.enabled is true")
	}
	if len(a.Username) > 255 {
		return errors.New("local_auth.username cannot be longer than 255 bytes")
	}
	if len(a.Password) > 255 {
		return errors.New("local_auth.password cannot be longer than 255 bytes")
	}
	return nil
}

func (c RelayConfig) Validate() error {
	if err := validateAddr("listen_addr", c.ListenAddr); err != nil {
		return err
	}
	if strings.TrimSpace(c.Token) == "" {
		return errors.New("token is required")
	}
	if c.DialTimeoutSeconds < 0 {
		return errors.New("dial_timeout_seconds cannot be negative")
	}
	if c.TLS.Enabled {
		if strings.TrimSpace(c.TLS.CertFile) == "" {
			return errors.New("tls.cert_file is required when tls.enabled is true")
		}
		if strings.TrimSpace(c.TLS.KeyFile) == "" {
			return errors.New("tls.key_file is required when tls.enabled is true")
		}
	}
	return nil
}

func (c RelayConfig) DialTimeout() time.Duration {
	if c.DialTimeoutSeconds <= 0 {
		return 10 * time.Second
	}
	return time.Duration(c.DialTimeoutSeconds) * time.Second
}

func (c RelayConfig) Redacted() RelayConfig {
	c.Token = redact(c.Token)
	return c
}

func redact(value string) string {
	if value == "" {
		return ""
	}
	return RedactedValue
}

func loadJSON(path string, dst any) error {
	file, err := os.Open(expandHome(path))
	if err != nil {
		return err
	}
	defer file.Close()

	dec := json.NewDecoder(file)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func writeJSON(path string, value any, force bool) error {
	path = expandHome(path)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	flag := os.O_WRONLY | os.O_CREATE
	if force {
		flag |= os.O_TRUNC
	} else {
		flag |= os.O_EXCL
	}

	file, err := os.OpenFile(path, flag, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	return nil
}

func validateAddr(name, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", name)
	}
	host, port, err := net.SplitHostPort(value)
	if err != nil {
		return fmt.Errorf("%s must be host:port: %w", name, err)
	}
	if strings.TrimSpace(host) == "" {
		return fmt.Errorf("%s host is required", name)
	}
	if strings.TrimSpace(port) == "" {
		return fmt.Errorf("%s port is required", name)
	}
	return nil
}

func defaultPath(file string) string {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		base = "."
	}
	return filepath.Join(base, "mingsui", file)
}

func expandHome(path string) string {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
