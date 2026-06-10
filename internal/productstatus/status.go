package productstatus

import (
	"fmt"
	"net"
	"strings"

	"github.com/coyoteXujie/mingsui/internal/config"
	"github.com/coyoteXujie/mingsui/internal/mihomo"
)

type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

type Action struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Command     string   `json:"command,omitempty"`
	Description string   `json:"description,omitempty"`
	Severity    Severity `json:"severity"`
}

type Status struct {
	OK                bool     `json:"ok"`
	ConfigPath        string   `json:"config_path,omitempty"`
	Mode              string   `json:"mode"`
	Readiness         string   `json:"readiness"`
	Managed           bool     `json:"managed"`
	SelectedType      string   `json:"selected_type,omitempty"`
	SelectedProfile   string   `json:"selected_profile,omitempty"`
	SelectedProxy     string   `json:"selected_proxy,omitempty"`
	ProxyProtocol     string   `json:"proxy_protocol,omitempty"`
	RelayProfiles     int      `json:"relay_profiles"`
	ProxyProfiles     int      `json:"proxy_profiles"`
	Subscriptions     int      `json:"subscriptions"`
	LocalAddr         string   `json:"local_addr,omitempty"`
	HTTPAddr          string   `json:"http_addr,omitempty"`
	RelayAddr         string   `json:"relay_addr,omitempty"`
	AuthEnabled       bool     `json:"auth_enabled"`
	TLSEnabled        bool     `json:"tls_enabled"`
	DefaultToken      bool     `json:"default_token,omitempty"`
	LocalProxyExposed bool     `json:"local_proxy_exposed,omitempty"`
	Message           string   `json:"message"`
	Warnings          []string `json:"warnings,omitempty"`
	Actions           []Action `json:"actions,omitempty"`
}

type Options struct {
	ConfigPath  string
	Managed     bool
	AutoProfile bool
	ForceRelay  bool
	ProfileName string
	Overrides   Overrides
}

type Overrides struct {
	LocalAddr   string
	HTTPAddr    string
	RelayAddr   string
	Token       string
	AuthEnabled bool
	AuthUser    string
	AuthPass    string
}

func Evaluate(cfg config.ClientConfig, opts Options) Status {
	status := Status{
		OK:            true,
		ConfigPath:    opts.ConfigPath,
		Mode:          "relay",
		Readiness:     "ready",
		Managed:       opts.Managed,
		RelayProfiles: len(cfg.Profiles),
		ProxyProfiles: len(cfg.ProxyProfiles),
		Subscriptions: len(cfg.Subscriptions),
	}

	if !opts.ForceRelay {
		proxyCfg := cfg.Clone()
		applyLocalOverrides(&proxyCfg, opts.Overrides)
		if proxy, ok := ResolveProxyProfile(proxyCfg, opts.AutoProfile); ok {
			fillCommon(&status, proxyCfg)
			status.Mode = "proxy"
			status.SelectedType = "proxy"
			status.SelectedProxy = proxy.Name
			status.ProxyProtocol = proxy.Protocol
			if !mihomo.CanExportProfile(proxy) {
				status.OK = false
				status.Readiness = "blocked"
				status.Message = fmt.Sprintf("当前机场节点 %s 暂不支持直接连接，请选择可连接节点", proxy.Name)
				status.addAction("list_proxy", "查看可连接节点", "mingsui config proxy list", "查看哪些节点可以直接连接或自动选择", SeverityError)
				return status
			}
			status.Message = "当前使用机场节点；连接会启动 Mihomo，本地 HTTP/SOCKS5 可供终端和 AI Agent 使用"
			status.addAction("connect", "启动连接", "mingsui connect", "保持当前代理连接", SeverityInfo)
			status.addAction("env", "导出代理环境变量", `eval "$(mingsui env)"`, "让当前终端和子进程走明隧代理", SeverityInfo)
			status.addAction("exec", "临时代理执行命令", "mingsui exec -connect -- curl https://example.com", "只为单个命令启动临时代理", SeverityInfo)
			addCommonWarnings(&status, proxyCfg)
			return status
		}
		if HasProxyModeWithoutAutoSelectableSelection(proxyCfg) {
			fillCommon(&status, proxyCfg)
			status.OK = false
			status.Mode = "proxy"
			status.Readiness = "blocked"
			status.SelectedType = "proxy"
			status.Message = "当前机场订阅中没有可自动选择的国外节点"
			status.addAction("list_proxy", "查看节点支持情况", "mingsui config proxy list", "确认哪些节点可连接、哪些节点只适合作为国内节点", SeverityError)
			status.addAction("check_best", "测速并选择最快节点", "mingsui config proxy check -select-best", "检测可用国外节点并自动选择最快节点", SeverityInfo)
			return status
		}
	}

	relayCfg, selectedProfile, err := resolveRelayConfig(cfg, opts)
	if err != nil {
		fillCommon(&status, cfg)
		status.OK = false
		status.Readiness = "blocked"
		status.SelectedType = "relay"
		status.Message = fmt.Sprintf("选择 relay profile 失败: %v", err)
		status.addAction("list_profile", "查看 relay profile", "mingsui config profile list", "确认已保存的 relay profile 名称", SeverityError)
		return status
	}
	fillCommon(&status, relayCfg)
	status.Mode = "relay"
	status.SelectedType = "relay"
	status.SelectedProfile = selectedProfile
	if err := relayCfg.Validate(); err != nil {
		status.OK = false
		status.Readiness = "blocked"
		status.Message = fmt.Sprintf("配置不正确: %v", err)
		status.addAction("show_config", "查看当前配置", "mingsui config show", "检查本地监听、relay、token 和认证设置", SeverityError)
		return status
	}
	status.Message = "当前使用自建 relay；连接会启动本地 HTTP/SOCKS5 代理"
	status.addAction("connect", "启动连接", "mingsui connect", "保持当前代理连接", SeverityInfo)
	status.addAction("env", "导出代理环境变量", `eval "$(mingsui env)"`, "让当前终端和子进程走明隧代理", SeverityInfo)
	status.addAction("doctor", "运行连接诊断", "mingsui doctor", "检查本地端口、relay 健康和配置问题", SeverityInfo)
	addCommonWarnings(&status, relayCfg)
	if status.DefaultToken && status.RelayProfiles == 0 && status.ProxyProfiles == 0 {
		status.Readiness = "needs_setup"
		status.addAction("import", "导入机场订阅", "mingsui import -source <订阅URL> -check", "最快完成可用代理配置", SeverityWarning)
		status.addAction("add_profile", "添加 relay profile", "mingsui config profile add tokyo -relay <host:port> -token <token>", "使用自建 relay 时保存一个可复用 profile", SeverityWarning)
	}
	return status
}

func ResolveProxyProfile(cfg config.ClientConfig, autoProfile bool) (config.ProxyProfile, bool) {
	name := strings.TrimSpace(cfg.ActiveProxyProfile)
	if name == "" && strings.TrimSpace(cfg.ActiveProfile) == "" && autoProfile && len(cfg.ProxyProfiles) > 0 {
		var ok bool
		name, ok = mihomo.FirstAutoSelectableProfileName(cfg.ProxyProfiles)
		if !ok {
			return config.ProxyProfile{}, false
		}
	}
	if name == "" {
		return config.ProxyProfile{}, false
	}
	return cfg.ProxyProfile(name)
}

func HasProxyModeWithoutAutoSelectableSelection(cfg config.ClientConfig) bool {
	return strings.TrimSpace(cfg.ActiveProfile) == "" && len(cfg.ProxyProfiles) > 0
}

func resolveRelayConfig(cfg config.ClientConfig, opts Options) (config.ClientConfig, string, error) {
	name := strings.TrimSpace(opts.ProfileName)
	if name == "" {
		name = strings.TrimSpace(cfg.ActiveProfile)
	}
	if name == "" && opts.AutoProfile && len(cfg.Profiles) > 0 && strings.TrimSpace(opts.Overrides.RelayAddr) == "" && strings.TrimSpace(opts.Overrides.Token) == "" {
		name = cfg.Profiles[0].Name
	}
	resolved, err := cfg.ResolveProfile(name)
	if err != nil {
		return config.ClientConfig{}, "", err
	}
	applyOverrides(&resolved, opts.Overrides)
	return resolved, strings.TrimSpace(resolved.ActiveProfile), nil
}

func fillCommon(status *Status, cfg config.ClientConfig) {
	status.RelayProfiles = len(cfg.Profiles)
	status.ProxyProfiles = len(cfg.ProxyProfiles)
	status.Subscriptions = len(cfg.Subscriptions)
	status.LocalAddr = cfg.LocalAddr
	status.HTTPAddr = cfg.HTTPAddr
	status.RelayAddr = cfg.RelayAddr
	status.AuthEnabled = cfg.LocalAuth.Enabled
	status.TLSEnabled = cfg.TLS.Enabled
	status.DefaultToken = cfg.Token == "change-me"
	status.LocalProxyExposed = LocalProxyMayBeExposed(cfg)
}

func addCommonWarnings(status *Status, cfg config.ClientConfig) {
	if cfg.Token == "change-me" && status.Mode == "relay" {
		status.addWarning("当前使用默认 token，生产环境必须修改")
		status.addAction("token", "生成安全 token", "mingsui token", "生成随机 token 后写入客户端和 relay 配置", SeverityWarning)
	}
	if LocalProxyMayBeExposed(cfg) {
		status.addWarning("本地代理监听在非 loopback 地址且未启用本地认证")
		status.addAction("local_auth", "开启本地代理认证", "mingsui config init -auth-user <user> -auth-pass <pass> -force", "对外暴露本地代理前必须加认证或改回 127.0.0.1", SeverityWarning)
	}
}

func (s *Status) addWarning(message string) {
	if strings.TrimSpace(message) != "" {
		s.Warnings = append(s.Warnings, message)
	}
}

func (s *Status) addAction(id, label, command, description string, severity Severity) {
	s.Actions = append(s.Actions, Action{
		ID:          id,
		Label:       label,
		Command:     command,
		Description: description,
		Severity:    severity,
	})
}

func applyLocalOverrides(cfg *config.ClientConfig, overrides Overrides) {
	if overrides.LocalAddr != "" {
		cfg.LocalAddr = overrides.LocalAddr
	}
	if overrides.HTTPAddr != "" {
		cfg.HTTPAddr = overrides.HTTPAddr
	}
	if overrides.AuthEnabled {
		cfg.LocalAuth.Enabled = true
	}
	if overrides.AuthUser != "" {
		cfg.LocalAuth.Enabled = true
		cfg.LocalAuth.Username = overrides.AuthUser
	}
	if overrides.AuthPass != "" {
		cfg.LocalAuth.Enabled = true
		cfg.LocalAuth.Password = overrides.AuthPass
	}
}

func applyOverrides(cfg *config.ClientConfig, overrides Overrides) {
	applyLocalOverrides(cfg, overrides)
	if overrides.RelayAddr != "" {
		cfg.RelayAddr = overrides.RelayAddr
	}
	if overrides.Token != "" {
		cfg.Token = overrides.Token
	}
}

func LocalProxyMayBeExposed(cfg config.ClientConfig) bool {
	if cfg.LocalAuth.Enabled {
		return false
	}
	if !listenAddrIsLoopback(cfg.LocalAddr) {
		return true
	}
	return strings.TrimSpace(cfg.HTTPAddr) != "" && !listenAddrIsLoopback(cfg.HTTPAddr)
}

func listenAddrIsLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return strings.EqualFold(host, "localhost")
	}
	return ip.IsLoopback()
}
