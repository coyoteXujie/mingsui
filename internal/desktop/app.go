package desktop

import (
	"context"
	"errors"
	"log"
	"os"
	"sync"

	"github.com/coyoteXujie/mingsui/internal/client"
	"github.com/coyoteXujie/mingsui/internal/config"
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
	controller, err := client.NewController(cfg, logger)
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

	controller, err := client.NewController(saved, a.logger)
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

func (a *App) Start(ctx context.Context) error {
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

func (a *App) CheckRelayStatus(ctx context.Context) (client.RelayHealth, error) {
	a.mu.Lock()
	cfg := a.cfg
	logger := a.logger
	a.mu.Unlock()

	service, err := client.NewService(cfg, logger)
	if err != nil {
		return client.RelayHealth{}, err
	}
	return service.CheckRelayStatus(ctx)
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
