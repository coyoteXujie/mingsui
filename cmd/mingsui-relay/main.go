package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
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
	"github.com/coyoteXujie/mingsui/internal/relay"
	"github.com/coyoteXujie/mingsui/internal/security"
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

	failed := false
	fmt.Fprintf(os.Stdout, "配置文件: %s\n", *cfgPath)
	if cfg.Token == "change-me" {
		fmt.Fprintln(os.Stdout, "警告: 当前使用默认 token，生产环境必须修改")
	}
	if cfg.TLS.Enabled {
		if _, err := tls.LoadX509KeyPair(cfg.TLS.CertFile, cfg.TLS.KeyFile); err != nil {
			fmt.Fprintf(os.Stdout, "[失败] TLS 证书加载失败: %v\n", err)
			failed = true
		} else {
			fmt.Fprintf(os.Stdout, "[正常] TLS 证书可以加载\n")
		}
	} else {
		fmt.Fprintf(os.Stdout, "[正常] TLS 未启用\n")
	}
	if !*skipListen {
		if !printListenCheck("relay 监听地址", cfg.ListenAddr) {
			failed = true
		}
	}

	if failed {
		return 1
	}
	return 0
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

func printListenCheck(label, addr string) bool {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintf(os.Stdout, "[失败] %s不可用: %s (%v)\n", label, addr, err)
		return false
	}
	_ = listener.Close()
	fmt.Fprintf(os.Stdout, "[正常] %s可用: %s\n", label, addr)
	return true
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
  mingsui-relay systemd [flags]
  mingsui-relay token [flags]
  mingsui-relay version

示例:
  TOKEN=$(mingsui-relay token)
  mingsui-relay cert -host example.com,127.0.0.1 -cert relay.crt -key relay.key
  mingsui-relay config init -listen 0.0.0.0:9443 -token "$TOKEN"
  mingsui-relay systemd -output mingsui-relay.service
  mingsui-relay check -config %s
  mingsui-relay serve -config %s

`, config.DefaultRelayPath(), config.DefaultRelayPath())
}

func printConfigUsage() {
	fmt.Fprintln(os.Stderr, `用法:
  mingsui-relay config init [flags]
  mingsui-relay config path`)
}
