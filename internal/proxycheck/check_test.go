package proxycheck

import (
	"errors"
	"testing"
	"time"

	"github.com/coyoteXujie/mingsui/internal/config"
)

func TestReportBestUsesLowestSuccessfulLatency(t *testing.T) {
	report := Report{Results: []Result{
		{Name: "tokyo", OK: true, LatencyMS: 180},
		{Name: "broken", OK: false, LatencyMS: 20},
		{Name: "osaka", OK: true, LatencyMS: 90},
	}}

	best, ok := report.Best()
	if !ok || best.Name != "osaka" {
		t.Fatalf("Best() = %+v, %v, want osaka true", best, ok)
	}
}

func TestCheckSkipsUnsupportedAndMainlandWithoutStartingKernel(t *testing.T) {
	cfg := config.DefaultClient()
	cfg.ProxyProfiles = []config.ProxyProfile{
		{Name: "future", Protocol: "tuic", URL: "tuic://00000000-0000-0000-0000-000000000000:pass@example.com:443#future"},
		{Name: "中国大陆", Protocol: "ss", URL: "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#cn"},
	}

	report, err := Check(nil, cfg, Options{Timeout: time.Millisecond})
	if !errors.Is(err, ErrNoCandidates) {
		t.Fatalf("Check() error = %v, want ErrNoCandidates", err)
	}
	if len(report.Results) != 2 {
		t.Fatalf("Results = %+v, want 2 skipped results", report.Results)
	}
	if report.Results[0].SkipReason != "暂不支持直接连接" || report.Results[1].SkipReason != "国内节点不自动选择" {
		t.Fatalf("Results = %+v, want unsupported and mainland skip reasons", report.Results)
	}
}

func TestCheckRejectsInvalidTargetURL(t *testing.T) {
	cfg := config.DefaultClient()
	report, err := Check(nil, cfg, Options{TargetURL: "ftp://example.com/file"})
	if err == nil {
		t.Fatal("Check() error = nil, want invalid URL error")
	}
	if report.Error == "" {
		t.Fatalf("Report = %+v, want error", report)
	}
}
