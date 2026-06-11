package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/coyoteXujie/mingsui/internal/config"
	"github.com/coyoteXujie/mingsui/internal/mihomo"
	"github.com/coyoteXujie/mingsui/internal/proxycheck"
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
	jsonOutput := fs.Bool("json", false, "以 JSON 格式输出导入结果")
	checkSettings := proxyCheckSettings{}
	if allowProxy {
		checkSettings.bind(fs)
	}
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
		checkSettings.read()
		return importProxyProfiles(cfg, *cfgPath, data, *source, strings.TrimSpace(*subscriptionName), *force, strings.TrimSpace(*selectName), *selectFirst, *jsonOutput, checkSettings, err)
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
	report := subscription.BuildSyncReport(subscription.SyncReportInput{
		Name:                  importedSubscriptionReportName(strings.TrimSpace(*subscriptionName)),
		Kind:                  subscription.SyncKindRelay,
		Message:               "节点已导入",
		Imported:              len(profiles),
		ImportedRelayProfiles: profiles,
		Config:                cfg,
	})
	if *jsonOutput {
		return writeSubscriptionCommandResult(report, nil, nil)
	}
	fmt.Fprintf(os.Stdout, "已导入 %d 个 profile\n", len(profiles))
	printSubscriptionSyncReport(report)
	return 0
}

type subscriptionCommandResult struct {
	OK         bool                    `json:"ok"`
	Message    string                  `json:"message"`
	Report     subscription.SyncReport `json:"report"`
	ProxyCheck *proxycheck.Report      `json:"proxy_check,omitempty"`
	Error      string                  `json:"error,omitempty"`
}

type proxyCheckSettings struct {
	Enabled    bool
	TargetURL  string
	Timeout    time.Duration
	Limit      int
	enabledPtr *bool
	urlPtr     *string
	timeoutPtr *time.Duration
	limitPtr   *int
}

func (s *proxyCheckSettings) bind(fs *flag.FlagSet) {
	s.enabledPtr = fs.Bool("check", false, "导入或同步机场后测速并选择最快国外节点")
	s.urlPtr = fs.String("check-url", proxycheck.DefaultTargetURL, "机场节点测速 URL")
	s.timeoutPtr = fs.Duration("check-timeout", proxycheck.DefaultTimeout, "每个机场节点测速超时时间")
	s.limitPtr = fs.Int("check-limit", 0, "最多检测多少个候选机场节点，0 表示不限制")
}

func (s *proxyCheckSettings) read() {
	if s.enabledPtr == nil {
		return
	}
	s.Enabled = *s.enabledPtr
	s.TargetURL = *s.urlPtr
	s.Timeout = *s.timeoutPtr
	s.Limit = *s.limitPtr
}

func (s proxyCheckSettings) options() proxycheck.Options {
	return proxycheck.Options{
		TargetURL: s.TargetURL,
		Timeout:   s.Timeout,
		Limit:     s.Limit,
	}
}

func importProxyProfiles(cfg config.ClientConfig, cfgPath string, data []byte, source, subscriptionName string, force bool, selectedName string, selectFirst bool, jsonOutput bool, check proxyCheckSettings, relayErr error) int {
	if check.Enabled && selectedName != "" {
		fmt.Fprintln(os.Stderr, "不能同时使用 -select 和 -check；测速选优会自动选择最快国外节点")
		return 2
	}
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
		if name, ok := mihomo.FirstAutoSelectableProfileName(profiles); ok {
			selectedName = name
		}
	}
	if selectedName != "" {
		if err := selectExportableProxyProfile(&cfg, selectedName); err != nil {
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
	report := subscription.BuildSyncReport(subscription.SyncReportInput{
		Name:                  importedSubscriptionReportName(subscriptionName),
		Kind:                  subscription.SyncKindProxy,
		Message:               "机场节点已导入",
		Imported:              len(profiles),
		ImportedProxyProfiles: profiles,
		Config:                cfg,
	})
	checkReport, checkStage, checkErr := runAndPersistBestProxyProfile(cfgPath, &cfg, check)
	if check.Enabled && checkErr == nil {
		report = subscription.BuildSyncReport(subscription.SyncReportInput{
			Name:                  importedSubscriptionReportName(subscriptionName),
			Kind:                  subscription.SyncKindProxy,
			Message:               "机场节点已导入",
			Imported:              len(profiles),
			ImportedProxyProfiles: profiles,
			Config:                cfg,
		})
	}
	if jsonOutput {
		if check.Enabled && checkErr != nil {
			return writeSubscriptionCommandResult(report, &checkReport, subscriptionProxyCheckError(checkStage, checkErr))
		}
		return writeSubscriptionCommandResult(report, optionalProxyCheckReport(check.Enabled, checkReport), nil)
	}
	fmt.Fprintf(os.Stdout, "已导入 %d 个机场节点\n", len(profiles))
	printSubscriptionSyncReport(report)
	if check.Enabled && checkErr != nil {
		printSubscriptionProxyCheckError(checkStage, checkErr)
		return 1
	}
	if check.Enabled {
		printPersistedBestProxyProfile(checkReport)
	}
	return 0
}

func importedSubscriptionReportName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "import"
	}
	return name
}

func runAndPersistBestProxyProfile(cfgPath string, cfg *config.ClientConfig, check proxyCheckSettings) (proxycheck.Report, string, error) {
	if !check.Enabled {
		return proxycheck.Report{}, "", nil
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	report, err := proxyCheckRunner(ctx, *cfg, check.options())
	if err != nil {
		return report, "测速选优失败", err
	}
	if err := selectBestProxyProfileFromReport(cfg, &report); err != nil {
		report.Error = err.Error()
		return report, "选择最快节点失败", err
	}
	if err := config.WriteClient(cfgPath, *cfg, true); err != nil {
		report.Error = err.Error()
		return report, "写入测速选择失败", err
	}
	return report, "", nil
}

func optionalProxyCheckReport(enabled bool, report proxycheck.Report) *proxycheck.Report {
	if !enabled {
		return nil
	}
	return &report
}

func writeSubscriptionCommandResult(report subscription.SyncReport, checkReport *proxycheck.Report, err error) int {
	result := subscriptionCommandResult{
		OK:         err == nil,
		Message:    report.Message,
		Report:     report,
		ProxyCheck: checkReport,
	}
	if err != nil {
		result.Error = err.Error()
		result.Message = err.Error()
	}
	if code := writeJSONOrError(result); code != 0 {
		return code
	}
	if err != nil {
		return 1
	}
	return 0
}

func subscriptionProxyCheckError(stage string, err error) error {
	if err == nil {
		return nil
	}
	if strings.TrimSpace(stage) == "写入测速选择失败" {
		return fmt.Errorf("%s: %w", stage, err)
	}
	return fmt.Errorf("机场节点已保存，但%s: %w", stage, err)
}

func printSubscriptionProxyCheckError(stage string, err error) {
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "%v\n", subscriptionProxyCheckError(stage, err))
}

func printPersistedBestProxyProfile(report proxycheck.Report) {
	best, _ := report.Best()
	fmt.Fprintf(os.Stdout, "已测速选择最快国外节点 %s (%d ms)\n", report.Selected, best.LatencyMS)
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
	jsonOutput := fs.Bool("json", false, "以 JSON 格式输出订阅列表")
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
	if *jsonOutput {
		return writeJSONOrError(subscriptionItems(cfg.Subscriptions))
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

type subscriptionItem struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type subscriptionMutationResult struct {
	OK      bool   `json:"ok"`
	Action  string `json:"action"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

func subscriptionItems(subscriptions []config.RelaySubscription) []subscriptionItem {
	items := make([]subscriptionItem, 0, len(subscriptions))
	for _, sub := range subscriptions {
		items = append(items, subscriptionItem{Name: sub.Name, URL: sub.URL})
	}
	return items
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
	jsonOutput := fs.Bool("json", false, "以 JSON 格式输出写入结果")
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
	if *jsonOutput {
		return writeJSONOrError(subscriptionMutationResult{
			OK:      true,
			Action:  "add",
			Name:    name,
			Message: "订阅已写入",
		})
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
	jsonOutput := fs.Bool("json", false, "以 JSON 格式输出删除结果")
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
	if *jsonOutput {
		return writeJSONOrError(subscriptionMutationResult{
			OK:      true,
			Action:  "remove",
			Name:    name,
			Message: "订阅已删除",
		})
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
	jsonOutput := fs.Bool("json", false, "以 JSON 格式输出同步结果")
	checkSettings := proxyCheckSettings{}
	checkSettings.bind(fs)
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	checkSettings.read()

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
	return syncSubscriptionData(cfg, *cfgPath, name, data, *force, strings.TrimSpace(*selectName), *selectFirst, *jsonOutput, checkSettings)
}

func syncSubscriptionData(cfg config.ClientConfig, cfgPath, subscriptionName string, data []byte, force bool, selectedName string, selectFirst bool, jsonOutput bool, check proxyCheckSettings) int {
	profiles, relayErr := subscription.ParseRelayProfiles(data)
	if relayErr != nil {
		return syncProxySubscription(cfg, cfgPath, subscriptionName, data, force, selectedName, selectFirst, jsonOutput, check, relayErr)
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
	report := subscription.BuildSyncReport(subscription.SyncReportInput{
		Name:                  subscriptionName,
		Kind:                  subscription.SyncKindRelay,
		Imported:              len(profiles),
		ImportedRelayProfiles: profiles,
		Config:                cfg,
	})
	if jsonOutput {
		return writeSubscriptionCommandResult(report, nil, nil)
	}
	fmt.Fprintf(os.Stdout, "已同步订阅 %s，导入 %d 个 profile\n", subscriptionName, len(profiles))
	printSubscriptionSyncReport(report)
	return 0
}

func syncProxySubscription(cfg config.ClientConfig, cfgPath, subscriptionName string, data []byte, force bool, selectedName string, selectFirst bool, jsonOutput bool, check proxyCheckSettings, relayErr error) int {
	if check.Enabled && selectedName != "" {
		fmt.Fprintln(os.Stderr, "不能同时使用 -select 和 -check；测速选优会自动选择最快国外节点")
		return 2
	}
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
		if name, ok := mihomo.FirstAutoSelectableProfileName(profiles); ok {
			selectedName = name
		}
	}
	if selectedName != "" {
		if err := selectExportableProxyProfile(&cfg, selectedName); err != nil {
			fmt.Fprintf(os.Stderr, "选择机场节点失败: %v\n", err)
			return 1
		}
	}
	if err := config.WriteClient(cfgPath, cfg, true); err != nil {
		fmt.Fprintf(os.Stderr, "写入配置失败: %v\n", err)
		return 1
	}
	report := subscription.BuildSyncReport(subscription.SyncReportInput{
		Name:                  subscriptionName,
		Kind:                  subscription.SyncKindProxy,
		Imported:              len(profiles),
		ImportedProxyProfiles: profiles,
		Config:                cfg,
	})
	checkReport, checkStage, checkErr := runAndPersistBestProxyProfile(cfgPath, &cfg, check)
	if check.Enabled && checkErr == nil {
		report = subscription.BuildSyncReport(subscription.SyncReportInput{
			Name:                  subscriptionName,
			Kind:                  subscription.SyncKindProxy,
			Imported:              len(profiles),
			ImportedProxyProfiles: profiles,
			Config:                cfg,
		})
	}
	if jsonOutput {
		if check.Enabled && checkErr != nil {
			return writeSubscriptionCommandResult(report, &checkReport, subscriptionProxyCheckError(checkStage, checkErr))
		}
		return writeSubscriptionCommandResult(report, optionalProxyCheckReport(check.Enabled, checkReport), nil)
	}
	fmt.Fprintf(os.Stdout, "已同步订阅 %s，导入 %d 个机场节点\n", subscriptionName, len(profiles))
	printSubscriptionSyncReport(report)
	if check.Enabled && checkErr != nil {
		printSubscriptionProxyCheckError(checkStage, checkErr)
		return 1
	}
	if check.Enabled {
		printPersistedBestProxyProfile(checkReport)
	}
	return 0
}

func printSubscriptionSyncReport(report subscription.SyncReport) {
	switch report.Kind {
	case subscription.SyncKindRelay:
		fmt.Fprintf(os.Stdout, "检查结果: relay profile 总数 %d", report.RelayProfiles)
	case subscription.SyncKindProxy:
		fmt.Fprintf(
			os.Stdout,
			"检查结果: 本次可连接 %d / 自动候选 %d；节点总数 %d / 可连接总数 %d",
			report.ImportedExportableProxyProfiles,
			report.ImportedAutoSelectableProxyProfiles,
			report.ProxyProfiles,
			report.ExportableProxyProfiles,
		)
	default:
		fmt.Fprint(os.Stdout, "检查结果: 未知订阅类型")
	}
	if report.Selected != "" {
		fmt.Fprintf(os.Stdout, "；当前 %s", report.Selected)
	}
	fmt.Fprintln(os.Stdout)
	for _, warning := range report.Warnings {
		fmt.Fprintf(os.Stdout, "警告: %s\n", warning)
	}
}

func selectExportableProxyProfile(cfg *config.ClientConfig, name string) error {
	profile, ok := cfg.ProxyProfile(name)
	if !ok {
		return fmt.Errorf("proxy profile %q not found", name)
	}
	if !mihomo.CanExportProfile(profile) {
		return fmt.Errorf("%s 当前暂不支持直接连接；使用 mingsui config proxy list 查看可连接节点", name)
	}
	return cfg.SelectProxyProfile(name)
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
  mingsui config subscription list [-json] [-secrets] [flags]
  mingsui config subscription add <name> -url <url> [-json] [flags]
  mingsui config subscription remove <name> [-json] [flags]
  mingsui config subscription sync <name> [-select <node>|-check] [-json] [flags]`)
}
