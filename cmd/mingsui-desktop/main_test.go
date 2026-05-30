package main

import (
	"runtime"
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
