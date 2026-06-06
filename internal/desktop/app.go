package desktop

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/coyoteXujie/mingsui/internal/client"
	"github.com/coyoteXujie/mingsui/internal/config"
	"github.com/coyoteXujie/mingsui/internal/mihomo"
	"github.com/coyoteXujie/mingsui/internal/proxycheck"
	"github.com/coyoteXujie/mingsui/internal/subscription"
	"github.com/coyoteXujie/mingsui/internal/systemproxy"
)

var proxyCheckRunner = proxycheck.Check

type App struct {
	mu         sync.Mutex
	cfgPath    string
	cfg        config.ClientConfig
	logger     *log.Logger
	controller *client.Controller
	kernel     *mihomo.Controller
}

func NewApp(cfgPath string, logger *log.Logger) (*App, error) {
	if cfgPath == "" {
		cfgPath = config.DefaultClientPath()
	}
	cfg, err := loadClientConfigOrDefault(cfgPath)
	if err != nil {
		return nil, err
	}
	controllerCfg, err := effectiveClientConfig(cfg)
	if err != nil {
		return nil, err
	}
	controller, err := client.NewController(controllerCfg, logger)
	if err != nil {
		return nil, err
	}
	if logger == nil {
		logger = log.Default()
	}

	return &App{
		cfgPath:    cfgPath,
		cfg:        cfg,
		logger:     logger,
		controller: controller,
		kernel:     mihomo.NewController(cfg, mihomo.Options{Stdout: logger.Writer(), Stderr: logger.Writer()}),
	}, nil
}

func (a *App) ConfigPath() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cfgPath
}

func (a *App) Config() config.ClientConfig {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cfg.Clone()
}

func (a *App) SaveConfig(cfg config.ClientConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	saved := cfg.Clone()

	a.mu.Lock()
	defer a.mu.Unlock()
	if a.controller.Status().Running || a.kernel.Status().Running {
		return errors.New("客户端运行中，请停止后再修改配置")
	}

	controllerCfg, err := effectiveClientConfig(saved)
	if err != nil {
		return err
	}
	controller, err := client.NewController(controllerCfg, a.logger)
	if err != nil {
		return err
	}
	if err := config.WriteClient(a.cfgPath, saved, true); err != nil {
		return err
	}
	a.cfg = saved
	a.controller = controller
	a.kernel = mihomo.NewController(saved, mihomo.Options{Stdout: a.logger.Writer(), Stderr: a.logger.Writer()})
	return nil
}

func (a *App) UpsertRelayProfile(profile config.RelayProfile, replace bool) error {
	cfg := a.Config()
	if err := cfg.UpsertRelayProfile(profile, replace); err != nil {
		return err
	}
	return a.SaveConfig(cfg)
}

func (a *App) SelectRelayProfile(name string) error {
	cfg := a.Config()
	if err := cfg.SelectRelayProfile(name); err != nil {
		return err
	}
	return a.SaveConfig(cfg)
}

func (a *App) SelectProxyProfile(name string) error {
	cfg := a.Config()
	if err := selectExportableProxyProfile(&cfg, name); err != nil {
		return err
	}
	return a.SaveConfig(cfg)
}

func (a *App) RemoveProxyProfile(name string) error {
	cfg := a.Config()
	if err := cfg.RemoveProxyProfile(name); err != nil {
		return err
	}
	return a.SaveConfig(cfg)
}

func (a *App) RemoveRelayProfile(name string) error {
	cfg := a.Config()
	if err := cfg.RemoveRelayProfile(name); err != nil {
		return err
	}
	return a.SaveConfig(cfg)
}

func (a *App) RenameRelayProfile(oldName, newName string) error {
	cfg := a.Config()
	if err := cfg.RenameRelayProfile(oldName, newName); err != nil {
		return err
	}
	return a.SaveConfig(cfg)
}

func (a *App) ImportRelayProfiles(data []byte, replace bool, selectName string) (int, error) {
	content := strings.TrimSpace(string(data))
	sourceURL := ""
	if strings.HasPrefix(content, "http://") || strings.HasPrefix(content, "https://") {
		var err error
		sourceURL = content
		data, err = subscription.LoadSource(context.Background(), content, nil)
		if err != nil {
			return 0, err
		}
	}

	profiles, err := subscription.ParseRelayProfiles(data)
	if err != nil {
		return a.importProxyProfiles(data, replace, selectName, sourceURL, err)
	}
	cfg := a.Config()
	if err := cfg.ImportRelayProfiles(profiles, replace); err != nil {
		return 0, err
	}
	if strings.TrimSpace(selectName) == "" && len(profiles) > 0 {
		selectName = profiles[0].Name
	}
	if selectName != "" {
		if err := cfg.SelectRelayProfile(selectName); err != nil {
			return 0, err
		}
	}
	if err := saveImportedSubscription(&cfg, sourceURL); err != nil {
		return 0, err
	}
	if err := a.SaveConfig(cfg); err != nil {
		return 0, err
	}
	return len(profiles), nil
}

func (a *App) importProxyProfiles(data []byte, replace bool, selectName, sourceURL string, relayErr error) (int, error) {
	profiles, err := subscription.ParseProxyProfiles(data)
	if err != nil {
		return 0, relayErr
	}
	cfg := a.Config()
	if err := cfg.ImportProxyProfiles(profiles, replace); err != nil {
		return 0, err
	}
	if strings.TrimSpace(selectName) == "" && len(profiles) > 0 {
		if name, ok := mihomo.FirstAutoSelectableProfileName(profiles); ok {
			selectName = name
		}
	}
	if selectName != "" {
		if err := selectExportableProxyProfile(&cfg, selectName); err != nil {
			return 0, err
		}
	}
	if err := saveImportedSubscription(&cfg, sourceURL); err != nil {
		return 0, err
	}
	if err := a.SaveConfig(cfg); err != nil {
		return 0, err
	}
	return len(profiles), nil
}

func (a *App) UpsertRelaySubscription(sub config.RelaySubscription, replace bool) error {
	cfg := a.Config()
	if err := cfg.UpsertRelaySubscription(sub, replace); err != nil {
		return err
	}
	return a.SaveConfig(cfg)
}

func (a *App) RemoveRelaySubscription(name string) error {
	cfg := a.Config()
	if err := cfg.RemoveRelaySubscription(name); err != nil {
		return err
	}
	return a.SaveConfig(cfg)
}

func (a *App) SyncRelaySubscription(ctx context.Context, name string, replace bool, selectName string) (int, error) {
	cfg := a.Config()
	sub, ok := cfg.RelaySubscription(name)
	if !ok {
		return 0, errors.New("订阅不存在")
	}
	profiles, err := subscription.LoadRelayProfiles(ctx, sub.URL, nil)
	if err == nil {
		if err := cfg.ImportRelayProfiles(profiles, replace); err != nil {
			return 0, err
		}
		if strings.TrimSpace(selectName) == "" && strings.TrimSpace(cfg.ActiveProfile) == "" && strings.TrimSpace(cfg.ActiveProxyProfile) == "" && len(profiles) > 0 {
			selectName = profiles[0].Name
		}
		if selectName != "" {
			if err := cfg.SelectRelayProfile(selectName); err != nil {
				return 0, err
			}
		}
		if err := a.SaveConfig(cfg); err != nil {
			return 0, err
		}
		return len(profiles), nil
	}

	data, dataErr := subscription.LoadSource(ctx, sub.URL, nil)
	if dataErr != nil {
		return 0, dataErr
	}
	proxyProfiles, proxyErr := subscription.ParseProxyProfiles(data)
	if proxyErr != nil {
		return 0, err
	}
	if err := cfg.ImportProxyProfiles(proxyProfiles, replace); err != nil {
		return 0, err
	}
	if strings.TrimSpace(selectName) == "" && strings.TrimSpace(cfg.ActiveProfile) == "" && strings.TrimSpace(cfg.ActiveProxyProfile) == "" && len(proxyProfiles) > 0 {
		if name, ok := mihomo.FirstAutoSelectableProfileName(proxyProfiles); ok {
			selectName = name
		}
	}
	if selectName != "" {
		if err := selectExportableProxyProfile(&cfg, selectName); err != nil {
			return 0, err
		}
	}
	if err := a.SaveConfig(cfg); err != nil {
		return 0, err
	}
	return len(proxyProfiles), nil
}

func saveImportedSubscription(cfg *config.ClientConfig, sourceURL string) error {
	sourceURL = strings.TrimSpace(sourceURL)
	if sourceURL == "" {
		return nil
	}
	name := "airport"
	for _, sub := range cfg.Subscriptions {
		if strings.TrimSpace(sub.URL) == sourceURL {
			name = sub.Name
			break
		}
	}
	return cfg.UpsertRelaySubscription(config.RelaySubscription{Name: name, URL: sourceURL}, true)
}

func (a *App) Start(ctx context.Context) error {
	if proxy, ok := activeProxyProfile(a.Config()); ok {
		a.logger.Printf("启动 Mihomo 内核: %s (%s)", proxy.Name, proxy.Protocol)
		a.mu.Lock()
		kernel := a.kernel
		a.mu.Unlock()
		return kernel.Start(ctx)
	}
	if hasProxyModeWithoutAutoSelectableSelection(a.Config()) {
		return errors.New("当前机场订阅中没有可自动选择的国外节点")
	}
	a.mu.Lock()
	controller := a.controller
	a.mu.Unlock()
	return controller.Start(ctx)
}

func (a *App) Stop(ctx context.Context) error {
	a.mu.Lock()
	controller := a.controller
	kernel := a.kernel
	a.mu.Unlock()
	if err := kernel.Stop(ctx); err != nil {
		return err
	}
	return controller.Stop(ctx)
}

func (a *App) Status() client.RuntimeStatus {
	a.mu.Lock()
	controller := a.controller
	kernel := a.kernel
	cfg := a.cfg
	a.mu.Unlock()
	if _, ok := activeProxyProfile(cfg); ok {
		return kernelClientStatus(kernel.Status())
	}
	return controller.Status()
}

func (a *App) CheckRelay(ctx context.Context) error {
	_, err := a.CheckRelayStatus(ctx)
	return err
}

func (a *App) EnableSystemProxy(ctx context.Context) error {
	cfg := a.Config()
	if cfg.LocalAuth.Enabled {
		return errors.New("系统代理暂不支持本地代理认证；请先关闭本地代理认证，或手动为浏览器配置代理认证")
	}
	return systemproxy.Enable(ctx, systemproxy.Config{HTTPAddr: cfg.HTTPAddr, SOCKSAddr: cfg.LocalAddr})
}

func (a *App) DisableSystemProxy(ctx context.Context) error {
	return systemproxy.Disable(ctx)
}

func (a *App) SystemProxyStatus(ctx context.Context) systemproxy.Status {
	return systemproxy.CurrentStatus(ctx)
}

func (a *App) CheckRelayProfile(ctx context.Context, name string) error {
	_, err := a.CheckRelayProfileStatus(ctx, name)
	return err
}

func (a *App) CheckProxyKernel(ctx context.Context) (config.ProxyProfile, error) {
	a.mu.Lock()
	cfg := a.cfg
	a.mu.Unlock()

	proxy, ok := activeProxyProfile(cfg)
	if !ok {
		return config.ProxyProfile{}, errors.New("当前没有选择机场节点")
	}
	workDir, err := os.MkdirTemp("", "mingsui-desktop-check-*")
	if err != nil {
		return config.ProxyProfile{}, err
	}
	defer os.RemoveAll(workDir)
	if _, err := mihomo.TestConfig(ctx, cfg, mihomo.Options{WorkDir: workDir}); err != nil {
		return config.ProxyProfile{}, err
	}
	return proxy, nil
}

func (a *App) CheckProxyProfiles(ctx context.Context, opts proxycheck.Options, selectBest bool) (proxycheck.Report, error) {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	running := a.controller.Status().Running || a.kernel.Status().Running
	a.mu.Unlock()
	if running {
		return proxycheck.Report{}, errors.New("客户端运行中，请先断开再测速选优")
	}

	report, err := proxyCheckRunner(ctx, cfg, opts)
	if err != nil {
		return report, err
	}
	if !selectBest {
		return report, nil
	}
	best, ok := report.Best()
	if !ok {
		return report, proxycheck.ErrNoHealthyNode
	}
	if err := cfg.SelectProxyProfile(best.Name); err != nil {
		return report, err
	}
	if err := a.SaveConfig(cfg); err != nil {
		return report, err
	}
	report.Selected = best.Name
	return report, nil
}

func (a *App) CheckProxyProfile(ctx context.Context, name string, opts proxycheck.Options) (proxycheck.Report, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return proxycheck.Report{}, errors.New("机场节点名称不能为空")
	}

	a.mu.Lock()
	cfg := a.cfg.Clone()
	running := a.controller.Status().Running || a.kernel.Status().Running
	a.mu.Unlock()
	if running {
		return proxycheck.Report{}, errors.New("客户端运行中，请先断开再检测节点")
	}

	profile, ok := cfg.ProxyProfile(name)
	if !ok {
		return proxycheck.Report{}, errors.New("机场节点不存在")
	}
	cfg.ActiveProfile = ""
	cfg.ActiveProxyProfile = profile.Name
	cfg.ProxyProfiles = []config.ProxyProfile{profile}
	opts.IncludeNonAutoSelectable = true
	return proxyCheckRunner(ctx, cfg, opts)
}

func (a *App) CheckRelayStatus(ctx context.Context) (client.RelayHealth, error) {
	a.mu.Lock()
	cfg := a.cfg
	logger := a.logger
	a.mu.Unlock()

	if proxy, ok := activeProxyProfile(cfg); ok {
		if _, err := mihomo.Prepare(cfg, mihomo.Options{}); err != nil {
			return client.RelayHealth{}, err
		}
		return client.RelayHealth{}, errors.New("当前选择的是机场节点 " + proxy.Name + "；请直接连接启动 Mihomo 内核")
	}
	if hasProxyModeWithoutAutoSelectableSelection(cfg) {
		return client.RelayHealth{}, errors.New("当前机场订阅中没有可自动选择的国外节点")
	}
	cfg, err := effectiveClientConfig(cfg)
	if err != nil {
		return client.RelayHealth{}, err
	}
	service, err := client.NewService(cfg, logger)
	if err != nil {
		return client.RelayHealth{}, err
	}
	return service.CheckRelayStatus(ctx)
}

func (a *App) CheckRelayProfileStatus(ctx context.Context, name string) (client.RelayHealth, error) {
	a.mu.Lock()
	cfg := a.cfg
	logger := a.logger
	a.mu.Unlock()

	cfg, err := cfg.ResolveProfile(name)
	if err != nil {
		return client.RelayHealth{}, err
	}
	service, err := client.NewService(cfg, logger)
	if err != nil {
		return client.RelayHealth{}, err
	}
	return service.CheckRelayStatus(ctx)
}

func effectiveClientConfig(cfg config.ClientConfig) (config.ClientConfig, error) {
	profileName := strings.TrimSpace(cfg.ActiveProfile)
	if profileName == "" && len(cfg.Profiles) > 0 {
		profileName = cfg.Profiles[0].Name
	}
	return cfg.ResolveProfile(profileName)
}

func activeProxyProfile(cfg config.ClientConfig) (config.ProxyProfile, bool) {
	name := strings.TrimSpace(cfg.ActiveProxyProfile)
	if name == "" && strings.TrimSpace(cfg.ActiveProfile) == "" && len(cfg.ProxyProfiles) > 0 {
		var ok bool
		name, ok = mihomo.FirstAutoSelectableProfileName(cfg.ProxyProfiles)
		if !ok {
			return config.ProxyProfile{}, false
		}
	}
	if name == "" {
		return config.ProxyProfile{}, false
	}
	return cfg.ProxyProfile(name)
}

func hasProxyModeWithoutAutoSelectableSelection(cfg config.ClientConfig) bool {
	return strings.TrimSpace(cfg.ActiveProfile) == "" && len(cfg.ProxyProfiles) > 0
}

func selectExportableProxyProfile(cfg *config.ClientConfig, name string) error {
	profile, ok := cfg.ProxyProfile(name)
	if !ok {
		return errors.New("机场节点不存在")
	}
	if !mihomo.CanExportProfile(profile) {
		return errors.New("该机场节点当前暂不支持直接连接")
	}
	return cfg.SelectProxyProfile(name)
}

func kernelClientStatus(status mihomo.RuntimeStatus) client.RuntimeStatus {
	relayAddr := ""
	if status.BinaryPath != "" {
		relayAddr = "mihomo: " + status.BinaryPath
	}
	return client.RuntimeStatus{
		Running:   status.Running,
		LocalAddr: status.LocalAddr,
		HTTPAddr:  status.HTTPAddr,
		RelayAddr: relayAddr,
		StartedAt: status.StartedAt,
		LastError: status.LastError,
	}
}

func loadClientConfigOrDefault(path string) (config.ClientConfig, error) {
	cfg, err := config.LoadClient(path)
	if err == nil {
		return cfg, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return config.DefaultClient(), nil
	}
	return config.ClientConfig{}, err
}
