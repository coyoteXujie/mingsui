package mihomo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/coyoteXujie/mingsui/internal/config"
)

type RuntimeStatus struct {
	Running    bool      `json:"running"`
	LocalAddr  string    `json:"local_addr"`
	HTTPAddr   string    `json:"http_addr,omitempty"`
	BinaryPath string    `json:"binary_path,omitempty"`
	ConfigPath string    `json:"config_path,omitempty"`
	WorkDir    string    `json:"work_dir,omitempty"`
	StartedAt  time.Time `json:"started_at,omitempty"`
	LastError  string    `json:"last_error,omitempty"`
}

type Controller struct {
	mu        sync.Mutex
	cfg       config.ClientConfig
	opts      Options
	cancel    context.CancelFunc
	done      chan error
	startedAt time.Time
	lastError string
	binary    string
	workDir   string
	config    string
}

func NewController(cfg config.ClientConfig, opts Options) *Controller {
	return &Controller{cfg: cfg, opts: opts}
}

func (c *Controller) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancel != nil {
		return errors.New("Mihomo 内核已经在运行")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	prepared, err := Prepare(c.cfg, c.opts)
	if err != nil {
		c.lastError = err.Error()
		return err
	}

	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	c.cancel = cancel
	c.done = done
	c.startedAt = time.Now()
	c.lastError = ""
	c.binary = prepared.BinaryPath
	c.workDir = prepared.WorkDir
	c.config = prepared.ConfigPath
	opts := c.opts

	go func() {
		err := prepared.Run(runCtx, opts)
		c.mu.Lock()
		if err != nil && runCtx.Err() == nil {
			c.lastError = err.Error()
		}
		c.cancel = nil
		c.done = nil
		c.mu.Unlock()
		done <- err
	}()

	return nil
}

func (c *Controller) Stop(ctx context.Context) error {
	c.mu.Lock()
	cancel := c.cancel
	done := c.done
	c.mu.Unlock()

	if cancel == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	cancel()

	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Controller) Status() RuntimeStatus {
	c.mu.Lock()
	defer c.mu.Unlock()
	return RuntimeStatus{
		Running:    c.cancel != nil,
		LocalAddr:  c.cfg.LocalAddr,
		HTTPAddr:   c.cfg.HTTPAddr,
		BinaryPath: c.binary,
		ConfigPath: c.config,
		WorkDir:    c.workDir,
		StartedAt:  c.startedAt,
		LastError:  c.lastError,
	}
}

type Runtime struct {
	BinaryPath string
	ConfigPath string
	WorkDir    string
}

func Prepare(cfg config.ClientConfig, opts Options) (Runtime, error) {
	binary, err := ResolveBinary(opts.BinaryPath)
	if err != nil {
		return Runtime{}, err
	}
	workDir := opts.WorkDir
	if workDir == "" {
		workDir = filepath.Join(os.TempDir(), "mingsui-mihomo")
	}
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		return Runtime{}, fmt.Errorf("创建 Mihomo 工作目录失败: %w", err)
	}
	data, err := Generate(cfg, opts)
	if err != nil {
		return Runtime{}, err
	}
	configPath := filepath.Join(workDir, "config.yaml")
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		return Runtime{}, fmt.Errorf("写入 Mihomo 配置失败: %w", err)
	}
	return Runtime{BinaryPath: binary, ConfigPath: configPath, WorkDir: workDir}, nil
}

func Run(ctx context.Context, cfg config.ClientConfig, opts Options) error {
	runtime, err := Prepare(cfg, opts)
	if err != nil {
		return err
	}
	return runtime.Run(ctx, opts)
}

func (r Runtime) Run(ctx context.Context, opts Options) error {
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := exec.CommandContext(ctx, r.BinaryPath, "-d", r.WorkDir, "-f", r.ConfigPath)
	cmd.Stdout = writerOrDiscard(opts.Stdout)
	cmd.Stderr = writerOrDiscard(opts.Stderr)
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("Mihomo 内核运行失败: %w", err)
	}
	return nil
}

func ResolveBinary(explicit string) (string, error) {
	if explicit != "" {
		return requireExecutable(explicit)
	}
	if fromEnv := os.Getenv("MINGSUI_MIHOMO_PATH"); fromEnv != "" {
		return requireExecutable(fromEnv)
	}
	for _, path := range bundledBinaryPaths() {
		if isExecutable(path) {
			return path, nil
		}
	}
	for _, name := range []string{"mihomo", "clash-meta", "clash"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	for _, path := range knownSidecarPaths() {
		if isExecutable(path) {
			return path, nil
		}
	}
	return "", errors.New("未找到 Mihomo 内核；请安装 mihomo，或设置 MINGSUI_MIHOMO_PATH 指向内核程序")
}

func bundledBinaryPaths() []string {
	name := "mihomo"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}

	paths := []string{
		filepath.Join("/usr/lib/mingsui", name),
	}
	exe, err := os.Executable()
	if err != nil {
		return paths
	}
	exeDir := filepath.Dir(exe)
	return append([]string{
		filepath.Join(exeDir, name),
		filepath.Join(exeDir, "kernel", name),
		filepath.Join(filepath.Dir(exeDir), "kernel", name),
	}, paths...)
}

func knownSidecarPaths() []string {
	switch runtime.GOOS {
	case "linux":
		return []string{"/opt/clash-party/resources/sidecar/mihomo"}
	default:
		return nil
	}
}

func requireExecutable(path string) (string, error) {
	if isExecutable(path) {
		return path, nil
	}
	return "", fmt.Errorf("Mihomo 内核不可执行或不存在: %s", path)
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode()&0o111 != 0
}

func writerOrDiscard(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
}
