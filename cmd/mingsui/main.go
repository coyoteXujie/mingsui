package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/coyoteXujie/mingsui/internal/buildinfo"
	"github.com/coyoteXujie/mingsui/internal/client"
	"github.com/coyoteXujie/mingsui/internal/config"
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
	case "config":
		return runConfig(args[1:])
	case "version":
		fmt.Println(buildinfo.String())
		return 0
	case "help", "-h", "--help":
		printUsage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", args[0])
		printUsage()
		return 2
	}
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
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		return 1
	}
	if *localAddr != "" {
		cfg.LocalAddr = *localAddr
	}
	if *httpAddr != "" {
		cfg.HTTPAddr = *httpAddr
	}
	if *relayAddr != "" {
		cfg.RelayAddr = *relayAddr
	}
	if *token != "" {
		cfg.Token = *token
	}
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "invalid config: %v\n", err)
		return 1
	}
	if cfg.Token == "change-me" {
		fmt.Fprintln(os.Stderr, "warning: using the default token; change it before exposing the relay")
	}

	logger := log.New(os.Stderr, "mingsui client: ", log.LstdFlags)
	service, err := client.NewService(cfg, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create client: %v\n", err)
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := service.Serve(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "run client: %v\n", err)
		return 1
	}
	return 0
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
		fmt.Fprintf(os.Stderr, "unknown config command %q\n\n", args[0])
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
		fmt.Fprintf(os.Stderr, "write config: %v\n", err)
		return 1
	}
	fmt.Printf("wrote %s\n", *cfgPath)
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
  mingsui config init [flags]
  mingsui config path
  mingsui version

示例:
  mingsui config init -relay example.com:9443 -token your-secret
  mingsui run -config %s
  curl --socks5-hostname 127.0.0.1:18080 https://example.com
  curl -x http://127.0.0.1:18081 https://example.com

`, config.DefaultClientPath())
}

func printConfigUsage() {
	fmt.Fprintln(os.Stderr, `用法:
  mingsui config init [flags]
  mingsui config path`)
}
