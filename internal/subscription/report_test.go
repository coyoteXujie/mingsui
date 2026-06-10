package subscription

import (
	"strings"
	"testing"

	"github.com/coyoteXujie/mingsui/internal/config"
)

func TestBuildSyncReportUsesImportedProxyProfilesForWarnings(t *testing.T) {
	cfg := config.DefaultClient()
	cfg.ProxyProfiles = []config.ProxyProfile{
		{Name: "tokyo", Protocol: "ss", URL: "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4Mzg4#tokyo"},
		{Name: "future", Protocol: "tuic", URL: "tuic://00000000-0000-0000-0000-000000000000:pass@example.com:443#future"},
	}
	cfg.ActiveProxyProfile = "tokyo"

	report := BuildSyncReport(SyncReportInput{
		Name:     "airport",
		Kind:     SyncKindProxy,
		Imported: 1,
		ImportedProxyProfiles: []config.ProxyProfile{
			{Name: "future", Protocol: "tuic", URL: "tuic://00000000-0000-0000-0000-000000000000:pass@example.com:443#future"},
		},
		Config: cfg,
	})

	if report.ExportableProxyProfiles != 1 || report.ImportedExportableProxyProfiles != 0 {
		t.Fatalf("report = %+v, want old total exportable but imported unsupported", report)
	}
	if !strings.Contains(strings.Join(report.Warnings, "\n"), "本次订阅导入了节点") {
		t.Fatalf("Warnings = %+v, want imported unsupported warning", report.Warnings)
	}
}
