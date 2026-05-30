package protocol

import (
	"bytes"
	"strings"
	"testing"
)

func TestReadWriteJSON(t *testing.T) {
	var buf bytes.Buffer
	want := ConnectRequest{
		Version: Version,
		Token:   "token",
		Network: "tcp",
		Address: "example.com:443",
	}

	if err := WriteJSON(&buf, want); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	var got ConnectRequest
	if err := ReadJSON(&buf, &got); err != nil {
		t.Fatalf("ReadJSON() error = %v", err)
	}

	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestConnectRequestEffectiveCommand(t *testing.T) {
	if got := (ConnectRequest{}).EffectiveCommand(); got != CommandConnect {
		t.Fatalf("EffectiveCommand() = %q, want %q", got, CommandConnect)
	}
	if got := (ConnectRequest{Command: CommandHealth}).EffectiveCommand(); got != CommandHealth {
		t.Fatalf("EffectiveCommand() = %q, want %q", got, CommandHealth)
	}
}

func TestReadWriteJSONWithMetrics(t *testing.T) {
	var buf bytes.Buffer
	want := ConnectResponse{
		Version: Version,
		OK:      true,
		Metrics: &Metrics{
			ActiveConnections: 1,
			TotalConnections:  2,
			UploadBytes:       3,
			DownloadBytes:     4,
		},
	}

	if err := WriteJSON(&buf, want); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	var got ConnectResponse
	if err := ReadJSON(&buf, &got); err != nil {
		t.Fatalf("ReadJSON() error = %v", err)
	}
	if got.Metrics == nil {
		t.Fatal("got Metrics = nil, want metrics")
	}
	if *got.Metrics != *want.Metrics {
		t.Fatalf("got metrics %+v, want %+v", *got.Metrics, *want.Metrics)
	}
}

func TestReadJSONRejectsLargeMessages(t *testing.T) {
	var buf bytes.Buffer
	buf.Write([]byte{0, 2, 0, 1})

	var got ConnectRequest
	err := ReadJSON(&buf, &got)
	if err == nil {
		t.Fatal("ReadJSON() error = nil, want large message error")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Fatalf("ReadJSON() error = %v, want too large", err)
	}
}
