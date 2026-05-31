package systemproxy

import (
	"context"
	"reflect"
	"testing"
)

type fakeRunner struct {
	output []byte
	calls  [][]string
}

func (r *fakeRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	call := append([]string{name}, args...)
	r.calls = append(r.calls, call)
	return r.output, nil
}

func TestSplitAddr(t *testing.T) {
	host, port, err := splitAddr("127.0.0.1:18081")
	if err != nil {
		t.Fatalf("splitAddr() error = %v", err)
	}
	if host != "127.0.0.1" || port != 18081 {
		t.Fatalf("splitAddr() = %s %d, want 127.0.0.1 18081", host, port)
	}
}

func TestEnableBuildsGSettingsCommands(t *testing.T) {
	run := &fakeRunner{}
	err := enable(context.Background(), Config{HTTPAddr: "127.0.0.1:18081", SOCKSAddr: "127.0.0.1:18080"}, run)
	if err != nil {
		t.Fatalf("enable() error = %v", err)
	}
	wantFirst := []string{"gsettings", "set", "org.gnome.system.proxy", "mode", "manual"}
	if !reflect.DeepEqual(run.calls[0], wantFirst) {
		t.Fatalf("first call = %+v, want %+v", run.calls[0], wantFirst)
	}
	wantLast := []string{"gsettings", "set", "org.gnome.system.proxy.socks", "port", "18080"}
	if !reflect.DeepEqual(run.calls[len(run.calls)-1], wantLast) {
		t.Fatalf("last call = %+v, want %+v", run.calls[len(run.calls)-1], wantLast)
	}
}

func TestCurrentStatusManual(t *testing.T) {
	run := &fakeRunner{output: []byte("'manual'\n")}
	status := currentStatus(context.Background(), run)
	if !status.Supported || !status.Enabled || status.Mode != "manual" {
		t.Fatalf("status = %+v, want supported manual enabled", status)
	}
}
