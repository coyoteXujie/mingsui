package diagnostic

import (
	"encoding/json"
	"io"

	"github.com/coyoteXujie/mingsui/internal/protocol"
)

type Report struct {
	OK         bool     `json:"ok"`
	ConfigPath string   `json:"config_path,omitempty"`
	Error      string   `json:"error,omitempty"`
	Warnings   []string `json:"warnings,omitempty"`
	Checks     []Check  `json:"checks,omitempty"`
}

type Check struct {
	Name        string              `json:"name"`
	Label       string              `json:"label"`
	Target      string              `json:"target,omitempty"`
	OK          bool                `json:"ok"`
	Skipped     bool                `json:"skipped,omitempty"`
	Error       string              `json:"error,omitempty"`
	Warning     string              `json:"warning,omitempty"`
	Metrics     *protocol.Metrics   `json:"metrics,omitempty"`
	Certificate *CertificateSummary `json:"certificate,omitempty"`
}

type CertificateSummary struct {
	Hosts            []string `json:"hosts,omitempty"`
	NotBefore        string   `json:"not_before"`
	NotAfter         string   `json:"not_after"`
	ExpiresInSeconds int64    `json:"expires_in_seconds"`
}

func NewReport(configPath string) Report {
	return Report{
		OK:         true,
		ConfigPath: configPath,
	}
}

func (r *Report) AddWarning(warning string) {
	if warning != "" {
		r.Warnings = append(r.Warnings, warning)
	}
}

func (r *Report) AddCheck(check Check) {
	r.Checks = append(r.Checks, check)
	if !check.OK {
		r.OK = false
	}
}

func (r *Report) Fail(message string) {
	r.OK = false
	r.Error = message
}

func (r Report) ExitCode() int {
	if r.OK {
		return 0
	}
	return 1
}

func WriteJSON(w io.Writer, report Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}
