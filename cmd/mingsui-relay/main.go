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
	"github.com/coyoteXujie/mingsui/internal/config"
	"github.com/coyoteXujie/mingsui/internal/relay"
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
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		return 1
	}
	if *listenAddr != "" {
		cfg.ListenAddr = *listenAddr
	}
	if *token != "" {
		cfg.Token = *token
	}
	if *allowPrivate {
		cfg.AllowPrivateNetworks = true
	}
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "invalid config: %v\n", err)
		return 1
	}
	if cfg.Token == "change-me" {
		fmt.Fprintln(os.Stderr, "warning: using the default token; change it before exposing the relay")
	}

	logger := log.New(os.Stderr, "mingsui relay: ", log.LstdFlags)
	server, err := relay.NewServer(cfg, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create relay: %v\n", err)
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := server.Serve(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "run relay: %v\n", err)
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
		return initRelayConfig(args[1:])
	case "path":
		fmt.Println(config.DefaultRelayPath())
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown config command %q\n\n", args[0])
		printConfigUsage()
		return 2
	}
}

func initRelayConfig(args []string) int {
	fs := flag.NewFlagSet("config init", flag.ContinueOnError)
	cfgPath := fs.String("path", config.DefaultRelayPath(), "relay 配置文件路径")
	force := fs.Bool("force", false, "覆盖已存在的配置文件")
	listenAddr := fs.String("listen", "0.0.0.0:9443", "relay 监听地址")
	token := fs.String("token", "change-me", "客户端和 relay 共享的 token")
	allowPrivate := fs.Bool("allow-private", false, "允许 relay 访问私有和本地目标网络")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg := config.DefaultRelay()
	cfg.ListenAddr = *listenAddr
	cfg.Token = *token
	cfg.AllowPrivateNetworks = *allowPrivate

	if err := config.WriteRelay(*cfgPath, cfg, *force); err != nil {
		fmt.Fprintf(os.Stderr, "write config: %v\n", err)
		return 1
	}
	fmt.Printf("wrote %s\n", *cfgPath)
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
  mingsui-relay config init [flags]
  mingsui-relay config path
  mingsui-relay version

示例:
  mingsui-relay config init -listen 0.0.0.0:9443 -token your-secret
  mingsui-relay serve -config %s

`, config.DefaultRelayPath())
}

func printConfigUsage() {
	fmt.Fprintln(os.Stderr, `用法:
  mingsui-relay config init [flags]
  mingsui-relay config path`)
}
