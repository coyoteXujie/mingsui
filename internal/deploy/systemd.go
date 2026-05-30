package deploy

import (
	"bytes"
	"fmt"
	"text/template"
)

type SystemdRelayOptions struct {
	Description string
	BinaryPath  string
	ConfigPath  string
	User        string
	Group       string
	WorkingDir  string
}

func RenderRelaySystemd(options SystemdRelayOptions) (string, error) {
	if options.BinaryPath == "" {
		return "", fmt.Errorf("二进制路径不能为空")
	}
	if options.ConfigPath == "" {
		return "", fmt.Errorf("配置路径不能为空")
	}
	if options.Description == "" {
		options.Description = "MingSui Relay"
	}
	if options.User == "" {
		options.User = "mingsui"
	}
	if options.Group == "" {
		options.Group = options.User
	}
	if options.WorkingDir == "" {
		options.WorkingDir = "/var/lib/mingsui"
	}

	var buf bytes.Buffer
	if err := relaySystemdTemplate.Execute(&buf, options); err != nil {
		return "", err
	}
	return buf.String(), nil
}

var relaySystemdTemplate = template.Must(template.New("relay-systemd").Parse(`[Unit]
Description={{ .Description }}
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User={{ .User }}
Group={{ .Group }}
WorkingDirectory={{ .WorkingDir }}
ExecStart={{ .BinaryPath }} serve -config {{ .ConfigPath }}
Restart=on-failure
RestartSec=3
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ProtectHome=true
ReadWritePaths={{ .WorkingDir }}

[Install]
WantedBy=multi-user.target
`))
