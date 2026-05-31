package systemproxy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type Config struct {
	HTTPAddr  string
	SOCKSAddr string
}

type Status struct {
	Supported bool   `json:"supported"`
	Enabled   bool   `json:"enabled"`
	Mode      string `json:"mode,omitempty"`
	Message   string `json:"message,omitempty"`
}

type runner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

func Enable(ctx context.Context, cfg Config) error {
	if err := requireLinuxGSettings(); err != nil {
		return err
	}
	return enable(ctx, cfg, execRunner{})
}

func Disable(ctx context.Context) error {
	if err := requireLinuxGSettings(); err != nil {
		return err
	}
	return disable(ctx, execRunner{})
}

func CurrentStatus(ctx context.Context) Status {
	if err := requireLinuxGSettings(); err != nil {
		return Status{Supported: false, Message: err.Error()}
	}
	return currentStatus(ctx, execRunner{})
}

func enable(ctx context.Context, cfg Config, run runner) error {
	httpHost, httpPort, err := splitAddr(cfg.HTTPAddr)
	if err != nil {
		return fmt.Errorf("HTTP 代理地址不正确: %w", err)
	}
	socksHost, socksPort, err := splitAddr(cfg.SOCKSAddr)
	if err != nil {
		return fmt.Errorf("SOCKS5 代理地址不正确: %w", err)
	}

	commands := [][]string{
		{"set", "org.gnome.system.proxy", "mode", "manual"},
		{"set", "org.gnome.system.proxy.http", "host", httpHost},
		{"set", "org.gnome.system.proxy.http", "port", strconv.Itoa(httpPort)},
		{"set", "org.gnome.system.proxy.https", "host", httpHost},
		{"set", "org.gnome.system.proxy.https", "port", strconv.Itoa(httpPort)},
		{"set", "org.gnome.system.proxy.socks", "host", socksHost},
		{"set", "org.gnome.system.proxy.socks", "port", strconv.Itoa(socksPort)},
	}
	for _, args := range commands {
		if out, err := run.Run(ctx, "gsettings", args...); err != nil {
			return fmt.Errorf("设置系统代理失败: %w: %s", err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

func disable(ctx context.Context, run runner) error {
	if out, err := run.Run(ctx, "gsettings", "set", "org.gnome.system.proxy", "mode", "none"); err != nil {
		return fmt.Errorf("关闭系统代理失败: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func currentStatus(ctx context.Context, run runner) Status {
	out, err := run.Run(ctx, "gsettings", "get", "org.gnome.system.proxy", "mode")
	if err != nil {
		return Status{Supported: false, Message: strings.TrimSpace(string(out))}
	}
	mode := strings.Trim(strings.TrimSpace(string(out)), "'\"")
	return Status{
		Supported: true,
		Enabled:   mode == "manual",
		Mode:      mode,
	}
}

func requireLinuxGSettings() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("当前系统 %s 暂不支持自动系统代理", runtime.GOOS)
	}
	if _, err := exec.LookPath("gsettings"); err != nil {
		return errors.New("未找到 gsettings，当前桌面环境暂不支持自动系统代理")
	}
	return nil
}

func splitAddr(addr string) (string, int, error) {
	host, portText, err := net.SplitHostPort(strings.TrimSpace(addr))
	if err != nil {
		return "", 0, err
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port <= 0 || port > 65535 {
		return "", 0, fmt.Errorf("端口不正确")
	}
	if host == "" {
		host = "127.0.0.1"
	}
	return host, port, nil
}
