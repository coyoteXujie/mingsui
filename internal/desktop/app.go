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
	"github.com/coyoteXujie/mingsui/internal/subscription"
)

type App struct {
	mu         sync.Mutex
	cfgPath    string
	cfg        config.ClientConfig
	logger     *log.Logger
	controller *client.Controller
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
	if a.controller.Status().Running {
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
	if err := cfg.SelectProxyProfile(name); err != nil {
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
	if strings.HasPrefix(content, "http://") || strings.HasPrefix(content, "https://") {
		var err error
		data, err = subscription.LoadSource(context.Background(), content, nil)
		if err != nil {
			return 0, err
		}
	}

	profiles, err := subscription.ParseRelayProfiles(data)
	if err != nil {
		return a.importProxyProfiles(data, replace, selectName, err)
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
	if err := a.SaveConfig(cfg); err != nil {
		return 0, err
	}
	return len(profiles), nil
}

func (a *App) importProxyProfiles(data []byte, replace bool, selectName string, relayErr error) (int, error) {
	profiles, err := subscription.ParseProxyProfiles(data)
	if err != nil {
		return 0, relayErr
	}
	cfg := a.Config()
	if err := cfg.ImportProxyProfiles(profiles, replace); err != nil {
		return 0, err
	}
	if strings.TrimSpace(selectName) == "" && len(profiles) > 0 {
		selectName = profiles[0].Name
	}
	if selectName != "" {
		if err := cfg.SelectProxyProfile(selectName); err != nil {
			return 0, err
		}
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
	if selectName != "" {
		if err := cfg.SelectProxyProfile(selectName); err != nil {
			return 0, err
		}
	}
	if err := a.SaveConfig(cfg); err != nil {
		return 0, err
	}
	return len(proxyProfiles), nil
}

func (a *App) Start(ctx context.Context) error {
	if proxy, ok := activeProxyProfile(a.Config()); ok {
		return errors.New("当前选择的是机场节点 " + proxy.Name + "，通用代理内核尚未接入")
	}
	a.mu.Lock()
	controller := a.controller
	a.mu.Unlock()
	return controller.Start(ctx)
}

func (a *App) Stop(ctx context.Context) error {
	a.mu.Lock()
	controller := a.controller
	a.mu.Unlock()
	return controller.Stop(ctx)
}

func (a *App) Status() client.RuntimeStatus {
	a.mu.Lock()
	controller := a.controller
	a.mu.Unlock()
	return controller.Status()
}

func (a *App) CheckRelay(ctx context.Context) error {
	_, err := a.CheckRelayStatus(ctx)
	return err
}

func (a *App) CheckRelayProfile(ctx context.Context, name string) error {
	_, err := a.CheckRelayProfileStatus(ctx, name)
	return err
}

func (a *App) CheckRelayStatus(ctx context.Context) (client.RelayHealth, error) {
	a.mu.Lock()
	cfg := a.cfg
	logger := a.logger
	a.mu.Unlock()

	if proxy, ok := activeProxyProfile(cfg); ok {
		return client.RelayHealth{}, errors.New("当前选择的是机场节点 " + proxy.Name + "，通用代理内核尚未接入")
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
		name = cfg.ProxyProfiles[0].Name
	}
	if name == "" {
		return config.ProxyProfile{}, false
	}
	return cfg.ProxyProfile(name)
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
