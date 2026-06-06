package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/coyoteXujie/mingsui/internal/config"
	"github.com/coyoteXujie/mingsui/internal/desktop"
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

func TestDesktopServiceURLNormalizesListenAddr(t *testing.T) {
	tests := []struct {
		name string
		addr string
		want string
		ok   bool
	}{
		{name: "loopback", addr: "127.0.0.1:18200", want: "http://127.0.0.1:18200", ok: true},
		{name: "wildcard", addr: "0.0.0.0:18200", want: "http://127.0.0.1:18200", ok: true},
		{name: "empty host", addr: ":18200", want: "http://127.0.0.1:18200", ok: true},
		{name: "ipv6", addr: "[::1]:18200", want: "http://[::1]:18200", ok: true},
		{name: "ephemeral", addr: "127.0.0.1:0", ok: false},
		{name: "invalid", addr: "127.0.0.1", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := desktopServiceURL(tt.addr)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("desktopServiceURL(%q) = %q, %v; want %q, %v", tt.addr, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestDesktopServiceAvailable(t *testing.T) {
	oldClient := desktopHTTPClient
	defer func() {
		desktopHTTPClient = oldClient
	}()

	desktopHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/state" {
			return &http.Response{StatusCode: http.StatusNotFound, Body: http.NoBody}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	})}

	if !desktopServiceAvailable(context.Background(), "http://127.0.0.1:18200") {
		t.Fatal("desktopServiceAvailable() = false, want true")
	}
	if desktopServiceAvailable(context.Background(), "http://127.0.0.1:18200/missing") {
		t.Fatal("desktopServiceAvailable(missing) = true, want false")
	}
}

func TestWithDesktopLogs(t *testing.T) {
	logs := desktop.NewLogBuffer(10)
	if _, err := logs.Write([]byte("first\nsecond\n")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	handler := withDesktopLogs(http.NotFoundHandler(), logs)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/logs", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got logsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(got.Logs) != 2 || got.Logs[0] != "first" || got.Logs[1] != "second" {
		t.Fatalf("Logs = %+v, want first/second", got.Logs)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
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

func TestNewDetachedLinuxBrowserSpecDoesNotUseIsolatedProfile(t *testing.T) {
	spec, err := newDetachedLinuxBrowserSpec("google-chrome", "http://127.0.0.1:18200")
	if err != nil {
		t.Fatalf("newDetachedLinuxBrowserSpec() error = %v", err)
	}
	if spec.Name != "google-chrome" || spec.Monitor || spec.Cleanup != nil {
		t.Fatalf("spec = %+v, want detached google-chrome spec", spec)
	}
	for _, arg := range spec.Args {
		if strings.HasPrefix(arg, "--user-data-dir=") {
			t.Fatalf("args = %v, detached spec should not use temporary profile", spec.Args)
		}
	}
}
