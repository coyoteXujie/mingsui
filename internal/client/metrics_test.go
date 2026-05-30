package client

import "testing"

func TestMetricsRecorderSnapshot(t *testing.T) {
	var recorder metricsRecorder
	recorder.OpenConnection()
	recorder.OpenConnection()
	recorder.AddUploadBytes(12)
	recorder.AddUploadBytes(-1)
	recorder.AddDownloadBytes(34)
	recorder.CloseConnection()

	got := recorder.Snapshot()
	if got.ActiveConnections != 1 {
		t.Fatalf("ActiveConnections = %d, want 1", got.ActiveConnections)
	}
	if got.TotalConnections != 2 {
		t.Fatalf("TotalConnections = %d, want 2", got.TotalConnections)
	}
	if got.UploadBytes != 12 {
		t.Fatalf("UploadBytes = %d, want 12", got.UploadBytes)
	}
	if got.DownloadBytes != 34 {
		t.Fatalf("DownloadBytes = %d, want 34", got.DownloadBytes)
	}
}
