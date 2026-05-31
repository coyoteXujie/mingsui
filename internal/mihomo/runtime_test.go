package mihomo

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/coyoteXujie/mingsui/internal/config"
)

func TestResolveBinaryUsesExplicitPath(t *testing.T) {
	path := fakeExecutable(t)
	got, err := ResolveBinary(path)
	if err != nil {
		t.Fatalf("ResolveBinary() error = %v", err)
	}
	if got != path {
		t.Fatalf("ResolveBinary() = %q, want %q", got, path)
	}
}

func TestBundledBinaryPathsIncludesExecutableDir(t *testing.T) {
	paths := bundledBinaryPaths()
	if len(paths) == 0 {
		t.Fatal("bundledBinaryPaths() returned no paths")
	}
	wantName := "mihomo"
	if filepath.Ext(paths[0]) == ".exe" {
		wantName = "mihomo.exe"
	}
	found := false
	for _, path := range paths {
		if filepath.Base(path) == wantName {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("bundledBinaryPaths() = %+v, want basename %s", paths, wantName)
	}
}

func TestResolveBinaryRejectsMissingEnvPath(t *testing.T) {
	t.Setenv("MINGSUI_MIHOMO_PATH", filepath.Join(t.TempDir(), "missing"))
	_, err := ResolveBinary("")
	if err == nil {
		t.Fatal("ResolveBinary() error = nil, want missing env path error")
	}
}

func TestPrepareWritesConfig(t *testing.T) {
	binary := fakeExecutable(t)
	cfg := config.DefaultClient()
	cfg.HTTPAddr = "127.0.0.1:18081"
	cfg.ProxyProfiles = []config.ProxyProfile{
		{Name: "tokyo", Protocol: "ss", URL: "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#tokyo"},
	}
	cfg.ActiveProxyProfile = "tokyo"

	runtime, err := Prepare(cfg, Options{BinaryPath: binary, WorkDir: t.TempDir()})
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if runtime.BinaryPath != binary {
		t.Fatalf("BinaryPath = %q, want %q", runtime.BinaryPath, binary)
	}
	data, err := os.ReadFile(runtime.ConfigPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if len(data) == 0 {
		t.Fatal("config is empty")
	}
}

func TestTestConfigRunsMihomoTest(t *testing.T) {
	binary := fakeShell(t, `case " $* " in *" -t "*) exit 0 ;; *) exit 3 ;; esac`)
	cfg := config.DefaultClient()
	cfg.HTTPAddr = "127.0.0.1:18081"
	cfg.ProxyProfiles = []config.ProxyProfile{
		{Name: "tokyo", Protocol: "ss", URL: "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#tokyo"},
	}
	cfg.ActiveProxyProfile = "tokyo"

	if _, err := TestConfig(context.Background(), cfg, Options{BinaryPath: binary, WorkDir: t.TempDir()}); err != nil {
		t.Fatalf("TestConfig() error = %v", err)
	}
}

func TestRuntimeRunHonorsContextCancel(t *testing.T) {
	binary := fakeShell(t, "sleep 10")
	runtime := Runtime{BinaryPath: binary, WorkDir: t.TempDir(), ConfigPath: filepath.Join(t.TempDir(), "config.yaml")}
	if err := os.WriteFile(runtime.ConfigPath, []byte("mode: rule\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := runtime.Run(ctx, Options{}); err != nil {
		t.Fatalf("Run() error = %v, want nil on canceled context", err)
	}
}

func fakeExecutable(t *testing.T) string {
	t.Helper()
	return fakeShell(t, "exit 0")
}

func fakeShell(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "mihomo")
	data := "#!/bin/sh\n" + body + "\n"
	if err := os.WriteFile(path, []byte(data), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
