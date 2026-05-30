package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/coyoteXujie/mingsui/internal/config"
	"github.com/coyoteXujie/mingsui/internal/subscription"
)

func importClientProfiles(args []string) int {
	fs := flag.NewFlagSet("config profile import", flag.ContinueOnError)
	cfgPath := fs.String("path", config.DefaultClientPath(), "客户端配置文件路径")
	source := fs.String("source", "", "订阅来源：本地 JSON 文件、HTTP(S) URL 或 - 表示 stdin")
	force := fs.Bool("force", false, "覆盖同名 profile")
	selectName := fs.String("select", "", "导入后选择指定 profile")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*source) == "" {
		fmt.Fprintln(os.Stderr, "订阅来源不能为空")
		return 2
	}

	cfg, err := loadClientOrDefault(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		return 1
	}
	profiles, err := loadProfilesFromSource(*source, os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取订阅失败: %v\n", err)
		return 1
	}
	if err := cfg.ImportRelayProfiles(profiles, *force); err != nil {
		fmt.Fprintf(os.Stderr, "导入 profile 失败: %v\n", err)
		return 1
	}
	if strings.TrimSpace(*selectName) != "" {
		if err := cfg.SelectRelayProfile(*selectName); err != nil {
			fmt.Fprintf(os.Stderr, "选择 profile 失败: %v\n", err)
			return 1
		}
	}
	if err := config.WriteClient(*cfgPath, cfg, true); err != nil {
		fmt.Fprintf(os.Stderr, "写入配置失败: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stdout, "已导入 %d 个 profile\n", len(profiles))
	return 0
}

func runConfigSubscription(args []string) int {
	if len(args) == 0 {
		printConfigSubscriptionUsage()
		return 2
	}

	switch args[0] {
	case "list":
		return listClientSubscriptions(args[1:])
	case "add":
		return addClientSubscription(args[1:])
	case "remove":
		return removeClientSubscription(args[1:])
	case "sync":
		return syncClientSubscription(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "未知 subscription 命令 %q\n\n", args[0])
		printConfigSubscriptionUsage()
		return 2
	}
}

func listClientSubscriptions(args []string) int {
	fs := flag.NewFlagSet("config subscription list", flag.ContinueOnError)
	cfgPath := fs.String("path", config.DefaultClientPath(), "客户端配置文件路径")
	showSecrets := fs.Bool("secrets", false, "显示真实订阅 URL")
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
	if len(cfg.Subscriptions) == 0 {
		fmt.Fprintln(os.Stdout, "没有 relay 订阅")
		return 0
	}
	for _, sub := range cfg.Subscriptions {
		fmt.Fprintf(os.Stdout, "%s %s\n", sub.Name, sub.URL)
	}
	return 0
}

func addClientSubscription(args []string) int {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		fmt.Fprintln(os.Stderr, "订阅名称不能为空")
		return 2
	}
	name := strings.TrimSpace(args[0])

	fs := flag.NewFlagSet("config subscription add", flag.ContinueOnError)
	cfgPath := fs.String("path", config.DefaultClientPath(), "客户端配置文件路径")
	url := fs.String("url", "", "订阅 URL")
	force := fs.Bool("force", false, "覆盖同名订阅")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}

	cfg, err := loadClientOrDefault(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		return 1
	}
	if err := cfg.UpsertRelaySubscription(config.RelaySubscription{Name: name, URL: *url}, *force); err != nil {
		fmt.Fprintf(os.Stderr, "写入订阅失败: %v\n", err)
		return 1
	}
	if err := config.WriteClient(*cfgPath, cfg, true); err != nil {
		fmt.Fprintf(os.Stderr, "写入配置失败: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stdout, "已写入订阅 %s\n", name)
	return 0
}

func removeClientSubscription(args []string) int {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		fmt.Fprintln(os.Stderr, "订阅名称不能为空")
		return 2
	}
	name := strings.TrimSpace(args[0])

	fs := flag.NewFlagSet("config subscription remove", flag.ContinueOnError)
	cfgPath := fs.String("path", config.DefaultClientPath(), "客户端配置文件路径")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}

	cfg, err := config.LoadClient(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		return 1
	}
	if err := cfg.RemoveRelaySubscription(name); err != nil {
		fmt.Fprintf(os.Stderr, "删除订阅失败: %v\n", err)
		return 1
	}
	if err := config.WriteClient(*cfgPath, cfg, true); err != nil {
		fmt.Fprintf(os.Stderr, "写入配置失败: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stdout, "已删除订阅 %s\n", name)
	return 0
}

func syncClientSubscription(args []string) int {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		fmt.Fprintln(os.Stderr, "订阅名称不能为空")
		return 2
	}
	name := strings.TrimSpace(args[0])

	fs := flag.NewFlagSet("config subscription sync", flag.ContinueOnError)
	cfgPath := fs.String("path", config.DefaultClientPath(), "客户端配置文件路径")
	force := fs.Bool("force", true, "覆盖同名 profile")
	selectName := fs.String("select", "", "同步后选择指定 profile")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}

	cfg, err := config.LoadClient(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		return 1
	}
	sub, ok := cfg.RelaySubscription(name)
	if !ok {
		fmt.Fprintf(os.Stderr, "订阅 %q 不存在\n", name)
		return 1
	}
	profiles, err := loadProfilesFromSource(sub.URL, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "同步订阅失败: %v\n", err)
		return 1
	}
	if err := cfg.ImportRelayProfiles(profiles, *force); err != nil {
		fmt.Fprintf(os.Stderr, "导入 profile 失败: %v\n", err)
		return 1
	}
	if strings.TrimSpace(*selectName) != "" {
		if err := cfg.SelectRelayProfile(*selectName); err != nil {
			fmt.Fprintf(os.Stderr, "选择 profile 失败: %v\n", err)
			return 1
		}
	}
	if err := config.WriteClient(*cfgPath, cfg, true); err != nil {
		fmt.Fprintf(os.Stderr, "写入配置失败: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stdout, "已同步订阅 %s，导入 %d 个 profile\n", name, len(profiles))
	return 0
}

func loadProfilesFromSource(source string, stdin io.Reader) ([]config.RelayProfile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return subscription.LoadRelayProfiles(ctx, source, stdin)
}

func printConfigSubscriptionUsage() {
	fmt.Fprintln(os.Stderr, `用法:
  mingsui config subscription list [flags]
  mingsui config subscription add <name> -url <url> [flags]
  mingsui config subscription remove <name> [flags]
  mingsui config subscription sync <name> [flags]`)
}
