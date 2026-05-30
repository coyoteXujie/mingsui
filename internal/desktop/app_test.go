package desktop

import (
	"context"
	"io"
	"log"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/coyoteXujie/mingsui/internal/config"
)

func TestNewAppUsesDefaultConfigWhenFileMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "client.json")
	app, err := NewApp(path, testLogger())
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	if app.ConfigPath() != path {
		t.Fatalf("ConfigPath() = %q, want %q", app.ConfigPath(), path)
	}
	if got := app.Config(); !reflect.DeepEqual(got, config.DefaultClient()) {
		t.Fatalf("Config() = %+v, want default %+v", got, config.DefaultClient())
	}
}

func TestAppSaveConfigAndReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "client.json")
	app, err := NewApp(path, testLogger())
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	want := config.DefaultClient()
	want.Token = "secret"
	want.RelayAddr = "relay.example.com:9443"
	if err := app.SaveConfig(want); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	reloaded, err := NewApp(path, testLogger())
	if err != nil {
		t.Fatalf("NewApp(reload) error = %v", err)
	}
	if got := reloaded.Config(); !reflect.DeepEqual(got, want) {
		t.Fatalf("reloaded Config() = %+v, want %+v", got, want)
	}
}

func TestAppStopIsNoopWhenNotRunning(t *testing.T) {
	app, err := NewApp(filepath.Join(t.TempDir(), "client.json"), testLogger())
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	if err := app.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func testLogger() *log.Logger {
	return log.New(io.Discard, "", 0)
}
