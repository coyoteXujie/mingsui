package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
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

type RelayProfile struct {
	Name      string          `json:"name"`
	RelayAddr string          `json:"relay_addr"`
	Token     string          `json:"token"`
	TLS       ClientTLSConfig `json:"tls"`
}

type RelaySubscription struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type ProxyProfile struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	URL      string `json:"url"`
}

type ClientConfig struct {
	LocalAddr          string              `json:"local_addr"`
	HTTPAddr           string              `json:"http_addr"`
	RelayAddr          string              `json:"relay_addr"`
	Token              string              `json:"token"`
	DialTimeoutSeconds int                 `json:"dial_timeout_seconds"`
	ActiveProfile      string              `json:"active_profile,omitempty"`
	ActiveProxyProfile string              `json:"active_proxy_profile,omitempty"`
	Profiles           []RelayProfile      `json:"profiles,omitempty"`
	ProxyProfiles      []ProxyProfile      `json:"proxy_profiles,omitempty"`
	Subscriptions      []RelaySubscription `json:"subscriptions,omitempty"`
	LocalAuth          ClientAuthConfig    `json:"local_auth"`
	TLS                ClientTLSConfig     `json:"tls"`
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
	MaxConnections       int            `json:"max_connections"`
	AllowPrivateNetworks bool           `json:"allow_private_networks"`
	TLS                  RelayTLSConfig `json:"tls"`
}

func DefaultClient() ClientConfig {
	return ClientConfig{
		LocalAddr:          "127.0.0.1:18080",
		HTTPAddr:           "127.0.0.1:18081",
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
	if err := validateRelayProfiles(c.Profiles); err != nil {
		return err
	}
	if err := validateProxyProfiles(c.ProxyProfiles); err != nil {
		return err
	}
	if err := validateRelaySubscriptions(c.Subscriptions); err != nil {
		return err
	}
	if strings.TrimSpace(c.ActiveProfile) != "" {
		if _, ok := findRelayProfile(c.Profiles, c.ActiveProfile); !ok {
			return fmt.Errorf("active_profile %q not found", c.ActiveProfile)
		}
	}
	if strings.TrimSpace(c.ActiveProxyProfile) != "" {
		if _, ok := findProxyProfile(c.ProxyProfiles, c.ActiveProxyProfile); !ok {
			return fmt.Errorf("active_proxy_profile %q not found", c.ActiveProxyProfile)
		}
	}
	return nil
}

func (c ClientConfig) DialTimeout() time.Duration {
	if c.DialTimeoutSeconds <= 0 {
		return 10 * time.Second
	}
	return time.Duration(c.DialTimeoutSeconds) * time.Second
}

// Clone 返回客户端配置副本，避免调用方通过 profile 切片修改内部状态。
func (c ClientConfig) Clone() ClientConfig {
	c.Profiles = append([]RelayProfile(nil), c.Profiles...)
	c.ProxyProfiles = append([]ProxyProfile(nil), c.ProxyProfiles...)
	c.Subscriptions = append([]RelaySubscription(nil), c.Subscriptions...)
	return c
}

func (c ClientConfig) Redacted() ClientConfig {
	c.Token = redact(c.Token)
	c.LocalAuth.Password = redact(c.LocalAuth.Password)
	c.Profiles = append([]RelayProfile(nil), c.Profiles...)
	for i := range c.Profiles {
		c.Profiles[i].Token = redact(c.Profiles[i].Token)
	}
	c.ProxyProfiles = append([]ProxyProfile(nil), c.ProxyProfiles...)
	for i := range c.ProxyProfiles {
		c.ProxyProfiles[i].URL = redact(c.ProxyProfiles[i].URL)
	}
	c.Subscriptions = append([]RelaySubscription(nil), c.Subscriptions...)
	for i := range c.Subscriptions {
		c.Subscriptions[i].URL = redact(c.Subscriptions[i].URL)
	}
	return c
}

func (c ClientConfig) ResolveProfile(profileName string) (ClientConfig, error) {
	name := strings.TrimSpace(profileName)
	if name == "" {
		name = strings.TrimSpace(c.ActiveProfile)
	}
	if name == "" {
		return c, nil
	}

	profile, ok := findRelayProfile(c.Profiles, name)
	if !ok {
		return ClientConfig{}, fmt.Errorf("profile %q not found", name)
	}
	c.ActiveProfile = profile.Name
	c.RelayAddr = profile.RelayAddr
	c.Token = profile.Token
	c.TLS = profile.TLS
	if err := c.Validate(); err != nil {
		return ClientConfig{}, err
	}
	return c, nil
}

func (c ClientConfig) ProfileNames() []string {
	names := make([]string, 0, len(c.Profiles))
	for _, profile := range c.Profiles {
		names = append(names, profile.Name)
	}
	return names
}

func (c ClientConfig) RelaySubscription(name string) (RelaySubscription, bool) {
	return findRelaySubscription(c.Subscriptions, name)
}

func (c ClientConfig) ProxyProfile(name string) (ProxyProfile, bool) {
	return findProxyProfile(c.ProxyProfiles, name)
}

// UpsertRelayProfile 新增或更新 relay profile。
func (c *ClientConfig) UpsertRelayProfile(profile RelayProfile, replace bool) error {
	profile.Name = strings.TrimSpace(profile.Name)
	profile.RelayAddr = strings.TrimSpace(profile.RelayAddr)
	profile.Token = strings.TrimSpace(profile.Token)
	if err := validateRelayProfile(profile); err != nil {
		return err
	}

	next := c.Clone()
	index := relayProfileIndex(next.Profiles, profile.Name)
	if index >= 0 {
		if !replace {
			return fmt.Errorf("profile %q 已存在", profile.Name)
		}
		next.Profiles[index] = profile
	} else {
		next.Profiles = append(next.Profiles, profile)
	}
	if err := next.Validate(); err != nil {
		return err
	}
	*c = next
	return nil
}

// ImportRelayProfiles 批量导入 relay profile，任意一个无效时不会修改原配置。
func (c *ClientConfig) ImportRelayProfiles(profiles []RelayProfile, replace bool) error {
	if len(profiles) == 0 {
		return fmt.Errorf("没有可导入的 relay profile")
	}

	next := c.Clone()
	for _, profile := range profiles {
		if err := next.UpsertRelayProfile(profile, replace); err != nil {
			return err
		}
	}
	*c = next
	return nil
}

// UpsertProxyProfile 新增或更新真实机场节点。
func (c *ClientConfig) UpsertProxyProfile(profile ProxyProfile, replace bool) error {
	profile.Name = strings.TrimSpace(profile.Name)
	profile.Protocol = strings.ToLower(strings.TrimSpace(profile.Protocol))
	profile.URL = strings.TrimSpace(profile.URL)
	if err := validateProxyProfile(profile); err != nil {
		return err
	}

	next := c.Clone()
	index := proxyProfileIndex(next.ProxyProfiles, profile.Name)
	if index >= 0 {
		if !replace {
			return fmt.Errorf("proxy profile %q 已存在", profile.Name)
		}
		next.ProxyProfiles[index] = profile
	} else {
		next.ProxyProfiles = append(next.ProxyProfiles, profile)
	}
	if err := next.Validate(); err != nil {
		return err
	}
	*c = next
	return nil
}

// ImportProxyProfiles 批量导入真实机场节点，任意一个无效时不会修改原配置。
func (c *ClientConfig) ImportProxyProfiles(profiles []ProxyProfile, replace bool) error {
	if len(profiles) == 0 {
		return fmt.Errorf("没有可导入的机场节点")
	}

	next := c.Clone()
	for _, profile := range profiles {
		if err := next.UpsertProxyProfile(profile, replace); err != nil {
			return err
		}
	}
	*c = next
	return nil
}

// UpsertRelaySubscription 新增或更新 relay 订阅地址。
func (c *ClientConfig) UpsertRelaySubscription(sub RelaySubscription, replace bool) error {
	sub.Name = strings.TrimSpace(sub.Name)
	sub.URL = strings.TrimSpace(sub.URL)
	if err := validateRelaySubscription(sub); err != nil {
		return err
	}

	next := c.Clone()
	index := relaySubscriptionIndex(next.Subscriptions, sub.Name)
	if index >= 0 {
		if !replace {
			return fmt.Errorf("subscription %q 已存在", sub.Name)
		}
		next.Subscriptions[index] = sub
	} else {
		next.Subscriptions = append(next.Subscriptions, sub)
	}
	if err := next.Validate(); err != nil {
		return err
	}
	*c = next
	return nil
}

// RemoveRelaySubscription 删除 relay 订阅地址。
func (c *ClientConfig) RemoveRelaySubscription(name string) error {
	name = strings.TrimSpace(name)
	index := relaySubscriptionIndex(c.Subscriptions, name)
	if index < 0 {
		return fmt.Errorf("subscription %q 不存在", name)
	}

	next := c.Clone()
	next.Subscriptions = append(next.Subscriptions[:index], next.Subscriptions[index+1:]...)
	if err := next.Validate(); err != nil {
		return err
	}
	*c = next
	return nil
}

// SelectRelayProfile 设置当前默认使用的 relay profile。
func (c *ClientConfig) SelectRelayProfile(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("profile 名称不能为空")
	}
	if _, err := c.ResolveProfile(name); err != nil {
		return err
	}

	next := c.Clone()
	next.ActiveProfile = name
	next.ActiveProxyProfile = ""
	if err := next.Validate(); err != nil {
		return err
	}
	*c = next
	return nil
}

// SelectProxyProfile 设置当前默认使用的真实机场节点。
func (c *ClientConfig) SelectProxyProfile(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("proxy profile 名称不能为空")
	}
	if _, ok := findProxyProfile(c.ProxyProfiles, name); !ok {
		return fmt.Errorf("proxy profile %q not found", name)
	}

	next := c.Clone()
	next.ActiveProxyProfile = name
	next.ActiveProfile = ""
	if err := next.Validate(); err != nil {
		return err
	}
	*c = next
	return nil
}

// RemoveRelayProfile 删除 relay profile，并在删除当前 profile 时清空选择。
func (c *ClientConfig) RemoveRelayProfile(name string) error {
	name = strings.TrimSpace(name)
	index := relayProfileIndex(c.Profiles, name)
	if index < 0 {
		return fmt.Errorf("profile %q 不存在", name)
	}

	next := c.Clone()
	next.Profiles = append(next.Profiles[:index], next.Profiles[index+1:]...)
	if next.ActiveProfile == name {
		next.ActiveProfile = ""
	}
	if err := next.Validate(); err != nil {
		return err
	}
	*c = next
	return nil
}

// RemoveProxyProfile 删除真实机场节点，并在删除当前节点时清空选择。
func (c *ClientConfig) RemoveProxyProfile(name string) error {
	name = strings.TrimSpace(name)
	index := proxyProfileIndex(c.ProxyProfiles, name)
	if index < 0 {
		return fmt.Errorf("proxy profile %q 不存在", name)
	}

	next := c.Clone()
	next.ProxyProfiles = append(next.ProxyProfiles[:index], next.ProxyProfiles[index+1:]...)
	if next.ActiveProxyProfile == name {
		next.ActiveProxyProfile = ""
	}
	if err := next.Validate(); err != nil {
		return err
	}
	*c = next
	return nil
}

// RenameRelayProfile 重命名 relay profile，并同步当前默认选择。
func (c *ClientConfig) RenameRelayProfile(oldName, newName string) error {
	oldName = strings.TrimSpace(oldName)
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return fmt.Errorf("profile 新名称不能为空")
	}

	index := relayProfileIndex(c.Profiles, oldName)
	if index < 0 {
		return fmt.Errorf("profile %q 不存在", oldName)
	}
	if relayProfileIndex(c.Profiles, newName) >= 0 {
		return fmt.Errorf("profile %q 已存在", newName)
	}

	next := c.Clone()
	next.Profiles[index].Name = newName
	if next.ActiveProfile == oldName {
		next.ActiveProfile = newName
	}
	if err := next.Validate(); err != nil {
		return err
	}
	*c = next
	return nil
}

func validateRelaySubscriptions(subscriptions []RelaySubscription) error {
	seen := make(map[string]struct{}, len(subscriptions))
	for _, sub := range subscriptions {
		name := strings.TrimSpace(sub.Name)
		if name == "" {
			return errors.New("subscriptions.name is required")
		}
		if _, ok := seen[name]; ok {
			return fmt.Errorf("duplicate subscription %q", name)
		}
		seen[name] = struct{}{}
		if err := validateRelaySubscription(sub); err != nil {
			return err
		}
	}
	return nil
}

func validateRelaySubscription(sub RelaySubscription) error {
	name := strings.TrimSpace(sub.Name)
	if name == "" {
		return errors.New("subscriptions.name is required")
	}
	rawURL := strings.TrimSpace(sub.URL)
	if rawURL == "" {
		return fmt.Errorf("subscriptions.%s url is required", name)
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("subscriptions.%s url must be http or https URL", name)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("subscriptions.%s url must be http or https URL", name)
	}
	return nil
}

func validateRelayProfiles(profiles []RelayProfile) error {
	seen := make(map[string]struct{}, len(profiles))
	for _, profile := range profiles {
		name := strings.TrimSpace(profile.Name)
		if name == "" {
			return errors.New("profiles.name is required")
		}
		if _, ok := seen[name]; ok {
			return fmt.Errorf("duplicate profile %q", name)
		}
		seen[name] = struct{}{}
		if err := validateRelayProfile(profile); err != nil {
			return err
		}
	}
	return nil
}

func validateRelayProfile(profile RelayProfile) error {
	name := strings.TrimSpace(profile.Name)
	if name == "" {
		return errors.New("profiles.name is required")
	}
	if err := validateAddr("profiles.relay_addr", profile.RelayAddr); err != nil {
		return err
	}
	if strings.TrimSpace(profile.Token) == "" {
		return fmt.Errorf("profiles.%s token is required", name)
	}
	return nil
}

func validateProxyProfiles(profiles []ProxyProfile) error {
	seen := make(map[string]struct{}, len(profiles))
	for _, profile := range profiles {
		name := strings.TrimSpace(profile.Name)
		if name == "" {
			return errors.New("proxy_profiles.name is required")
		}
		if _, ok := seen[name]; ok {
			return fmt.Errorf("duplicate proxy profile %q", name)
		}
		seen[name] = struct{}{}
		if err := validateProxyProfile(profile); err != nil {
			return err
		}
	}
	return nil
}

func validateProxyProfile(profile ProxyProfile) error {
	name := strings.TrimSpace(profile.Name)
	if name == "" {
		return errors.New("proxy_profiles.name is required")
	}
	protocol := strings.ToLower(strings.TrimSpace(profile.Protocol))
	if !isSupportedProxyProtocol(protocol) {
		return fmt.Errorf("proxy_profiles.%s protocol is not supported", name)
	}
	rawURL := strings.TrimSpace(profile.URL)
	if rawURL == "" {
		return fmt.Errorf("proxy_profiles.%s url is required", name)
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || strings.ToLower(parsed.Scheme) != protocol {
		return fmt.Errorf("proxy_profiles.%s url must use %s://", name, protocol)
	}
	return nil
}

func isSupportedProxyProtocol(protocol string) bool {
	switch protocol {
	case "ss", "ssr", "vmess", "vless", "trojan", "hysteria", "hysteria2", "tuic":
		return true
	default:
		return false
	}
}

func findRelayProfile(profiles []RelayProfile, name string) (RelayProfile, bool) {
	name = strings.TrimSpace(name)
	for _, profile := range profiles {
		if strings.TrimSpace(profile.Name) == name {
			return profile, true
		}
	}
	return RelayProfile{}, false
}

func relayProfileIndex(profiles []RelayProfile, name string) int {
	name = strings.TrimSpace(name)
	for i, profile := range profiles {
		if strings.TrimSpace(profile.Name) == name {
			return i
		}
	}
	return -1
}

func findProxyProfile(profiles []ProxyProfile, name string) (ProxyProfile, bool) {
	name = strings.TrimSpace(name)
	for _, profile := range profiles {
		if strings.TrimSpace(profile.Name) == name {
			return profile, true
		}
	}
	return ProxyProfile{}, false
}

func proxyProfileIndex(profiles []ProxyProfile, name string) int {
	name = strings.TrimSpace(name)
	for i, profile := range profiles {
		if strings.TrimSpace(profile.Name) == name {
			return i
		}
	}
	return -1
}

func findRelaySubscription(subscriptions []RelaySubscription, name string) (RelaySubscription, bool) {
	name = strings.TrimSpace(name)
	for _, sub := range subscriptions {
		if strings.TrimSpace(sub.Name) == name {
			return sub, true
		}
	}
	return RelaySubscription{}, false
}

func relaySubscriptionIndex(subscriptions []RelaySubscription, name string) int {
	name = strings.TrimSpace(name)
	for i, sub := range subscriptions {
		if strings.TrimSpace(sub.Name) == name {
			return i
		}
	}
	return -1
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
	if c.MaxConnections < 0 {
		return errors.New("max_connections cannot be negative")
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
