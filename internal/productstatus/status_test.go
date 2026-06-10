package productstatus

import (
	"strings"
	"testing"

	"github.com/coyoteXujie/mingsui/internal/config"
)

func TestEvaluateDefaultConfigNeedsSetup(t *testing.T) {
	status := Evaluate(config.DefaultClient(), Options{AutoProfile: true, ConfigPath: "/tmp/client.json"})

	if !status.OK {
		t.Fatalf("OK = false, want true for syntactically usable default config")
	}
	if status.Readiness != "needs_setup" {
		t.Fatalf("Readiness = %q, want needs_setup", status.Readiness)
	}
	if status.Mode != "relay" || status.DefaultToken != true {
		t.Fatalf("Status = %+v, want relay mode with default token warning", status)
	}
	if !hasAction(status, "import") || !hasAction(status, "token") {
		t.Fatalf("Actions = %+v, want import and token guidance", status.Actions)
	}
}

func TestEvaluateProxyProfile(t *testing.T) {
	cfg := config.DefaultClient()
	cfg.ProxyProfiles = []config.ProxyProfile{
		{Name: "future", Protocol: "tuic", URL: "tuic://00000000-0000-0000-0000-000000000000:pass@example.com:443#future"},
		{Name: "tokyo", Protocol: "ss", URL: "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#tokyo"},
	}

	status := Evaluate(cfg, Options{AutoProfile: true})

	if !status.OK || status.Mode != "proxy" || status.SelectedProxy != "tokyo" {
		t.Fatalf("Status = %+v, want ready proxy tokyo", status)
	}
	if !hasAction(status, "connect") || !hasAction(status, "env") || !hasAction(status, "exec") {
		t.Fatalf("Actions = %+v, want connect/env/exec guidance", status.Actions)
	}
}

func TestEvaluateProxyProfileBlocksUnsupportedActiveSelection(t *testing.T) {
	cfg := config.DefaultClient()
	cfg.ProxyProfiles = []config.ProxyProfile{
		{Name: "future", Protocol: "tuic", URL: "tuic://00000000-0000-0000-0000-000000000000:pass@example.com:443#future"},
	}
	cfg.ActiveProxyProfile = "future"

	status := Evaluate(cfg, Options{AutoProfile: true})

	if status.OK || status.Readiness != "blocked" || status.SelectedProxy != "future" {
		t.Fatalf("Status = %+v, want blocked unsupported proxy", status)
	}
	if !strings.Contains(status.Message, "暂不支持") {
		t.Fatalf("Message = %q, want unsupported guidance", status.Message)
	}
}

func TestEvaluateWarnsOnExposedLocalProxy(t *testing.T) {
	cfg := config.DefaultClient()
	cfg.LocalAddr = "0.0.0.0:18080"
	cfg.HTTPAddr = "0.0.0.0:18081"
	cfg.Token = "secret"

	status := Evaluate(cfg, Options{AutoProfile: true})

	if !status.LocalProxyExposed {
		t.Fatalf("LocalProxyExposed = false, want true")
	}
	if !hasAction(status, "local_auth") {
		t.Fatalf("Actions = %+v, want local auth guidance", status.Actions)
	}
}

func hasAction(status Status, id string) bool {
	for _, action := range status.Actions {
		if action.ID == id {
			return true
		}
	}
	return false
}
