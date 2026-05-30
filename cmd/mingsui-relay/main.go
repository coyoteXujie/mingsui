package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/coyoteXujie/mingsui/internal/buildinfo"
	"github.com/coyoteXujie/mingsui/internal/config"
	"github.com/coyoteXujie/mingsui/internal/deploy"
	"github.com/coyoteXujie/mingsui/internal/diagnostic"
	"github.com/coyoteXujie/mingsui/internal/relay"
	"github.com/coyoteXujie/mingsui/internal/security"
)

const certificateExpiryWarning = 30 * 24 * time.Hour

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 2
	}

	switch args[0] {
	case "serve":
		return runRelay(args[1:])
	case "check":
		return runCheck(args[1:])
	case "cert":
		return runCert(args[1:])
	case "config":
		return runConfig(args[1:])
	case "systemd":
		return runSystemd(args[1:])
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

func runSystemd(args []string) int {
	fs := flag.NewFlagSet("systemd", flag.ContinueOnError)
	outputPath := fs.String("output", "", "输出文件路径，留空则打印到 stdout")
	binaryPath := fs.String("binary", "/usr/local/bin/mingsui-relay", "mingsui-relay 二进制路径")
	cfgPath := fs.String("config", "/etc/mingsui/relay.json", "relay 配置文件路径")
	user := fs.String("user", "mingsui", "systemd 服务运行用户")
	group := fs.String("group", "", "systemd 服务运行用户组，留空则与 user 相同")
	workDir := fs.String("workdir", "/var/lib/mingsui", "服务工作目录")
	description := fs.String("description", "MingSui Relay", "systemd 服务描述")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	unit, err := deploy.RenderRelaySystemd(deploy.SystemdRelayOptions{
		Description: *description,
		BinaryPath:  *binaryPath,
		ConfigPath:  *cfgPath,
		User:        *user,
		Group:       *group,
		WorkingDir:  *workDir,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "生成 systemd 服务失败: %v\n", err)
		return 1
	}

	if *outputPath == "" {
		fmt.Print(unit)
		return 0
	}
	if err := os.WriteFile(*outputPath, []byte(unit), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "写入 systemd 服务失败: %v\n", err)
		return 1
	}
	fmt.Printf("已写入 %s\n", *outputPath)
	return 0
}

func runCert(args []string) int {
	fs := flag.NewFlagSet("cert", flag.ContinueOnError)
	certPath := fs.String("cert", "relay.crt", "证书输出路径")
	keyPath := fs.String("key", "relay.key", "私钥输出路径")
	hosts := fs.String("host", "localhost,127.0.0.1", "证书主机名或 IP，多个值用英文逗号分隔")
	days := fs.Int("days", 365, "证书有效天数")
	keyBits := fs.Int("rsa-bits", 2048, "RSA 密钥长度")
	force := fs.Bool("force", false, "覆盖已存在的证书和私钥")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *days <= 0 {
		fmt.Fprintln(os.Stderr, "证书有效天数必须大于 0")
		return 1
	}

	certPEM, keyPEM, err := security.GenerateSelfSignedCertificate(security.CertificateOptions{
		Hosts:      strings.Split(*hosts, ","),
		ValidFor:   time.Duration(*days) * 24 * time.Hour,
		RSAKeyBits: *keyBits,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "生成证书失败: %v\n", err)
		return 1
	}
	if err := security.WriteCertificateFiles(*certPath, *keyPath, certPEM, keyPEM, *force); err != nil {
		fmt.Fprintf(os.Stderr, "写入证书失败: %v\n", err)
		return 1
	}

	fmt.Printf("已写入证书: %s\n", *certPath)
	fmt.Printf("已写入私钥: %s\n", *keyPath)
	fmt.Println("relay 配置中设置 tls.enabled=true，并填写 tls.cert_file / tls.key_file。")
	fmt.Println("客户端可以把 tls.ca_file 指向这个证书文件。")
	return 0
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

func runCheck(args []string) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	cfgPath := fs.String("config", config.DefaultRelayPath(), "relay 配置文件路径")
	listenAddr := fs.String("listen", "", "relay 监听地址")
	token := fs.String("token", "", "客户端和 relay 共享的 token")
	allowPrivate := fs.Bool("allow-private", false, "允许 relay 访问私有和本地目标网络")
	skipListen := fs.Bool("skip-listen", false, "跳过监听端口可用性检查")
	jsonOutput := fs.Bool("json", false, "以 JSON 格式输出诊断结果")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	report := diagnostic.NewReport(*cfgPath)
	cfg, err := loadRelayOrDefault(*cfgPath)
	if err != nil {
		report.Fail(fmt.Sprintf("加载配置失败: %v", err))
		if *jsonOutput {
			return writeDiagnosticReport(report)
		}
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		return 1
	}
	applyRelayOverrides(&cfg, *listenAddr, *token, *allowPrivate)
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
	tlsCheck := checkTLS(cfg.TLS, time.Now())
	report.AddCheck(tlsCheck)
	if !*jsonOutput {
		printTLSResult(tlsCheck)
	}
	if !*skipListen {
		check := checkListen("relay_listen", "relay 监听地址", cfg.ListenAddr)
		report.AddCheck(check)
		if !*jsonOutput {
			printListenResult(check)
		}
	}

	if *jsonOutput {
		return writeDiagnosticReport(report)
	}
	return report.ExitCode()
}

func printTLSCheck(cfg config.RelayTLSConfig, now time.Time) bool {
	check := checkTLS(cfg, now)
	printTLSResult(check)
	return check.OK
}

func checkTLS(cfg config.RelayTLSConfig, now time.Time) diagnostic.Check {
	check := diagnostic.Check{
		Name:  "tls",
		Label: "TLS",
		OK:    true,
	}
	if !cfg.Enabled {
		check.Skipped = true
		return check
	}
	check.Target = cfg.CertFile
	if _, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile); err != nil {
		check.OK = false
		check.Error = fmt.Sprintf("TLS 证书加载失败: %v", err)
		return check
	}

	info, err := security.LoadCertificateInfo(cfg.CertFile)
	if err != nil {
		check.OK = false
		check.Error = fmt.Sprintf("TLS 证书解析失败: %v", err)
		return check
	}
	remaining := info.NotAfter.Sub(now)
	check.Certificate = &diagnostic.CertificateSummary{
		Hosts:            info.Hosts(),
		NotBefore:        info.NotBefore.Format(time.RFC3339),
		NotAfter:         info.NotAfter.Format(time.RFC3339),
		ExpiresInSeconds: int64(remaining.Seconds()),
	}
	if now.Before(info.NotBefore) {
		check.OK = false
		check.Error = "TLS 证书尚未生效"
		return check
	}
	if !now.Before(info.NotAfter) {
		check.OK = false
		check.Error = "TLS 证书已过期"
		return check
	}
	if remaining <= certificateExpiryWarning {
		check.Warning = fmt.Sprintf("TLS 证书将在 %s 后过期", formatRemaining(remaining))
	}
	return check
}

func printTLSResult(check diagnostic.Check) {
	if check.Skipped {
		fmt.Fprintf(os.Stdout, "[正常] TLS 未启用\n")
		return
	}
	if !check.OK {
		fmt.Fprintf(os.Stdout, "[失败] %s\n", check.Error)
		return
	}
	fmt.Fprintf(os.Stdout, "[正常] TLS 证书可以加载\n")
	if check.Certificate == nil {
		return
	}
	if len(check.Certificate.Hosts) > 0 {
		fmt.Fprintf(os.Stdout, "  TLS 证书主机: %s\n", strings.Join(check.Certificate.Hosts, ", "))
	}
	fmt.Fprintf(os.Stdout, "  TLS 证书有效期: %s 至 %s\n",
		formatCertificateTimeString(check.Certificate.NotBefore),
		formatCertificateTimeString(check.Certificate.NotAfter),
	)
	if check.Warning != "" {
		fmt.Fprintf(os.Stdout, "[警告] %s\n", check.Warning)
	}
}

func formatCertificateTime(t time.Time) string {
	return t.Local().Format("2006-01-02 15:04:05 MST")
}

func formatCertificateTimeString(value string) string {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return formatCertificateTime(t)
}

func formatRemaining(d time.Duration) string {
	if d < time.Minute {
		return "不足 1 分钟"
	}
	days := int(d.Hours()) / 24
	if days > 0 {
		return fmt.Sprintf("%d 天", days)
	}
	hours := int(d.Hours())
	if hours > 0 {
		return fmt.Sprintf("%d 小时", hours)
	}
	return fmt.Sprintf("%d 分钟", int(d.Minutes()))
}

func runRelay(args []string) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	cfgPath := fs.String("config", config.DefaultRelayPath(), "relay 配置文件路径")
	listenAddr := fs.String("listen", "", "relay 监听地址")
	token := fs.String("token", "", "客户端和 relay 共享的 token")
	allowPrivate := fs.Bool("allow-private", false, "允许 relay 访问私有和本地目标网络")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := loadRelayOrDefault(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		return 1
	}
	applyRelayOverrides(&cfg, *listenAddr, *token, *allowPrivate)
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "配置不正确: %v\n", err)
		return 1
	}
	if cfg.Token == "change-me" {
		fmt.Fprintln(os.Stderr, "警告: 当前使用默认 token，对外暴露 relay 前必须修改")
	}

	logger := log.New(os.Stderr, "mingsui relay: ", log.LstdFlags)
	server, err := relay.NewServer(cfg, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建 relay 失败: %v\n", err)
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := server.Serve(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "运行 relay 失败: %v\n", err)
		return 1
	}
	return 0
}

func applyRelayOverrides(cfg *config.RelayConfig, listenAddr, token string, allowPrivate bool) {
	if listenAddr != "" {
		cfg.ListenAddr = listenAddr
	}
	if token != "" {
		cfg.Token = token
	}
	if allowPrivate {
		cfg.AllowPrivateNetworks = true
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

func writeDiagnosticReport(report diagnostic.Report) int {
	if err := diagnostic.WriteJSON(os.Stdout, report); err != nil {
		fmt.Fprintf(os.Stderr, "输出 JSON 诊断结果失败: %v\n", err)
		return 1
	}
	return report.ExitCode()
}

func runConfig(args []string) int {
	if len(args) == 0 {
		printConfigUsage()
		return 2
	}

	switch args[0] {
	case "init":
		return initRelayConfig(args[1:])
	case "path":
		fmt.Println(config.DefaultRelayPath())
		return 0
	case "show":
		return showRelayConfig(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "未知配置命令 %q\n\n", args[0])
		printConfigUsage()
		return 2
	}
}

func initRelayConfig(args []string) int {
	fs := flag.NewFlagSet("config init", flag.ContinueOnError)
	cfgPath := fs.String("path", config.DefaultRelayPath(), "relay 配置文件路径")
	force := fs.Bool("force", false, "覆盖已存在的配置文件")
	listenAddr := fs.String("listen", "0.0.0.0:9443", "relay 监听地址")
	token := fs.String("token", "auto", "客户端和 relay 共享的 token，默认 auto 自动生成")
	tokenBytes := fs.Int("token-bytes", security.DefaultTokenBytes, "自动生成 token 时使用的随机字节长度")
	allowPrivate := fs.Bool("allow-private", false, "允许 relay 访问私有和本地目标网络")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	finalToken := *token
	generatedToken := false
	if finalToken == "" || finalToken == "auto" {
		generated, err := security.GenerateToken(*tokenBytes)
		if err != nil {
			fmt.Fprintf(os.Stderr, "生成 token 失败: %v\n", err)
			return 1
		}
		finalToken = generated
		generatedToken = true
	}

	cfg := config.DefaultRelay()
	cfg.ListenAddr = *listenAddr
	cfg.Token = finalToken
	cfg.AllowPrivateNetworks = *allowPrivate

	if err := config.WriteRelay(*cfgPath, cfg, *force); err != nil {
		fmt.Fprintf(os.Stderr, "写入配置失败: %v\n", err)
		return 1
	}
	fmt.Printf("已写入 %s\n", *cfgPath)
	if generatedToken {
		fmt.Printf("已自动生成 token: %s\n", finalToken)
		fmt.Println("请把这个 token 写入客户端配置。")
	}
	return 0
}

func showRelayConfig(args []string) int {
	fs := flag.NewFlagSet("config show", flag.ContinueOnError)
	cfgPath := fs.String("path", config.DefaultRelayPath(), "relay 配置文件路径")
	showSecrets := fs.Bool("secrets", false, "显示真实 token")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := config.LoadRelay(*cfgPath)
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

func loadRelayOrDefault(path string) (config.RelayConfig, error) {
	cfg, err := config.LoadRelay(path)
	if err == nil {
		return cfg, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return config.DefaultRelay(), nil
	}
	return config.RelayConfig{}, err
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `明隧 relay 服务端

用法:
  mingsui-relay serve [flags]
  mingsui-relay check [flags]
  mingsui-relay cert [flags]
  mingsui-relay config init [flags]
  mingsui-relay config path
  mingsui-relay config show [flags]
  mingsui-relay systemd [flags]
  mingsui-relay token [flags]
  mingsui-relay version

示例:
  TOKEN=$(mingsui-relay token)
  mingsui-relay cert -host example.com,127.0.0.1 -cert relay.crt -key relay.key
  mingsui-relay config init -listen 0.0.0.0:9443 -token "$TOKEN"
  mingsui-relay config show -path %s
  mingsui-relay systemd -output mingsui-relay.service
  mingsui-relay check -config %s
  mingsui-relay check -json -config %s
  mingsui-relay serve -config %s

`, config.DefaultRelayPath(), config.DefaultRelayPath(), config.DefaultRelayPath(), config.DefaultRelayPath())
}

func printConfigUsage() {
	fmt.Fprintln(os.Stderr, `用法:
  mingsui-relay config init [flags]
  mingsui-relay config path
  mingsui-relay config show [flags]`)
}

func writeJSON(w io.Writer, value any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}
