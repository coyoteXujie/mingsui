package deploy

import (
	"strings"
	"testing"
)

func TestRenderRelaySystemd(t *testing.T) {
	unit, err := RenderRelaySystemd(SystemdRelayOptions{
		BinaryPath: "/usr/local/bin/mingsui-relay",
		ConfigPath: "/etc/mingsui/relay.json",
		User:       "relay",
		Group:      "relay",
		WorkingDir: "/var/lib/relay",
	})
	if err != nil {
		t.Fatalf("RenderRelaySystemd() error = %v", err)
	}

	wants := []string{
		"Description=MingSui Relay",
		"User=relay",
		"Group=relay",
		"WorkingDirectory=/var/lib/relay",
		"ExecStart=/usr/local/bin/mingsui-relay serve -config /etc/mingsui/relay.json",
		"Restart=on-failure",
	}
	for _, want := range wants {
		if !strings.Contains(unit, want) {
			t.Fatalf("unit does not contain %q:\n%s", want, unit)
		}
	}
}

func TestRenderRelaySystemdRequiresBinaryPath(t *testing.T) {
	_, err := RenderRelaySystemd(SystemdRelayOptions{ConfigPath: "/etc/mingsui/relay.json"})
	if err == nil {
		t.Fatal("RenderRelaySystemd() error = nil, want binary path error")
	}
}

func TestRenderRelaySystemdRequiresConfigPath(t *testing.T) {
	_, err := RenderRelaySystemd(SystemdRelayOptions{BinaryPath: "/usr/local/bin/mingsui-relay"})
	if err == nil {
		t.Fatal("RenderRelaySystemd() error = nil, want config path error")
	}
}

func TestRenderRelaySystemdDefaultsGroupToUser(t *testing.T) {
	unit, err := RenderRelaySystemd(SystemdRelayOptions{
		BinaryPath: "/usr/local/bin/mingsui-relay",
		ConfigPath: "/etc/mingsui/relay.json",
		User:       "relay",
	})
	if err != nil {
		t.Fatalf("RenderRelaySystemd() error = %v", err)
	}
	if !strings.Contains(unit, "Group=relay") {
		t.Fatalf("unit does not default group to user:\n%s", unit)
	}
}
