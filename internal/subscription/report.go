package subscription

import (
	"strings"

	"github.com/coyoteXujie/mingsui/internal/config"
	"github.com/coyoteXujie/mingsui/internal/mihomo"
)

const (
	SyncKindRelay = "relay"
	SyncKindProxy = "proxy"
)

type SyncReport struct {
	Name                                string   `json:"name"`
	Kind                                string   `json:"kind"`
	Imported                            int      `json:"imported"`
	RelayProfiles                       int      `json:"relay_profiles"`
	ProxyProfiles                       int      `json:"proxy_profiles"`
	ExportableProxyProfiles             int      `json:"exportable_proxy_profiles"`
	AutoSelectableProxyProfiles         int      `json:"auto_selectable_proxy_profiles"`
	ImportedExportableProxyProfiles     int      `json:"imported_exportable_proxy_profiles"`
	ImportedAutoSelectableProxyProfiles int      `json:"imported_auto_selectable_proxy_profiles"`
	ActiveProfile                       string   `json:"active_profile,omitempty"`
	ActiveProxyProfile                  string   `json:"active_proxy_profile,omitempty"`
	Selected                            string   `json:"selected,omitempty"`
	Warnings                            []string `json:"warnings,omitempty"`
	Message                             string   `json:"message"`
}

type SyncReportInput struct {
	Name                  string
	Kind                  string
	Message               string
	Imported              int
	ImportedRelayProfiles []config.RelayProfile
	ImportedProxyProfiles []config.ProxyProfile
	Config                config.ClientConfig
}

func BuildSyncReport(input SyncReportInput) SyncReport {
	cfg := input.Config
	report := SyncReport{
		Name:               strings.TrimSpace(input.Name),
		Kind:               strings.TrimSpace(input.Kind),
		Imported:           input.Imported,
		RelayProfiles:      len(cfg.Profiles),
		ProxyProfiles:      len(cfg.ProxyProfiles),
		ActiveProfile:      strings.TrimSpace(cfg.ActiveProfile),
		ActiveProxyProfile: strings.TrimSpace(cfg.ActiveProxyProfile),
	}
	for _, profile := range cfg.ProxyProfiles {
		if mihomo.CanExportProfile(profile) {
			report.ExportableProxyProfiles++
		}
		if mihomo.CanAutoSelectProfile(profile) {
			report.AutoSelectableProxyProfiles++
		}
	}
	for _, profile := range input.ImportedProxyProfiles {
		if mihomo.CanExportProfile(profile) {
			report.ImportedExportableProxyProfiles++
		}
		if mihomo.CanAutoSelectProfile(profile) {
			report.ImportedAutoSelectableProxyProfiles++
		}
	}
	if input.Kind == SyncKindProxy && len(input.ImportedProxyProfiles) == 0 && input.Imported > 0 {
		report.ImportedExportableProxyProfiles = report.ExportableProxyProfiles
		report.ImportedAutoSelectableProxyProfiles = report.AutoSelectableProxyProfiles
	}

	switch report.Kind {
	case SyncKindRelay:
		report.Selected = report.ActiveProfile
		if report.Imported == 0 {
			report.Warnings = append(report.Warnings, "订阅没有导入任何 relay profile")
		}
		if report.Selected == "" && report.RelayProfiles > 0 {
			report.Warnings = append(report.Warnings, "已有 relay profile，但当前未选择")
		}
	case SyncKindProxy:
		report.Selected = report.ActiveProxyProfile
		if report.Imported == 0 {
			report.Warnings = append(report.Warnings, "订阅没有导入任何机场节点")
		}
		if report.Imported > 0 && report.ImportedExportableProxyProfiles == 0 {
			report.Warnings = append(report.Warnings, "本次订阅导入了节点，但当前协议暂不支持直接连接")
		}
		if report.ImportedExportableProxyProfiles > 0 && report.ImportedAutoSelectableProxyProfiles == 0 {
			report.Warnings = append(report.Warnings, "本次订阅没有可自动选择的国外节点；可能都是国内/回国线路")
		}
		if report.ExportableProxyProfiles > 0 && report.Selected == "" {
			report.Warnings = append(report.Warnings, "已有可连接节点，但当前未选择节点")
		}
	default:
		report.Warnings = append(report.Warnings, "未知订阅类型")
	}
	if strings.TrimSpace(input.Message) != "" {
		report.Message = strings.TrimSpace(input.Message)
	} else {
		report.Message = syncReportMessage(report)
	}
	return report
}

func syncReportMessage(report SyncReport) string {
	switch report.Kind {
	case SyncKindRelay:
		if report.Selected != "" {
			return "订阅已同步，当前 relay profile: " + report.Selected
		}
		return "订阅已同步 relay profile"
	case SyncKindProxy:
		if report.Selected != "" {
			return "订阅已同步机场节点，当前节点: " + report.Selected
		}
		return "订阅已同步机场节点"
	default:
		return "订阅已同步"
	}
}
