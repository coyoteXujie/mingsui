package main

import (
	"errors"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/coyoteXujie/mingsui/internal/config"
)

func TestDesktopUsesClientDefaultConfigPath(t *testing.T) {
	if got := defaultDesktopConfigPath(); got != config.DefaultClientPath() {
		t.Fatalf("defaultDesktopConfigPath() = %q, want CLI client path %q", got, config.DefaultClientPath())
	}
}

func TestBrowserCommand(t *testing.T) {
	name, args := browserCommand("http://127.0.0.1:18200")
	if name == "" {
		t.Fatal("browserCommand() name is empty")
	}
	if len(args) == 0 {
		t.Fatal("browserCommand() args is empty")
	}
	if runtime.GOOS != "windows" && args[len(args)-1] != "http://127.0.0.1:18200" {
		t.Fatalf("browserCommand() args = %v, want URL as last arg", args)
	}
}

func TestFirstAppBrowserCommandUsesAvailableBrowser(t *testing.T) {
	oldLookPath := lookPath
	lookPath = func(name string) (string, error) {
		if name == "chromium" {
			return "/usr/bin/chromium", nil
		}
		return "", errors.New("not found")
	}
	defer func() {
		lookPath = oldLookPath
	}()

	spec, ok := firstAppBrowserCommand([]string{"google-chrome", "chromium"}, "http://127.0.0.1:18200", func(name, url string) (desktopWindowSpec, error) {
		return desktopWindowSpec{Name: name, Args: []string{"--app=" + url}}, nil
	})
	if !ok || spec.Name != "chromium" {
		t.Fatalf("firstAppBrowserCommand() = %+v, %v; want chromium true", spec, ok)
	}
	if len(spec.Args) == 0 || spec.Args[0] != "--app=http://127.0.0.1:18200" {
		t.Fatalf("args = %v, want --app URL", spec.Args)
	}
}

func TestFirstAppBrowserCommandReportsMissingBrowser(t *testing.T) {
	oldLookPath := lookPath
	lookPath = func(name string) (string, error) {
		return "", errors.New("not found")
	}
	defer func() {
		lookPath = oldLookPath
	}()

	spec, ok := firstAppBrowserCommand([]string{"google-chrome"}, "http://127.0.0.1:18200", func(name, url string) (desktopWindowSpec, error) {
		return desktopWindowSpec{Name: name, Args: []string{"--app=" + url}}, nil
	})
	if ok || spec.Name != "" || spec.Args != nil {
		t.Fatalf("firstAppBrowserCommand() = %+v, %v; want missing", spec, ok)
	}
}

func TestOpenDesktopWindowReportsMissingHost(t *testing.T) {
	oldLookPath := lookPath
	lookPath = func(name string) (string, error) {
		return "", errors.New("not found")
	}
	defer func() {
		lookPath = oldLookPath
	}()

	if _, err := openDesktopWindow("http://127.0.0.1:18200"); !errors.Is(err, errNoDesktopWindowHost) {
		t.Fatalf("openDesktopWindow() error = %v, want errNoDesktopWindowHost", err)
	}
}

func TestNewLinuxBrowserSpecUsesIsolatedProfile(t *testing.T) {
	spec, err := newLinuxBrowserSpec("google-chrome", "http://127.0.0.1:18200")
	if err != nil {
		t.Fatalf("newLinuxBrowserSpec() error = %v", err)
	}
	if spec.Name != "google-chrome" || !spec.Monitor {
		t.Fatalf("spec = %+v, want monitored google-chrome spec", spec)
	}
	var profileDir string
	for _, arg := range spec.Args {
		if strings.HasPrefix(arg, "--user-data-dir=") {
			profileDir = strings.TrimPrefix(arg, "--user-data-dir=")
			break
		}
	}
	if profileDir == "" {
		t.Fatalf("args = %v, want --user-data-dir", spec.Args)
	}
	if _, err := os.Stat(profileDir); err != nil {
		t.Fatalf("profile dir stat error = %v", err)
	}
	spec.Cleanup()
	if _, err := os.Stat(profileDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("profile dir after cleanup error = %v, want not exist", err)
	}
}
