package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/coyoteXujie/mingsui/internal/config"
	"github.com/coyoteXujie/mingsui/internal/mihomo"
	"github.com/coyoteXujie/mingsui/internal/subscription"
)

func importClientProfiles(args []string) int {
	return importClientProfilesCommand("config profile import", args, false, false, false)
}

func importClientProfilesProduct(args []string) int {
	return importClientProfilesCommand("import", args, true, true, true)
}

func importClientProfilesCommand(name string, args []string, forceDefault, selectFirstDefault, allowProxy bool) int {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	cfgPath := fs.String("path", config.DefaultClientPath(), "客户端配置文件路径")
	source := fs.String("source", "", "订阅来源：本地 JSON 文件、HTTP(S) URL 或 - 表示 stdin")
	force := fs.Bool("force", forceDefault, "覆盖同名节点")
	selectName := fs.String("select", "", "导入后选择指定节点")
	selectFirst := fs.Bool("select-first", selectFirstDefault, "未指定 -select 时选择导入的第一个节点")
	subscriptionName := fs.String("subscription", "", "导入成功后把 HTTP(S) 来源保存为指定订阅名称")
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
	data, err := loadSourceData(*source, os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取订阅失败: %v\n", err)
		return 1
	}

	profiles, err := subscription.ParseRelayProfiles(data)
	if err != nil && allowProxy {
		return importProxyProfiles(cfg, *cfgPath, data, *source, strings.TrimSpace(*subscriptionName), *force, strings.TrimSpace(*selectName), *selectFirst, err)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取订阅失败: %v\n", err)
		return 1
	}
	if err := cfg.ImportRelayProfiles(profiles, *force); err != nil {
		fmt.Fprintf(os.Stderr, "导入 profile 失败: %v\n", err)
		return 1
	}
	selectedName := strings.TrimSpace(*selectName)
	if selectedName == "" && *selectFirst && len(profiles) > 0 {
		selectedName = profiles[0].Name
	}
	if selectedName != "" {
		if err := cfg.SelectRelayProfile(selectedName); err != nil {
			fmt.Fprintf(os.Stderr, "选择 profile 失败: %v\n", err)
			return 1
		}
	}
	if err := saveImportedSubscription(&cfg, strings.TrimSpace(*subscriptionName), *source, *force); err != nil {
		fmt.Fprintf(os.Stderr, "保存订阅失败: %v\n", err)
		return 1
	}
	if err := config.WriteClient(*cfgPath, cfg, true); err != nil {
		fmt.Fprintf(os.Stderr, "写入配置失败: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stdout, "已导入 %d 个 profile\n", len(profiles))
	return 0
}

func importProxyProfiles(cfg config.ClientConfig, cfgPath string, data []byte, source, subscriptionName string, force bool, selectedName string, selectFirst bool, relayErr error) int {
	profiles, err := subscription.ParseProxyProfiles(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取订阅失败: %v\n", relayErr)
		return 1
	}
	if err := cfg.ImportProxyProfiles(profiles, force); err != nil {
		fmt.Fprintf(os.Stderr, "导入机场节点失败: %v\n", err)
		return 1
	}
	if selectedName == "" && selectFirst && len(profiles) > 0 {
		if name, ok := mihomo.FirstExportableProfileName(profiles); ok {
			selectedName = name
		} else {
			selectedName = profiles[0].Name
		}
	}
	if selectedName != "" {
		if err := cfg.SelectProxyProfile(selectedName); err != nil {
			fmt.Fprintf(os.Stderr, "选择机场节点失败: %v\n", err)
			return 1
		}
	}
	if err := saveImportedSubscription(&cfg, subscriptionName, source, force); err != nil {
		fmt.Fprintf(os.Stderr, "保存订阅失败: %v\n", err)
		return 1
	}
	if err := config.WriteClient(cfgPath, cfg, true); err != nil {
		fmt.Fprintf(os.Stderr, "写入配置失败: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stdout, "已导入 %d 个机场节点\n", len(profiles))
	return 0
}

func saveImportedSubscription(cfg *config.ClientConfig, name, source string, force bool) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	source = strings.TrimSpace(source)
	if !strings.HasPrefix(source, "http://") && !strings.HasPrefix(source, "https://") {
		return fmt.Errorf("只有 HTTP(S) 来源可以保存为订阅")
	}
	return cfg.UpsertRelaySubscription(config.RelaySubscription{Name: name, URL: source}, force)
}

func exportClientProfiles(args []string) int {
	fs := flag.NewFlagSet("config profile export", flag.ContinueOnError)
	cfgPath := fs.String("path", config.DefaultClientPath(), "客户端配置文件路径")
	outputPath := fs.String("output", "", "输出文件路径，留空则打印到 stdout")
	showSecrets := fs.Bool("secrets", false, "导出真实 profile token")
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
	profiles, err := selectExportProfiles(cfg.Profiles, fs.Args())
	if err != nil {
		fmt.Fprintf(os.Stderr, "导出 profile 失败: %v\n", err)
		return 1
	}
	doc := subscription.Document{
		Version:  1,
		Profiles: profiles,
	}
	if *outputPath == "" {
		if err := writeSubscriptionJSON(os.Stdout, doc); err != nil {
			fmt.Fprintf(os.Stderr, "输出订阅失败: %v\n", err)
			return 1
		}
		return 0
	}
	file, err := os.OpenFile(*outputPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "打开输出文件失败: %v\n", err)
		return 1
	}
	defer file.Close()
	if err := writeSubscriptionJSON(file, doc); err != nil {
		fmt.Fprintf(os.Stderr, "写入订阅失败: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stdout, "已导出 %d 个 profile 到 %s\n", len(profiles), *outputPath)
	return 0
}

func selectExportProfiles(profiles []config.RelayProfile, names []string) ([]config.RelayProfile, error) {
	if len(profiles) == 0 {
		return nil, fmt.Errorf("没有可导出的 relay profile")
	}
	if len(names) == 0 {
		return append([]config.RelayProfile(nil), profiles...), nil
	}

	selected := make([]config.RelayProfile, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		found := false
		for _, profile := range profiles {
			if strings.TrimSpace(profile.Name) == name {
				selected = append(selected, profile)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("profile %q 不存在", name)
		}
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("没有可导出的 relay profile")
	}
	return selected, nil
}

func writeSubscriptionJSON(w io.Writer, doc subscription.Document) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(doc)
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
	selectFirst := fs.Bool("select-first", true, "未指定 -select 且当前未选择节点时选择同步到的第一个节点")
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

	data, err := loadSourceData(sub.URL, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取订阅失败: %v\n", err)
		return 1
	}
	return syncSubscriptionData(cfg, *cfgPath, name, data, *force, strings.TrimSpace(*selectName), *selectFirst)
}

func syncSubscriptionData(cfg config.ClientConfig, cfgPath, subscriptionName string, data []byte, force bool, selectedName string, selectFirst bool) int {
	profiles, relayErr := subscription.ParseRelayProfiles(data)
	if relayErr != nil {
		return syncProxySubscription(cfg, cfgPath, subscriptionName, data, force, selectedName, selectFirst, relayErr)
	}
	if err := cfg.ImportRelayProfiles(profiles, force); err != nil {
		fmt.Fprintf(os.Stderr, "导入 profile 失败: %v\n", err)
		return 1
	}
	if selectedName == "" && selectFirst && strings.TrimSpace(cfg.ActiveProfile) == "" && strings.TrimSpace(cfg.ActiveProxyProfile) == "" && len(profiles) > 0 {
		selectedName = profiles[0].Name
	}
	if selectedName != "" {
		if err := cfg.SelectRelayProfile(selectedName); err != nil {
			fmt.Fprintf(os.Stderr, "选择 profile 失败: %v\n", err)
			return 1
		}
	}
	if err := config.WriteClient(cfgPath, cfg, true); err != nil {
		fmt.Fprintf(os.Stderr, "写入配置失败: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stdout, "已同步订阅 %s，导入 %d 个 profile\n", subscriptionName, len(profiles))
	return 0
}

func syncProxySubscription(cfg config.ClientConfig, cfgPath, subscriptionName string, data []byte, force bool, selectedName string, selectFirst bool, relayErr error) int {
	profiles, err := subscription.ParseProxyProfiles(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "同步订阅失败: %v\n", relayErr)
		return 1
	}
	if err := cfg.ImportProxyProfiles(profiles, force); err != nil {
		fmt.Fprintf(os.Stderr, "导入机场节点失败: %v\n", err)
		return 1
	}
	if selectedName == "" && selectFirst && strings.TrimSpace(cfg.ActiveProfile) == "" && strings.TrimSpace(cfg.ActiveProxyProfile) == "" && len(profiles) > 0 {
		if name, ok := mihomo.FirstExportableProfileName(profiles); ok {
			selectedName = name
		} else {
			selectedName = profiles[0].Name
		}
	}
	if selectedName != "" {
		if err := cfg.SelectProxyProfile(selectedName); err != nil {
			fmt.Fprintf(os.Stderr, "选择机场节点失败: %v\n", err)
			return 1
		}
	}
	if err := config.WriteClient(cfgPath, cfg, true); err != nil {
		fmt.Fprintf(os.Stderr, "写入配置失败: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stdout, "已同步订阅 %s，导入 %d 个机场节点\n", subscriptionName, len(profiles))
	return 0
}

func loadProfilesFromSource(source string, stdin io.Reader) ([]config.RelayProfile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return subscription.LoadRelayProfiles(ctx, source, stdin)
}

func loadSourceData(source string, stdin io.Reader) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return subscription.LoadSource(ctx, source, stdin)
}

func printConfigSubscriptionUsage() {
	fmt.Fprintln(os.Stderr, `用法:
  mingsui config subscription list [flags]
  mingsui config subscription add <name> -url <url> [flags]
  mingsui config subscription remove <name> [flags]
  mingsui config subscription sync <name> [-select <node>] [flags]`)
}
