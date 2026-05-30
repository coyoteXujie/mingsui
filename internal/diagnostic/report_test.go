package diagnostic

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestReportTracksFailedChecks(t *testing.T) {
	report := NewReport("/tmp/client.json")
	report.AddWarning("默认 token")
	report.AddCheck(Check{Name: "local", Label: "本地监听", OK: true})
	report.AddCheck(Check{Name: "relay", Label: "relay", OK: false, Error: "dial failed"})

	if report.OK {
		t.Fatal("report.OK = true, want false")
	}
	if report.ExitCode() != 1 {
		t.Fatalf("ExitCode() = %d, want 1", report.ExitCode())
	}
	if len(report.Warnings) != 1 {
		t.Fatalf("Warnings = %v, want one warning", report.Warnings)
	}
}

func TestWriteJSON(t *testing.T) {
	report := NewReport("/tmp/relay.json")
	report.AddCheck(Check{Name: "tls", Label: "TLS", OK: true, Skipped: true})

	var buf bytes.Buffer
	if err := WriteJSON(&buf, report); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	var got Report
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got.ConfigPath != report.ConfigPath {
		t.Fatalf("ConfigPath = %q, want %q", got.ConfigPath, report.ConfigPath)
	}
	if len(got.Checks) != 1 || !got.Checks[0].Skipped {
		t.Fatalf("Checks = %+v, want skipped check", got.Checks)
	}
}
