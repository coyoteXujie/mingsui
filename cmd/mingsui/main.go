package main

import (
	"context"
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
	"github.com/coyoteXujie/mingsui/internal/client"
	"github.com/coyoteXujie/mingsui/internal/config"
	"github.com/coyoteXujie/mingsui/internal/diagnostic"
	"github.com/coyoteXujie/mingsui/internal/protocol"
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

func runDoctor(args []string) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	cfgPath := fs.String("config", config.DefaultClientPath(), "客户端配置文件路径")
	localAddr := fs.String("local", "", "本地 SOCKS5 监听地址")
	httpAddr := fs.String("http", "", "本地 HTTP 代理监听地址")
	relayAddr := fs.String("relay", "", "relay 服务端地址")
	token := fs.String("token", "", "客户端和 relay 共享的 token")
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
	applyClientOverrides(&cfg, *localAddr, *httpAddr, *relayAddr, *token)
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
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	cfgPath := fs.String("config", config.DefaultClientPath(), "客户端配置文件路径")
	localAddr := fs.String("local", "", "本地 SOCKS5 监听地址")
	httpAddr := fs.String("http", "", "本地 HTTP 代理监听地址")
	relayAddr := fs.String("relay", "", "relay 服务端地址")
	token := fs.String("token", "", "客户端和 relay 共享的 token")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := loadClientOrDefault(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		return 1
	}
	applyClientOverrides(&cfg, *localAddr, *httpAddr, *relayAddr, *token)
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "配置不正确: %v\n", err)
		return 1
	}
	if cfg.Token == "change-me" {
		fmt.Fprintln(os.Stderr, "警告: 当前使用默认 token，对外暴露 relay 前必须修改")
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

func applyClientOverrides(cfg *config.ClientConfig, localAddr, httpAddr, relayAddr, token string) {
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
	default:
		fmt.Fprintf(os.Stderr, "未知配置命令 %q\n\n", args[0])
		printConfigUsage()
		return 2
	}
}

func initClientConfig(args []string) int {
	fs := flag.NewFlagSet("config init", flag.ContinueOnError)
	cfgPath := fs.String("path", config.DefaultClientPath(), "客户端配置文件路径")
	force := fs.Bool("force", false, "覆盖已存在的配置文件")
	localAddr := fs.String("local", "127.0.0.1:18080", "本地 SOCKS5 监听地址")
	httpAddr := fs.String("http", "127.0.0.1:18081", "本地 HTTP 代理监听地址")
	relayAddr := fs.String("relay", "127.0.0.1:9443", "relay 服务端地址")
	token := fs.String("token", "change-me", "客户端和 relay 共享的 token")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg := config.DefaultClient()
	cfg.LocalAddr = *localAddr
	cfg.HTTPAddr = *httpAddr
	cfg.RelayAddr = *relayAddr
	cfg.Token = *token

	if err := config.WriteClient(*cfgPath, cfg, *force); err != nil {
		fmt.Fprintf(os.Stderr, "写入配置失败: %v\n", err)
		return 1
	}
	fmt.Printf("已写入 %s\n", *cfgPath)
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
  mingsui run [flags]
  mingsui doctor [flags]
  mingsui config init [flags]
  mingsui config path
  mingsui token [flags]
  mingsui version

示例:
  TOKEN=$(mingsui token)
  mingsui config init -relay example.com:9443 -token "$TOKEN"
  mingsui doctor -config %s
  mingsui doctor -json -config %s
  mingsui run -config %s
  curl --socks5-hostname 127.0.0.1:18080 https://example.com
  curl -x http://127.0.0.1:18081 https://example.com

`, config.DefaultClientPath(), config.DefaultClientPath(), config.DefaultClientPath())
}

func printConfigUsage() {
	fmt.Fprintln(os.Stderr, `用法:
  mingsui config init [flags]
  mingsui config path`)
}
