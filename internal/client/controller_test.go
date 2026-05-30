package client

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/coyoteXujie/mingsui/internal/config"
)

func TestControllerStartStop(t *testing.T) {
	controller := newTestController(func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	})

	if err := controller.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	status := controller.Status()
	if !status.Running {
		t.Fatal("Status().Running = false, want true")
	}
	if status.LocalAddr == "" || status.RelayAddr == "" {
		t.Fatalf("Status() missing addresses: %+v", status)
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := controller.Stop(stopCtx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if controller.Status().Running {
		t.Fatal("Status().Running = true after Stop, want false")
	}
}

func TestControllerRejectsDoubleStart(t *testing.T) {
	controller := newTestController(func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	})
	if err := controller.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() {
		_ = controller.Stop(context.Background())
	}()

	err := controller.Start(context.Background())
	if err == nil {
		t.Fatal("Start() second call error = nil, want running error")
	}
	if !strings.Contains(err.Error(), "已经在运行") {
		t.Fatalf("Start() second call error = %v, want running error", err)
	}
}

func TestControllerRecordsServeError(t *testing.T) {
	wantErr := errors.New("启动失败")
	controller := newTestController(func(ctx context.Context) error {
		return wantErr
	})

	if err := controller.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		status := controller.Status()
		if !status.Running {
			if status.LastError != wantErr.Error() {
				t.Fatalf("LastError = %q, want %q", status.LastError, wantErr.Error())
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("controller still running after serve returned error")
}

func TestControllerStatusIncludesMetrics(t *testing.T) {
	controller := newTestController(func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	})
	controller.metrics = func() RuntimeMetrics {
		return RuntimeMetrics{
			ActiveConnections: 1,
			TotalConnections:  2,
			UploadBytes:       3,
			DownloadBytes:     4,
		}
	}

	got := controller.Status().Metrics
	if got.ActiveConnections != 1 || got.TotalConnections != 2 || got.UploadBytes != 3 || got.DownloadBytes != 4 {
		t.Fatalf("Status().Metrics = %+v, want populated metrics", got)
	}
}

func newTestController(serve func(context.Context) error) *Controller {
	cfg := config.DefaultClient()
	cfg.HTTPAddr = "127.0.0.1:18081"
	return &Controller{
		cfg:     cfg,
		serve:   serve,
		metrics: func() RuntimeMetrics { return RuntimeMetrics{} },
	}
}
