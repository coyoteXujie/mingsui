package main

import (
	"context"
	"io"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/coyoteXujie/mingsui/internal/client"
	"github.com/coyoteXujie/mingsui/internal/config"
	"github.com/coyoteXujie/mingsui/internal/desktop"
	"github.com/coyoteXujie/mingsui/internal/mihomo"
	"github.com/coyoteXujie/mingsui/internal/proxycheck"
	"github.com/coyoteXujie/mingsui/internal/systemproxy"
)

type App struct {
	ctx        context.Context
	mu         sync.Mutex
	desktopApp *desktop.App
	logs       *desktop.LogBuffer
}

func NewApp() *App {
	return &App{logs: desktop.NewLogBuffer(300)}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) ensureDesktopApp() (*desktop.App, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.desktopApp != nil {
		return a.desktopApp, nil
	}
	if a.logs == nil {
		a.logs = desktop.NewLogBuffer(300)
	}
	logger := log.New(io.MultiWriter(log.Writer(), a.logs), "mingsui desktop: ", log.LstdFlags)
	app, err := desktop.NewApp("", logger)
	if err != nil {
		return nil, err
	}
	a.desktopApp = app
	return app, nil
}

func (a *App) desktopAppIfReady() *desktop.App {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.desktopApp
}

func (a *App) GetState() (map[string]interface{}, error) {
	app, err := a.ensureDesktopApp()
	if err != nil {
		return nil, err
	}

	cfg := app.Config()
	return map[string]interface{}{
		"config_path":        app.ConfigPath(),
		"config":             cfg.Redacted(),
		"status":             app.Status(),
		"system_proxy":       app.SystemProxyStatus(context.Background()),
		"proxy_capabilities": proxyCapabilities(cfg.ProxyProfiles),
	}, nil
}

func proxyCapabilities(profiles []config.ProxyProfile) []map[string]interface{} {
	items := make([]map[string]interface{}, 0, len(profiles))
	for _, profile := range profiles {
		items = append(items, map[string]interface{}{
			"name":            profile.Name,
			"exportable":      mihomo.CanExportProfile(profile),
			"auto_selectable": mihomo.CanAutoSelectProfile(profile),
		})
	}
	return items
}

func (a *App) Start() (string, error) {
	app, err := a.ensureDesktopApp()
	if err != nil {
		return "", err
	}
	if err := app.Start(context.Background()); err != nil {
		return "", err
	}
	return "客户端已启动", nil
}

func (a *App) Stop() (string, error) {
	app := a.desktopAppIfReady()
	if app == nil {
		return "客户端未启动", nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.Stop(ctx); err != nil {
		return "", err
	}
	return "客户端已停止", nil
}

func (a *App) GetConfig() (config.ClientConfig, error) {
	app, err := a.ensureDesktopApp()
	if err != nil {
		return config.ClientConfig{}, err
	}
	return app.Config().Redacted(), nil
}

func (a *App) SaveConfig(cfg config.ClientConfig) (string, error) {
	app, err := a.ensureDesktopApp()
	if err != nil {
		return "", err
	}
	preserveRedactedSecrets(app.Config(), &cfg)
	if err := app.SaveConfig(cfg); err != nil {
		return "", err
	}
	return "配置已保存", nil
}

func (a *App) ImportProfiles(content string, replace bool, selectName string) ([]interface{}, error) {
	app, err := a.ensureDesktopApp()
	if err != nil {
		return nil, err
	}
	count, err := app.ImportRelayProfiles([]byte(content), replace, selectName)
	if err != nil {
		return nil, err
	}
	return []interface{}{count, "节点已导入"}, nil
}

func (a *App) SelectProxy(name string) (string, error) {
	app, err := a.ensureDesktopApp()
	if err != nil {
		return "", err
	}
	if err := app.SelectProxyProfile(name); err != nil {
		return "", err
	}
	return "机场节点已选择", nil
}

func (a *App) DeleteProxy(name string) (string, error) {
	app, err := a.ensureDesktopApp()
	if err != nil {
		return "", err
	}
	if err := app.RemoveProxyProfile(name); err != nil {
		return "", err
	}
	return "机场节点已删除", nil
}

func (a *App) CheckProxy(name string, timeoutSeconds int) (map[string]interface{}, error) {
	app, err := a.ensureDesktopApp()
	if err != nil {
		return nil, err
	}
	timeout := time.Duration(timeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = proxycheck.DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	report, err := app.CheckProxyProfile(ctx, name, proxycheck.Options{Timeout: timeout})
	if err != nil {
		return map[string]interface{}{"ok": false, "message": err.Error(), "report": report}, err
	}
	message := "节点测速完成"
	if best, ok := report.Best(); ok {
		message = "节点可连接，延迟 " + formatLatencyMS(best.LatencyMS)
	}
	return map[string]interface{}{"ok": true, "message": message, "report": report}, nil
}

func (a *App) CheckBestProxy(timeoutSeconds int) (map[string]interface{}, error) {
	app, err := a.ensureDesktopApp()
	if err != nil {
		return nil, err
	}
	timeout := time.Duration(timeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = proxycheck.DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	report, err := app.CheckProxyProfiles(ctx, proxycheck.Options{Timeout: timeout}, true)
	if err != nil {
		return map[string]interface{}{"ok": false, "message": err.Error(), "report": report}, err
	}
	message := "测速完成"
	if report.Selected != "" {
		message = "测速完成，已选择 " + report.Selected
	}
	return map[string]interface{}{"ok": true, "message": message, "report": report}, nil
}

func (a *App) EnableSystemProxy() (string, error) {
	app, err := a.ensureDesktopApp()
	if err != nil {
		return "", err
	}
	if err := app.EnableSystemProxy(context.Background()); err != nil {
		return "", err
	}
	return "系统代理已开启", nil
}

func (a *App) DisableSystemProxy() (string, error) {
	app, err := a.ensureDesktopApp()
	if err != nil {
		return "", err
	}
	if err := app.DisableSystemProxy(context.Background()); err != nil {
		return "", err
	}
	return "系统代理已关闭", nil
}

func (a *App) GetSystemProxyStatus() systemproxy.Status {
	app := a.desktopAppIfReady()
	if app == nil {
		return systemproxy.Status{}
	}
	return app.SystemProxyStatus(context.Background())
}

type RelayProfileRequest struct {
	Name      string                 `json:"name"`
	RelayAddr string                 `json:"relay_addr"`
	Token     string                 `json:"token"`
	TLS       config.ClientTLSConfig `json:"tls"`
	Replace   bool                   `json:"replace"`
}

func (a *App) SaveRelayProfile(req RelayProfileRequest) (string, error) {
	app, err := a.ensureDesktopApp()
	if err != nil {
		return "", err
	}
	profile := config.RelayProfile{
		Name:      req.Name,
		RelayAddr: req.RelayAddr,
		Token:     req.Token,
		TLS:       req.TLS,
	}
	preserveRedactedProfileSecret(app.Config(), &profile)
	if err := app.UpsertRelayProfile(profile, req.Replace); err != nil {
		return "", err
	}
	return "profile 已保存", nil
}

func (a *App) DeleteRelayProfile(name string) (string, error) {
	app, err := a.ensureDesktopApp()
	if err != nil {
		return "", err
	}
	if err := app.RemoveRelayProfile(name); err != nil {
		return "", err
	}
	return "profile 已删除", nil
}

func (a *App) SelectRelayProfile(name string) (string, error) {
	app, err := a.ensureDesktopApp()
	if err != nil {
		return "", err
	}
	if err := app.SelectRelayProfile(name); err != nil {
		return "", err
	}
	return "profile 已选择", nil
}

func (a *App) CheckRelayProfile(name string) (map[string]interface{}, error) {
	app, err := a.ensureDesktopApp()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	health, err := app.CheckRelayProfileStatus(ctx, name)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"ok": true, "message": "profile 可连接", "health": health}, nil
}

type SubscriptionRequest struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Replace bool   `json:"replace"`
}

func (a *App) SaveSubscription(req SubscriptionRequest) (string, error) {
	app, err := a.ensureDesktopApp()
	if err != nil {
		return "", err
	}
	sub := config.RelaySubscription{Name: req.Name, URL: req.URL}
	preserveRedactedSubscriptionSecret(app.Config(), &sub)
	if err := app.UpsertRelaySubscription(sub, req.Replace); err != nil {
		return "", err
	}
	return "订阅已保存", nil
}

func (a *App) DeleteSubscription(name string) (string, error) {
	app, err := a.ensureDesktopApp()
	if err != nil {
		return "", err
	}
	if err := app.RemoveRelaySubscription(name); err != nil {
		return "", err
	}
	return "订阅已删除", nil
}

func (a *App) SyncSubscription(name string, replace bool) ([]interface{}, error) {
	app, err := a.ensureDesktopApp()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	count, err := app.SyncRelaySubscription(ctx, name, replace, "")
	if err != nil {
		return nil, err
	}
	return []interface{}{count, "订阅已同步"}, nil
}

func (a *App) GetRuntimeStatus() client.RuntimeStatus {
	app := a.desktopAppIfReady()
	if app == nil {
		return client.RuntimeStatus{}
	}
	return app.Status()
}

func (a *App) GetLogs() []string {
	a.mu.Lock()
	logs := a.logs
	a.mu.Unlock()
	if logs == nil {
		return nil
	}
	return logs.Lines()
}

func formatLatencyMS(latency int64) string {
	return strconv.FormatInt(latency, 10) + " ms"
}

func preserveRedactedProfileSecret(current config.ClientConfig, next *config.RelayProfile) {
	if next.Token != config.RedactedValue {
		return
	}
	for _, profile := range current.Profiles {
		if profile.Name == next.Name {
			next.Token = profile.Token
			return
		}
	}
}

func preserveRedactedSubscriptionSecret(current config.ClientConfig, next *config.RelaySubscription) {
	if next.URL != config.RedactedValue {
		return
	}
	for _, sub := range current.Subscriptions {
		if sub.Name == next.Name {
			next.URL = sub.URL
			return
		}
	}
}

func preserveRedactedSecrets(current config.ClientConfig, next *config.ClientConfig) {
	if next.Token == config.RedactedValue {
		next.Token = current.Token
	}
	if next.LocalAuth.Password == config.RedactedValue {
		next.LocalAuth.Password = current.LocalAuth.Password
	}
	for i := range next.Profiles {
		if next.Profiles[i].Token != config.RedactedValue {
			continue
		}
		for _, profile := range current.Profiles {
			if profile.Name == next.Profiles[i].Name {
				next.Profiles[i].Token = profile.Token
				break
			}
		}
	}
	for i := range next.ProxyProfiles {
		if next.ProxyProfiles[i].URL != config.RedactedValue {
			continue
		}
		for _, profile := range current.ProxyProfiles {
			if profile.Name == next.ProxyProfiles[i].Name {
				next.ProxyProfiles[i].URL = profile.URL
				break
			}
		}
	}
	for i := range next.Subscriptions {
		if next.Subscriptions[i].URL != config.RedactedValue {
			continue
		}
		for _, sub := range current.Subscriptions {
			if sub.Name == next.Subscriptions[i].Name {
				next.Subscriptions[i].URL = sub.URL
				break
			}
		}
	}
}
