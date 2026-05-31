package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/coyoteXujie/mingsui/internal/buildinfo"
	"github.com/coyoteXujie/mingsui/internal/client"
	"github.com/coyoteXujie/mingsui/internal/config"
	"github.com/coyoteXujie/mingsui/internal/diagnostic"
	"github.com/coyoteXujie/mingsui/internal/mihomo"
	"github.com/coyoteXujie/mingsui/internal/protocol"
	"github.com/coyoteXujie/mingsui/internal/security"
	"github.com/coyoteXujie/mingsui/internal/systemproxy"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 2
	}

	switch args[0] {
	case "connect":
		return runConnect(args[1:])
	case "import":
		return importClientProfilesProduct(args[1:])
	case "status":
		return runStatus(args[1:])
	case "disconnect":
		return runDisconnect(args[1:])
	case "env":
		return runEnv(args[1:])
	case "exec":
		return runExec(args[1:])
	case "system-proxy":
		return runSystemProxy(args[1:])
	case "kernel":
		return runKernel(args[1:])
	case "run":
		return runClient(args[1:])
	case "doctor":
		return runDoctor(args[1:])
	case "config":
		return runConfig(args[1:])
	case "token":
		return runToken(args[1:])
	case "version":
		fmt.Println(buildinfo.String())
		return 0
	case "help", "-h", "--help":
		printUsage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "未知命令 %q\n\n", args[0])
		printUsage()
		return 2
	}
}

func runToken(args []string) int {
	fs := flag.NewFlagSet("token", flag.ContinueOnError)
	byteLen := fs.Int("bytes", security.DefaultTokenBytes, "随机字节长度，至少 16")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	token, err := security.GenerateToken(*byteLen)
	if err != nil {
		fmt.Fprintf(os.Stderr, "生成 token 失败: %v\n", err)
		return 1
	}
	fmt.Println(token)
	return 0
}

func runKernel(args []string) int {
	if len(args) == 0 {
		printKernelUsage()
		return 2
	}

	switch args[0] {
	case "export":
		return exportKernelConfig(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "未知内核命令 %q\n\n", args[0])
		printKernelUsage()
		return 2
	}
}

func exportKernelConfig(args []string) int {
	fs := flag.NewFlagSet("kernel export", flag.ContinueOnError)
	cfgPath := fs.String("config", config.DefaultClientPath(), "客户端配置文件路径")
	outputPath := fs.String("output", "", "输出文件路径，留空则打印到 stdout")
	format := fs.String("format", "mihomo", "导出格式，目前支持 mihomo")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := config.LoadClient(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		return 1
	}
	var data []byte
	switch strings.ToLower(strings.TrimSpace(*format)) {
	case "mihomo":
		data, err = mihomo.Generate(cfg, mihomo.Options{})
	default:
		err = fmt.Errorf("不支持的内核格式 %q", *format)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "导出内核配置失败: %v\n", err)
		return 1
	}
	if *outputPath == "" {
		if _, err := os.Stdout.Write(data); err != nil {
			fmt.Fprintf(os.Stderr, "输出内核配置失败: %v\n", err)
			return 1
		}
		return 0
	}
	if err := os.WriteFile(*outputPath, data, 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "写入内核配置失败: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stdout, "已导出 %s 配置到 %s\n", *format, *outputPath)
	return 0
}

func runDoctor(args []string) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	cfgPath := fs.String("config", config.DefaultClientPath(), "客户端配置文件路径")
	localAddr := fs.String("local", "", "本地 SOCKS5 监听地址")
	httpAddr := fs.String("http", "", "本地 HTTP 代理监听地址")
	relayAddr := fs.String("relay", "", "relay 服务端地址")
	token := fs.String("token", "", "客户端和 relay 共享的 token")
	profileName := fs.String("profile", "", "使用指定 relay profile")
	authEnabled := fs.Bool("auth", false, "启用本地代理认证")
	authUser := fs.String("auth-user", "", "本地代理认证用户名")
	authPass := fs.String("auth-pass", "", "本地代理认证密码")
	skipLocal := fs.Bool("skip-local", false, "跳过本地监听端口可用性检查")
	jsonOutput := fs.Bool("json", false, "以 JSON 格式输出诊断结果")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	report := diagnostic.NewReport(*cfgPath)
	cfg, err := loadClientOrDefault(*cfgPath)
	if err != nil {
		report.Fail(fmt.Sprintf("加载配置失败: %v", err))
		if *jsonOutput {
			return writeDiagnosticReport(report)
		}
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		return 1
	}
	cfg, err = cfg.ResolveProfile(*profileName)
	if err != nil {
		report.Fail(fmt.Sprintf("选择 profile 失败: %v", err))
		if *jsonOutput {
			return writeDiagnosticReport(report)
		}
		fmt.Fprintf(os.Stderr, "选择 profile 失败: %v\n", err)
		return 1
	}
	applyClientOverrides(&cfg, *localAddr, *httpAddr, *relayAddr, *token, *authEnabled, *authUser, *authPass)
	if err := cfg.Validate(); err != nil {
		report.Fail(fmt.Sprintf("配置不正确: %v", err))
		if *jsonOutput {
			return writeDiagnosticReport(report)
		}
		fmt.Fprintf(os.Stderr, "配置不正确: %v\n", err)
		return 1
	}

	if !*jsonOutput {
		fmt.Fprintf(os.Stdout, "配置文件: %s\n", *cfgPath)
	}
	if cfg.Token == "change-me" {
		warning := "当前使用默认 token，生产环境必须修改"
		report.AddWarning(warning)
		if !*jsonOutput {
			fmt.Fprintf(os.Stdout, "警告: %s\n", warning)
		}
	}
	if localProxyMayBeExposed(cfg) {
		warning := "本地代理监听在非 loopback 地址且未启用本地认证"
		report.AddWarning(warning)
		if !*jsonOutput {
			fmt.Fprintf(os.Stdout, "警告: %s\n", warning)
		}
	}

	if !*skipLocal {
		check := checkListen("socks5_listen", "SOCKS5 监听地址", cfg.LocalAddr)
		report.AddCheck(check)
		if !*jsonOutput {
			printListenResult(check)
		}
		if strings.TrimSpace(cfg.HTTPAddr) != "" {
			check := checkListen("http_listen", "HTTP 代理监听地址", cfg.HTTPAddr)
			report.AddCheck(check)
			if !*jsonOutput {
				printListenResult(check)
			}
		}
	}

	logger := log.New(io.Discard, "", 0)
	service, err := client.NewService(cfg, logger)
	if err != nil {
		report.Fail(fmt.Sprintf("创建客户端失败: %v", err))
		if *jsonOutput {
			return writeDiagnosticReport(report)
		}
		fmt.Fprintf(os.Stderr, "创建客户端失败: %v\n", err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.DialTimeout()+2*time.Second)
	defer cancel()
	health, err := service.CheckRelayStatus(ctx)
	relayCheck := diagnostic.Check{
		Name:   "relay_health",
		Label:  "relay 健康检查",
		Target: cfg.RelayAddr,
		OK:     err == nil,
	}
	if err != nil {
		relayCheck.Error = err.Error()
	} else {
		relayCheck.Metrics = health.Metrics
	}
	report.AddCheck(relayCheck)
	if !*jsonOutput {
		printRelayHealthResult(relayCheck)
	}

	if *jsonOutput {
		return writeDiagnosticReport(report)
	}
	return report.ExitCode()
}

func runClient(args []string) int {
	return runClientCommand("run", args, false)
}

func runConnect(args []string) int {
	return runClientCommand("connect", args, true)
}

func runClientCommand(name string, args []string, autoProfileDefault bool) int {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	cfgPath := fs.String("config", config.DefaultClientPath(), "客户端配置文件路径")
	localAddr := fs.String("local", "", "本地 SOCKS5 监听地址")
	httpAddr := fs.String("http", "", "本地 HTTP 代理监听地址")
	relayAddr := fs.String("relay", "", "relay 服务端地址")
	token := fs.String("token", "", "客户端和 relay 共享的 token")
	profileName := fs.String("profile", "", "使用指定 relay profile")
	autoProfile := fs.Bool("auto-profile", autoProfileDefault, "未选择 profile 时自动使用第一个 relay profile")
	authEnabled := fs.Bool("auth", false, "启用本地代理认证")
	authUser := fs.String("auth-user", "", "本地代理认证用户名")
	authPass := fs.String("auth-pass", "", "本地代理认证密码")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := loadClientOrDefault(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		return 1
	}
	if name == "connect" && *profileName == "" && *relayAddr == "" && *token == "" {
		if proxy, ok := resolveClientProxyProfile(cfg, true); ok {
			applyClientOverrides(&cfg, *localAddr, *httpAddr, "", "", *authEnabled, *authUser, *authPass)
			return runProxyKernel(cfg, proxy)
		}
	}
	var selectedProfile string
	cfg, selectedProfile, err = resolveClientProfile(cfg, *profileName, *autoProfile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "选择 profile 失败: %v\n", err)
		return 1
	}
	applyClientOverrides(&cfg, *localAddr, *httpAddr, *relayAddr, *token, *authEnabled, *authUser, *authPass)
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "配置不正确: %v\n", err)
		return 1
	}
	if cfg.Token == "change-me" {
		fmt.Fprintln(os.Stderr, "警告: 当前使用默认 token，对外暴露 relay 前必须修改")
	}
	if localProxyMayBeExposed(cfg) {
		fmt.Fprintln(os.Stderr, "警告: 本地代理监听在非 loopback 地址且未启用本地认证")
	}
	if selectedProfile != "" && name == "connect" {
		fmt.Fprintf(os.Stderr, "使用 profile: %s\n", selectedProfile)
	}

	logger := log.New(os.Stderr, "mingsui client: ", log.LstdFlags)
	service, err := client.NewService(cfg, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建客户端失败: %v\n", err)
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := service.Serve(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "运行客户端失败: %v\n", err)
		return 1
	}
	return 0
}

func runProxyKernel(cfg config.ClientConfig, proxy config.ProxyProfile) int {
	if localProxyMayBeExposed(cfg) {
		fmt.Fprintln(os.Stderr, "警告: 本地代理监听在非 loopback 地址")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Fprintf(os.Stderr, "使用机场节点: %s（%s）\n", proxy.Name, proxy.Protocol)
	fmt.Fprintln(os.Stderr, "正在启动 Mihomo 内核；停止当前进程即可断开")
	if err := mihomo.Run(ctx, cfg, mihomo.Options{Stdout: os.Stderr, Stderr: os.Stderr}); err != nil {
		fmt.Fprintf(os.Stderr, "启动 Mihomo 失败: %v\n", err)
		return 1
	}
	return 0
}

type cliStatus struct {
	OK              bool   `json:"ok"`
	ConfigPath      string `json:"config_path"`
	Mode            string `json:"mode"`
	Managed         bool   `json:"managed"`
	SelectedType    string `json:"selected_type,omitempty"`
	SelectedProfile string `json:"selected_profile,omitempty"`
	SelectedProxy   string `json:"selected_proxy,omitempty"`
	ProxyProtocol   string `json:"proxy_protocol,omitempty"`
	RelayProfiles   int    `json:"relay_profiles"`
	ProxyProfiles   int    `json:"proxy_profiles"`
	LocalAddr       string `json:"local_addr"`
	HTTPAddr        string `json:"http_addr,omitempty"`
	RelayAddr       string `json:"relay_addr"`
	AuthEnabled     bool   `json:"auth_enabled"`
	TLSEnabled      bool   `json:"tls_enabled"`
	Message         string `json:"message"`
}

func runStatus(args []string) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	cfgPath := fs.String("config", config.DefaultClientPath(), "客户端配置文件路径")
	localAddr := fs.String("local", "", "本地 SOCKS5 监听地址")
	httpAddr := fs.String("http", "", "本地 HTTP 代理监听地址")
	relayAddr := fs.String("relay", "", "relay 服务端地址")
	token := fs.String("token", "", "客户端和 relay 共享的 token")
	profileName := fs.String("profile", "", "使用指定 relay profile")
	autoProfile := fs.Bool("auto-profile", true, "未选择 profile 时自动使用第一个 relay profile")
	authEnabled := fs.Bool("auth", false, "启用本地代理认证")
	authUser := fs.String("auth-user", "", "本地代理认证用户名")
	authPass := fs.String("auth-pass", "", "本地代理认证密码")
	jsonOutput := fs.Bool("json", true, "以 JSON 格式输出状态，传 -json=false 可输出文本")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	status := cliStatus{
		OK:         false,
		ConfigPath: *cfgPath,
		Mode:       "foreground",
		Managed:    false,
		Message:    "CLI 使用前台连接模式；运行 mingsui connect 后保持该进程即可联网",
	}
	cfg, err := loadClientOrDefault(*cfgPath)
	if err != nil {
		status.Message = fmt.Sprintf("加载配置失败: %v", err)
		return writeCLIStatus(status, *jsonOutput)
	}
	status.RelayProfiles = len(cfg.Profiles)
	status.ProxyProfiles = len(cfg.ProxyProfiles)
	if proxy, ok := resolveClientProxyProfile(cfg, true); ok && *profileName == "" && *relayAddr == "" && *token == "" {
		status.OK = true
		status.Mode = "proxy"
		status.SelectedType = "proxy"
		status.SelectedProxy = proxy.Name
		status.ProxyProtocol = proxy.Protocol
		status.LocalAddr = cfg.LocalAddr
		status.HTTPAddr = cfg.HTTPAddr
		status.Message = "当前选择的是机场节点；运行 mingsui connect 会启动 Mihomo 内核"
		return writeCLIStatus(status, *jsonOutput)
	}
	var selectedProfile string
	cfg, selectedProfile, err = resolveClientProfile(cfg, *profileName, *autoProfile)
	if err != nil {
		status.Message = fmt.Sprintf("选择 profile 失败: %v", err)
		return writeCLIStatus(status, *jsonOutput)
	}
	applyClientOverrides(&cfg, *localAddr, *httpAddr, *relayAddr, *token, *authEnabled, *authUser, *authPass)
	if err := cfg.Validate(); err != nil {
		status.Message = fmt.Sprintf("配置不正确: %v", err)
		return writeCLIStatus(status, *jsonOutput)
	}

	status.OK = true
	status.SelectedType = "relay"
	status.SelectedProfile = selectedProfile
	status.RelayProfiles = len(cfg.Profiles)
	status.ProxyProfiles = len(cfg.ProxyProfiles)
	status.LocalAddr = cfg.LocalAddr
	status.HTTPAddr = cfg.HTTPAddr
	status.RelayAddr = cfg.RelayAddr
	status.AuthEnabled = cfg.LocalAuth.Enabled
	status.TLSEnabled = cfg.TLS.Enabled
	return writeCLIStatus(status, *jsonOutput)
}

func runDisconnect(args []string) int {
	fs := flag.NewFlagSet("disconnect", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", true, "以 JSON 格式输出结果，传 -json=false 可输出文本")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	status := cliStatus{
		OK:      true,
		Mode:    "foreground",
		Managed: false,
		Message: "CLI 当前以前台进程方式连接；请停止正在运行的 mingsui connect 或 mingsui run 进程。桌面端可直接点击断开。",
	}
	return writeCLIStatus(status, *jsonOutput)
}

func writeCLIStatus(status cliStatus, jsonOutput bool) int {
	if jsonOutput {
		if err := writeJSON(os.Stdout, status); err != nil {
			fmt.Fprintf(os.Stderr, "输出状态失败: %v\n", err)
			return 1
		}
	} else if status.OK {
		fmt.Fprintln(os.Stdout, status.Message)
		if status.RelayAddr != "" {
			fmt.Fprintf(os.Stdout, "relay: %s\n", status.RelayAddr)
		}
		if status.LocalAddr != "" {
			fmt.Fprintf(os.Stdout, "SOCKS5: %s\n", status.LocalAddr)
		}
		if status.HTTPAddr != "" {
			fmt.Fprintf(os.Stdout, "HTTP: %s\n", status.HTTPAddr)
		}
	} else {
		fmt.Fprintln(os.Stderr, status.Message)
	}
	if status.OK {
		return 0
	}
	return 1
}

type proxyEnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func runEnv(args []string) int {
	fs := flag.NewFlagSet("env", flag.ContinueOnError)
	cfgPath := fs.String("config", config.DefaultClientPath(), "客户端配置文件路径")
	localAddr := fs.String("local", "", "本地 SOCKS5 监听地址")
	httpAddr := fs.String("http", "", "本地 HTTP 代理监听地址")
	noProxy := fs.String("no-proxy", "localhost,127.0.0.1,::1", "NO_PROXY/no_proxy 值")
	format := fs.String("format", "shell", "输出格式：shell 或 json")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := loadClientOrDefault(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		return 1
	}
	applyClientOverrides(&cfg, *localAddr, *httpAddr, "", "", false, "", "")
	vars := proxyEnv(cfg, *noProxy)
	if len(vars) == 0 {
		fmt.Fprintln(os.Stderr, "配置中没有可用的本地代理监听地址")
		return 1
	}

	switch strings.ToLower(strings.TrimSpace(*format)) {
	case "shell":
		for _, item := range vars {
			fmt.Fprintf(os.Stdout, "export %s=%s\n", item.Name, shellQuote(item.Value))
		}
	case "json":
		if err := writeJSON(os.Stdout, vars); err != nil {
			fmt.Fprintf(os.Stderr, "输出代理环境失败: %v\n", err)
			return 1
		}
	default:
		fmt.Fprintf(os.Stderr, "不支持的 env 输出格式 %q\n", *format)
		return 2
	}
	return 0
}

func runExec(args []string) int {
	fs := flag.NewFlagSet("exec", flag.ContinueOnError)
	cfgPath := fs.String("config", config.DefaultClientPath(), "客户端配置文件路径")
	localAddr := fs.String("local", "", "本地 SOCKS5 监听地址")
	httpAddr := fs.String("http", "", "本地 HTTP 代理监听地址")
	noProxy := fs.String("no-proxy", "localhost,127.0.0.1,::1", "NO_PROXY/no_proxy 值")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	commandArgs := fs.Args()
	if len(commandArgs) == 0 {
		fmt.Fprintln(os.Stderr, "缺少要执行的命令，例如：mingsui exec -- curl https://example.com")
		return 2
	}

	cfg, err := loadClientOrDefault(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		return 1
	}
	applyClientOverrides(&cfg, *localAddr, *httpAddr, "", "", false, "", "")
	vars := proxyEnv(cfg, *noProxy)
	if len(vars) == 0 {
		fmt.Fprintln(os.Stderr, "配置中没有可用的本地代理监听地址")
		return 1
	}

	cmd := exec.Command(commandArgs[0], commandArgs[1:]...)
	cmd.Env = mergeEnv(os.Environ(), vars)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "执行命令失败: %v\n", err)
		return 1
	}
	return 0
}

func runSystemProxy(args []string) int {
	if len(args) == 0 {
		printSystemProxyUsage()
		return 2
	}
	switch args[0] {
	case "enable", "on":
		return enableSystemProxy(args[1:])
	case "disable", "off":
		return disableSystemProxy(args[1:])
	case "status":
		return statusSystemProxy(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "未知系统代理命令 %q\n\n", args[0])
		printSystemProxyUsage()
		return 2
	}
}

func enableSystemProxy(args []string) int {
	fs := flag.NewFlagSet("system-proxy enable", flag.ContinueOnError)
	cfgPath := fs.String("config", config.DefaultClientPath(), "客户端配置文件路径")
	localAddr := fs.String("local", "", "本地 SOCKS5 监听地址")
	httpAddr := fs.String("http", "", "本地 HTTP 代理监听地址")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := loadClientOrDefault(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		return 1
	}
	applyClientOverrides(&cfg, *localAddr, *httpAddr, "", "", false, "", "")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := systemproxy.Enable(ctx, systemProxyConfig(cfg)); err != nil {
		fmt.Fprintf(os.Stderr, "开启系统代理失败: %v\n", err)
		return 1
	}
	fmt.Fprintln(os.Stdout, "系统代理已开启")
	return 0
}

func disableSystemProxy(args []string) int {
	fs := flag.NewFlagSet("system-proxy disable", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := systemproxy.Disable(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "关闭系统代理失败: %v\n", err)
		return 1
	}
	fmt.Fprintln(os.Stdout, "系统代理已关闭")
	return 0
}

func statusSystemProxy(args []string) int {
	fs := flag.NewFlagSet("system-proxy status", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", true, "以 JSON 格式输出状态，传 -json=false 可输出文本")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	status := systemproxy.CurrentStatus(ctx)
	if *jsonOutput {
		if err := writeJSON(os.Stdout, status); err != nil {
			fmt.Fprintf(os.Stderr, "输出系统代理状态失败: %v\n", err)
			return 1
		}
		return 0
	}
	if !status.Supported {
		fmt.Fprintf(os.Stdout, "不支持: %s\n", status.Message)
		return 0
	}
	if status.Enabled {
		fmt.Fprintln(os.Stdout, "系统代理已开启")
	} else {
		fmt.Fprintln(os.Stdout, "系统代理未开启")
	}
	return 0
}

func systemProxyConfig(cfg config.ClientConfig) systemproxy.Config {
	return systemproxy.Config{
		HTTPAddr:  cfg.HTTPAddr,
		SOCKSAddr: cfg.LocalAddr,
	}
}

func proxyEnv(cfg config.ClientConfig, noProxy string) []proxyEnvVar {
	vars := make([]proxyEnvVar, 0, 10)
	localAddr := strings.TrimSpace(cfg.LocalAddr)
	httpAddr := strings.TrimSpace(cfg.HTTPAddr)
	standardProxy := ""
	if httpAddr != "" {
		standardProxy = "http://" + httpAddr
	} else if localAddr != "" {
		standardProxy = "socks5h://" + localAddr
	}
	if standardProxy != "" {
		vars = append(vars,
			proxyEnvVar{Name: "HTTP_PROXY", Value: standardProxy},
			proxyEnvVar{Name: "HTTPS_PROXY", Value: standardProxy},
			proxyEnvVar{Name: "http_proxy", Value: standardProxy},
			proxyEnvVar{Name: "https_proxy", Value: standardProxy},
		)
	}
	if localAddr != "" {
		socksProxy := "socks5h://" + localAddr
		vars = append(vars,
			proxyEnvVar{Name: "ALL_PROXY", Value: socksProxy},
			proxyEnvVar{Name: "all_proxy", Value: socksProxy},
			proxyEnvVar{Name: "MINGSUI_SOCKS5_PROXY", Value: socksProxy},
		)
	}
	if value := strings.TrimSpace(noProxy); value != "" {
		vars = append(vars,
			proxyEnvVar{Name: "NO_PROXY", Value: value},
			proxyEnvVar{Name: "no_proxy", Value: value},
		)
	}
	if httpAddr != "" {
		vars = append(vars, proxyEnvVar{Name: "MINGSUI_HTTP_PROXY", Value: "http://" + httpAddr})
	}
	return vars
}

func mergeEnv(base []string, vars []proxyEnvVar) []string {
	replacements := make(map[string]string, len(vars))
	for _, item := range vars {
		replacements[item.Name] = item.Value
	}

	merged := make([]string, 0, len(base)+len(vars))
	for _, item := range base {
		name, _, ok := strings.Cut(item, "=")
		if !ok {
			merged = append(merged, item)
			continue
		}
		value, exists := replacements[name]
		if !exists {
			merged = append(merged, item)
			continue
		}
		merged = append(merged, name+"="+value)
		delete(replacements, name)
	}
	for _, item := range vars {
		if _, exists := replacements[item.Name]; !exists {
			continue
		}
		merged = append(merged, item.Name+"="+item.Value)
		delete(replacements, item.Name)
	}
	return merged
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func resolveClientProfile(cfg config.ClientConfig, profileName string, autoProfile bool) (config.ClientConfig, string, error) {
	name := strings.TrimSpace(profileName)
	if name == "" && strings.TrimSpace(cfg.ActiveProfile) == "" && autoProfile && len(cfg.Profiles) > 0 {
		name = cfg.Profiles[0].Name
	}
	resolved, err := cfg.ResolveProfile(name)
	if err != nil {
		return config.ClientConfig{}, "", err
	}
	return resolved, strings.TrimSpace(resolved.ActiveProfile), nil
}

func resolveClientProxyProfile(cfg config.ClientConfig, autoProfile bool) (config.ProxyProfile, bool) {
	name := strings.TrimSpace(cfg.ActiveProxyProfile)
	if name == "" && strings.TrimSpace(cfg.ActiveProfile) == "" && autoProfile && len(cfg.ProxyProfiles) > 0 {
		name = cfg.ProxyProfiles[0].Name
	}
	if name == "" {
		return config.ProxyProfile{}, false
	}
	return cfg.ProxyProfile(name)
}

func applyClientOverrides(cfg *config.ClientConfig, localAddr, httpAddr, relayAddr, token string, authEnabled bool, authUser, authPass string) {
	if localAddr != "" {
		cfg.LocalAddr = localAddr
	}
	if httpAddr != "" {
		cfg.HTTPAddr = httpAddr
	}
	if relayAddr != "" {
		cfg.RelayAddr = relayAddr
	}
	if token != "" {
		cfg.Token = token
	}
	if authEnabled {
		cfg.LocalAuth.Enabled = true
	}
	if authUser != "" {
		cfg.LocalAuth.Enabled = true
		cfg.LocalAuth.Username = authUser
	}
	if authPass != "" {
		cfg.LocalAuth.Enabled = true
		cfg.LocalAuth.Password = authPass
	}
}

func checkListen(name, label, addr string) diagnostic.Check {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return diagnostic.Check{
			Name:   name,
			Label:  label,
			Target: addr,
			OK:     false,
			Error:  err.Error(),
		}
	}
	_ = listener.Close()
	return diagnostic.Check{
		Name:   name,
		Label:  label,
		Target: addr,
		OK:     true,
	}
}

func printListenResult(check diagnostic.Check) {
	if check.OK {
		fmt.Fprintf(os.Stdout, "[正常] %s可用: %s\n", check.Label, check.Target)
		return
	}
	fmt.Fprintf(os.Stdout, "[失败] %s不可用: %s (%s)\n", check.Label, check.Target, check.Error)
}

func printRelayHealthResult(check diagnostic.Check) {
	if check.OK {
		fmt.Fprintf(os.Stdout, "[正常] %s: %s\n", check.Label, check.Target)
		printRelayMetrics(check.Metrics)
		return
	}
	fmt.Fprintf(os.Stdout, "[失败] %s: %s\n", check.Label, check.Error)
}

func printRelayMetrics(metrics *protocol.Metrics) {
	if metrics == nil {
		return
	}
	fmt.Fprintf(os.Stdout, "  relay 活跃连接: %d\n", metrics.ActiveConnections)
	fmt.Fprintf(os.Stdout, "  relay 累计连接: %d\n", metrics.TotalConnections)
	fmt.Fprintf(os.Stdout, "  relay 累计上行: %d B\n", metrics.UploadBytes)
	fmt.Fprintf(os.Stdout, "  relay 累计下行: %d B\n", metrics.DownloadBytes)
}

func writeDiagnosticReport(report diagnostic.Report) int {
	if err := diagnostic.WriteJSON(os.Stdout, report); err != nil {
		fmt.Fprintf(os.Stderr, "输出 JSON 诊断结果失败: %v\n", err)
		return 1
	}
	return report.ExitCode()
}

func localProxyMayBeExposed(cfg config.ClientConfig) bool {
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

func runConfig(args []string) int {
	if len(args) == 0 {
		printConfigUsage()
		return 2
	}

	switch args[0] {
	case "init":
		return initClientConfig(args[1:])
	case "path":
		fmt.Println(config.DefaultClientPath())
		return 0
	case "show":
		return showClientConfig(args[1:])
	case "profile":
		return runConfigProfile(args[1:])
	case "subscription":
		return runConfigSubscription(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "未知配置命令 %q\n\n", args[0])
		printConfigUsage()
		return 2
	}
}

func runConfigProfile(args []string) int {
	if len(args) == 0 {
		printConfigProfileUsage()
		return 2
	}

	switch args[0] {
	case "list":
		return listClientProfiles(args[1:])
	case "add":
		return addClientProfile(args[1:])
	case "select":
		return selectClientProfile(args[1:])
	case "remove":
		return removeClientProfile(args[1:])
	case "rename":
		return renameClientProfile(args[1:])
	case "check":
		return checkClientProfile(args[1:])
	case "import":
		return importClientProfiles(args[1:])
	case "export":
		return exportClientProfiles(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "未知 profile 命令 %q\n\n", args[0])
		printConfigProfileUsage()
		return 2
	}
}

func listClientProfiles(args []string) int {
	fs := flag.NewFlagSet("config profile list", flag.ContinueOnError)
	cfgPath := fs.String("path", config.DefaultClientPath(), "客户端配置文件路径")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := config.LoadClient(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		return 1
	}
	if len(cfg.Profiles) == 0 {
		fmt.Fprintln(os.Stdout, "没有 relay profile")
		return 0
	}
	for _, profile := range cfg.Profiles {
		marker := " "
		if profile.Name == cfg.ActiveProfile {
			marker = "*"
		}
		fmt.Fprintf(os.Stdout, "%s %s %s\n", marker, profile.Name, profile.RelayAddr)
	}
	return 0
}

func addClientProfile(args []string) int {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		fmt.Fprintln(os.Stderr, "profile 名称不能为空")
		return 2
	}
	name := strings.TrimSpace(args[0])

	fs := flag.NewFlagSet("config profile add", flag.ContinueOnError)
	cfgPath := fs.String("path", config.DefaultClientPath(), "客户端配置文件路径")
	relayAddr := fs.String("relay", "", "relay 服务端地址")
	token := fs.String("token", "", "客户端和 relay 共享的 token")
	tlsEnabled := fs.Bool("tls", false, "为该 profile 启用 relay TLS")
	tlsServerName := fs.String("server-name", "", "relay TLS ServerName")
	tlsCAFile := fs.String("ca-file", "", "relay TLS CA 文件")
	tlsInsecure := fs.Bool("insecure-skip-verify", false, "跳过 relay TLS 证书校验")
	force := fs.Bool("force", false, "覆盖同名 profile")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}

	cfg, err := loadClientOrDefault(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		return 1
	}
	if err := cfg.UpsertRelayProfile(config.RelayProfile{
		Name:      name,
		RelayAddr: *relayAddr,
		Token:     *token,
		TLS: config.ClientTLSConfig{
			Enabled:            *tlsEnabled,
			ServerName:         *tlsServerName,
			CAFile:             *tlsCAFile,
			InsecureSkipVerify: *tlsInsecure,
		},
	}, *force); err != nil {
		fmt.Fprintf(os.Stderr, "写入 profile 失败: %v\n", err)
		return 1
	}
	if err := config.WriteClient(*cfgPath, cfg, true); err != nil {
		fmt.Fprintf(os.Stderr, "写入配置失败: %v\n", err)
		return 1
	}
	fmt.Printf("已写入 profile %s\n", name)
	return 0
}

func selectClientProfile(args []string) int {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		fmt.Fprintln(os.Stderr, "profile 名称不能为空")
		return 2
	}
	name := strings.TrimSpace(args[0])

	fs := flag.NewFlagSet("config profile select", flag.ContinueOnError)
	cfgPath := fs.String("path", config.DefaultClientPath(), "客户端配置文件路径")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}

	cfg, err := config.LoadClient(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		return 1
	}
	if err := cfg.SelectRelayProfile(name); err != nil {
		fmt.Fprintf(os.Stderr, "选择 profile 失败: %v\n", err)
		return 1
	}
	if err := config.WriteClient(*cfgPath, cfg, true); err != nil {
		fmt.Fprintf(os.Stderr, "写入配置失败: %v\n", err)
		return 1
	}
	fmt.Printf("已选择 profile %s\n", name)
	return 0
}

func removeClientProfile(args []string) int {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		fmt.Fprintln(os.Stderr, "profile 名称不能为空")
		return 2
	}
	name := strings.TrimSpace(args[0])

	fs := flag.NewFlagSet("config profile remove", flag.ContinueOnError)
	cfgPath := fs.String("path", config.DefaultClientPath(), "客户端配置文件路径")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}

	cfg, err := config.LoadClient(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		return 1
	}
	if err := cfg.RemoveRelayProfile(name); err != nil {
		fmt.Fprintf(os.Stderr, "删除 profile 失败: %v\n", err)
		return 1
	}
	if err := config.WriteClient(*cfgPath, cfg, true); err != nil {
		fmt.Fprintf(os.Stderr, "写入配置失败: %v\n", err)
		return 1
	}
	fmt.Printf("已删除 profile %s\n", name)
	return 0
}

func renameClientProfile(args []string) int {
	if len(args) < 2 || strings.TrimSpace(args[0]) == "" || strings.TrimSpace(args[1]) == "" {
		fmt.Fprintln(os.Stderr, "profile 旧名称和新名称不能为空")
		return 2
	}
	oldName := strings.TrimSpace(args[0])
	newName := strings.TrimSpace(args[1])

	fs := flag.NewFlagSet("config profile rename", flag.ContinueOnError)
	cfgPath := fs.String("path", config.DefaultClientPath(), "客户端配置文件路径")
	if err := fs.Parse(args[2:]); err != nil {
		return 2
	}

	cfg, err := config.LoadClient(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		return 1
	}
	if err := cfg.RenameRelayProfile(oldName, newName); err != nil {
		fmt.Fprintf(os.Stderr, "重命名 profile 失败: %v\n", err)
		return 1
	}
	if err := config.WriteClient(*cfgPath, cfg, true); err != nil {
		fmt.Fprintf(os.Stderr, "写入配置失败: %v\n", err)
		return 1
	}
	fmt.Printf("已重命名 profile %s -> %s\n", oldName, newName)
	return 0
}

func checkClientProfile(args []string) int {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		fmt.Fprintln(os.Stderr, "profile 名称不能为空")
		return 2
	}
	name := strings.TrimSpace(args[0])

	fs := flag.NewFlagSet("config profile check", flag.ContinueOnError)
	cfgPath := fs.String("path", config.DefaultClientPath(), "客户端配置文件路径")
	jsonOutput := fs.Bool("json", false, "以 JSON 格式输出诊断结果")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}

	report := diagnostic.NewReport(*cfgPath)
	cfg, err := config.LoadClient(*cfgPath)
	if err != nil {
		report.Fail(fmt.Sprintf("加载配置失败: %v", err))
		if *jsonOutput {
			return writeDiagnosticReport(report)
		}
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		return 1
	}
	cfg, err = cfg.ResolveProfile(name)
	if err != nil {
		report.Fail(fmt.Sprintf("选择 profile 失败: %v", err))
		if *jsonOutput {
			return writeDiagnosticReport(report)
		}
		fmt.Fprintf(os.Stderr, "选择 profile 失败: %v\n", err)
		return 1
	}

	logger := log.New(io.Discard, "", 0)
	service, err := client.NewService(cfg, logger)
	if err != nil {
		report.Fail(fmt.Sprintf("创建客户端失败: %v", err))
		if *jsonOutput {
			return writeDiagnosticReport(report)
		}
		fmt.Fprintf(os.Stderr, "创建客户端失败: %v\n", err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.DialTimeout()+2*time.Second)
	defer cancel()
	health, err := service.CheckRelayStatus(ctx)
	check := diagnostic.Check{
		Name:   "profile_health",
		Label:  fmt.Sprintf("relay profile %s 健康检查", name),
		Target: cfg.RelayAddr,
		OK:     err == nil,
	}
	if err != nil {
		check.Error = err.Error()
	} else {
		check.Metrics = health.Metrics
	}
	report.AddCheck(check)

	if *jsonOutput {
		return writeDiagnosticReport(report)
	}
	printRelayHealthResult(check)
	return report.ExitCode()
}

func initClientConfig(args []string) int {
	fs := flag.NewFlagSet("config init", flag.ContinueOnError)
	cfgPath := fs.String("path", config.DefaultClientPath(), "客户端配置文件路径")
	force := fs.Bool("force", false, "覆盖已存在的配置文件")
	localAddr := fs.String("local", "127.0.0.1:18080", "本地 SOCKS5 监听地址")
	httpAddr := fs.String("http", "127.0.0.1:18081", "本地 HTTP 代理监听地址")
	relayAddr := fs.String("relay", "127.0.0.1:9443", "relay 服务端地址")
	token := fs.String("token", "change-me", "客户端和 relay 共享的 token")
	authEnabled := fs.Bool("auth", false, "启用本地代理认证")
	authUser := fs.String("auth-user", "", "本地代理认证用户名")
	authPass := fs.String("auth-pass", "", "本地代理认证密码")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg := config.DefaultClient()
	cfg.LocalAddr = *localAddr
	cfg.HTTPAddr = *httpAddr
	cfg.RelayAddr = *relayAddr
	cfg.Token = *token
	cfg.LocalAuth.Enabled = *authEnabled || *authUser != "" || *authPass != ""
	cfg.LocalAuth.Username = *authUser
	cfg.LocalAuth.Password = *authPass

	if err := config.WriteClient(*cfgPath, cfg, *force); err != nil {
		fmt.Fprintf(os.Stderr, "写入配置失败: %v\n", err)
		return 1
	}
	fmt.Printf("已写入 %s\n", *cfgPath)
	return 0
}

func showClientConfig(args []string) int {
	fs := flag.NewFlagSet("config show", flag.ContinueOnError)
	cfgPath := fs.String("path", config.DefaultClientPath(), "客户端配置文件路径")
	showSecrets := fs.Bool("secrets", false, "显示真实 token 和本地代理密码")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := config.LoadClient(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		return 1
	}
	if !*showSecrets {
		cfg = cfg.Redacted()
	}
	if err := writeJSON(os.Stdout, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "输出配置失败: %v\n", err)
		return 1
	}
	return 0
}

func loadClientOrDefault(path string) (config.ClientConfig, error) {
	cfg, err := config.LoadClient(path)
	if err == nil {
		return cfg, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return config.DefaultClient(), nil
	}
	return config.ClientConfig{}, err
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `明隧客户端

用法:
  mingsui import -source <file|url|-> [flags]
  mingsui connect [flags]
  mingsui status [flags]
  mingsui disconnect [flags]
  mingsui env [flags]
  mingsui exec [flags] -- <command> [args...]
  mingsui system-proxy enable|disable|status [flags]
  mingsui kernel export [flags]
  mingsui run [flags]
  mingsui doctor [flags]
  mingsui config init [flags]
  mingsui config path
  mingsui config profile add|list|select|remove|rename|check|import|export [flags]
  mingsui config subscription add|list|remove|sync [flags]
  mingsui config show [flags]
  mingsui token [flags]
  mingsui version

示例:
  TOKEN=$(mingsui token)
  mingsui import -source ./nodes.json
  mingsui connect
  mingsui status
  eval "$(mingsui env)"
  mingsui exec -- curl https://example.com
  mingsui system-proxy enable
  mingsui kernel export -config %s -output /tmp/mingsui-mihomo.yaml
  mingsui config init -relay example.com:9443 -token "$TOKEN"
  mingsui config profile add tokyo -relay tokyo.example.com:9443 -token "$TOKEN"
  mingsui config profile check tokyo
  mingsui config profile import -source ./nodes.json -force
  mingsui config profile export -output ./nodes.json -secrets
  mingsui config subscription add team -url https://example.com/mingsui/nodes.json
  mingsui config subscription sync team
  mingsui config profile rename tokyo jp-tokyo
  mingsui run -profile tokyo -config %s
  mingsui config init -local 0.0.0.0:18080 -auth-user user -auth-pass pass -relay example.com:9443 -token "$TOKEN"
  mingsui config show -path %s
  mingsui doctor -config %s
  mingsui doctor -json -config %s
  mingsui run -config %s
  curl --socks5-hostname 127.0.0.1:18080 https://example.com
  curl -x http://127.0.0.1:18081 https://example.com

`, config.DefaultClientPath(), config.DefaultClientPath(), config.DefaultClientPath(), config.DefaultClientPath(), config.DefaultClientPath(), config.DefaultClientPath())
}

func printKernelUsage() {
	fmt.Fprintln(os.Stderr, `用法:
  mingsui kernel export [flags]`)
}

func printSystemProxyUsage() {
	fmt.Fprintln(os.Stderr, `用法:
  mingsui system-proxy enable [flags]
  mingsui system-proxy disable
  mingsui system-proxy status [flags]`)
}

func printConfigUsage() {
	fmt.Fprintln(os.Stderr, `用法:
  mingsui config init [flags]
  mingsui config path
  mingsui config profile add|list|select|remove|rename|check|import|export [flags]
  mingsui config subscription add|list|remove|sync [flags]
  mingsui config show [flags]`)
}

func printConfigProfileUsage() {
	fmt.Fprintln(os.Stderr, `用法:
  mingsui config profile list [flags]
  mingsui config profile add <name> -relay <addr> -token <token> [flags]
  mingsui config profile select <name> [flags]
  mingsui config profile remove <name> [flags]
  mingsui config profile rename <old-name> <new-name> [flags]
  mingsui config profile check <name> [flags]
  mingsui config profile import -source <file|url|-> [flags]
  mingsui config profile export [flags] [name...]`)
}

func writeJSON(w io.Writer, value any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}
