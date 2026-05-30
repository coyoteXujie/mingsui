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
	cfgPath := fs.String("config", config.DefaultClientPath(), "client config path")
	localAddr := fs.String("local", "", "local socks5 listen address")
	relayAddr := fs.String("relay", "", "relay server address")
	token := fs.String("token", "", "shared relay token")
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
	cfgPath := fs.String("path", config.DefaultClientPath(), "client config path")
	force := fs.Bool("force", false, "overwrite existing config")
	localAddr := fs.String("local", "127.0.0.1:18080", "local socks5 listen address")
	relayAddr := fs.String("relay", "127.0.0.1:9443", "relay server address")
	token := fs.String("token", "change-me", "shared relay token")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg := config.DefaultClient()
	cfg.LocalAddr = *localAddr
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

`, config.DefaultClientPath())
}

func printConfigUsage() {
	fmt.Fprintln(os.Stderr, `用法:
  mingsui config init [flags]
  mingsui config path`)
}
