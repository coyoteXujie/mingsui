package mihomo

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/coyoteXujie/mingsui/internal/config"
)

type Options struct {
	Mode       string
	LogLevel   string
	BinaryPath string
	WorkDir    string
	Stdout     io.Writer
	Stderr     io.Writer
}

func Generate(cfg config.ClientConfig, opts Options) ([]byte, error) {
	if len(cfg.ProxyProfiles) == 0 {
		return nil, fmt.Errorf("没有可导出的机场节点")
	}
	if opts.Mode == "" {
		opts.Mode = "rule"
	}
	if opts.LogLevel == "" {
		opts.LogLevel = "info"
	}

	selected := selectedProxyName(cfg)
	if selected == "" {
		selected = cfg.ProxyProfiles[0].Name
	}

	var out bytes.Buffer
	writeScalar(&out, "mode", opts.Mode)
	writeScalar(&out, "log-level", opts.LogLevel)
	writeBool(&out, "allow-lan", false)
	writeBindAndPorts(&out, cfg)
	names := make([]string, 0, len(cfg.ProxyProfiles))
	proxies := make([][]byte, 0, len(cfg.ProxyProfiles))
	for _, profile := range cfg.ProxyProfiles {
		proxyYAML, err := proxyToYAML(profile)
		if err != nil {
			if profile.Name == selected {
				return nil, fmt.Errorf("当前选择的机场节点 %s 暂不能导出到 Mihomo: %w", profile.Name, err)
			}
			continue
		}
		names = append(names, profile.Name)
		proxies = append(proxies, proxyYAML)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("没有可导出到 Mihomo 的机场节点，当前支持 ss 和 vmess")
	}
	if selected == "" {
		selected = names[0]
	}

	out.WriteString("proxies:\n")
	for _, proxyYAML := range proxies {
		out.Write(proxyYAML)
	}

	out.WriteString("proxy-groups:\n")
	out.WriteString("  - name: ")
	writeInlineString(&out, "明隧")
	out.WriteString("\n    type: select\n    proxies:\n")
	writeListItem(&out, selected)
	for _, name := range names {
		if name == selected {
			continue
		}
		writeListItem(&out, name)
	}
	writeListItem(&out, "DIRECT")
	out.WriteString("rules:\n")
	out.WriteString("  - MATCH,明隧\n")
	return out.Bytes(), nil
}

func CanExportProfile(profile config.ProxyProfile) bool {
	_, err := proxyToYAML(profile)
	return err == nil
}

func FirstExportableProfileName(profiles []config.ProxyProfile) (string, bool) {
	for _, profile := range profiles {
		if CanExportProfile(profile) {
			return profile.Name, true
		}
	}
	return "", false
}

func selectedProxyName(cfg config.ClientConfig) string {
	name := strings.TrimSpace(cfg.ActiveProxyProfile)
	if name == "" {
		return ""
	}
	if _, ok := cfg.ProxyProfile(name); ok {
		return name
	}
	return ""
}

func writeBindAndPorts(out *bytes.Buffer, cfg config.ClientConfig) {
	host := "127.0.0.1"
	if h, port, ok := splitAddr(cfg.LocalAddr); ok {
		host = h
		writeInt(out, "socks-port", port)
	}
	if h, port, ok := splitAddr(cfg.HTTPAddr); ok {
		if h != "" {
			host = h
		}
		writeInt(out, "port", port)
	}
	writeScalar(out, "bind-address", host)
}

func splitAddr(addr string) (string, int, bool) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "", 0, false
	}
	host, portText, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, false
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port <= 0 || port > 65535 {
		return "", 0, false
	}
	if host == "" {
		host = "127.0.0.1"
	}
	return host, port, true
}

func proxyToYAML(profile config.ProxyProfile) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(profile.Protocol)) {
	case "ss":
		return ssToYAML(profile)
	case "vmess":
		return vmessToYAML(profile)
	default:
		return nil, fmt.Errorf("暂不支持 %s 导出到 Mihomo", profile.Protocol)
	}
}

func ssToYAML(profile config.ProxyProfile) ([]byte, error) {
	parsed, err := parseSS(profile.URL)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	writeProxyHeader(&out, profile.Name, "ss")
	writeIndentedScalar(&out, "server", parsed.Server)
	writeIndentedInt(&out, "port", parsed.Port)
	writeIndentedScalar(&out, "cipher", parsed.Cipher)
	writeIndentedScalar(&out, "password", parsed.Password)
	return out.Bytes(), nil
}

type ssNode struct {
	Cipher   string
	Password string
	Server   string
	Port     int
}

func parseSS(raw string) (ssNode, error) {
	value := strings.TrimSpace(raw)
	if !strings.HasPrefix(strings.ToLower(value), "ss://") {
		return ssNode{}, fmt.Errorf("不是 ss:// 链接")
	}
	withoutScheme := value[len("ss://"):]
	if i := strings.IndexAny(withoutScheme, "#?"); i >= 0 {
		withoutScheme = withoutScheme[:i]
	}

	var credential, serverPort string
	if before, after, ok := strings.Cut(withoutScheme, "@"); ok {
		credential = decodeMaybeBase64(before)
		serverPort = after
	} else {
		decoded := decodeMaybeBase64(withoutScheme)
		before, after, ok := strings.Cut(decoded, "@")
		if !ok {
			return ssNode{}, fmt.Errorf("ss 链接缺少 server")
		}
		credential = before
		serverPort = after
	}

	cipher, password, ok := strings.Cut(credential, ":")
	if !ok {
		return ssNode{}, fmt.Errorf("ss 链接缺少 cipher/password")
	}
	cipher, _ = url.QueryUnescape(cipher)
	password, _ = url.QueryUnescape(password)
	server, port, err := splitServerPort(serverPort)
	if err != nil {
		return ssNode{}, err
	}
	return ssNode{
		Cipher:   cipher,
		Password: password,
		Server:   server,
		Port:     port,
	}, nil
}

func splitServerPort(value string) (string, int, error) {
	value = strings.TrimSpace(value)
	if host, portText, err := net.SplitHostPort(value); err == nil {
		port, err := strconv.Atoi(portText)
		if err != nil {
			return "", 0, fmt.Errorf("端口不正确: %w", err)
		}
		return strings.Trim(host, "[]"), port, nil
	}
	index := strings.LastIndex(value, ":")
	if index <= 0 || index == len(value)-1 {
		return "", 0, fmt.Errorf("节点地址必须是 host:port")
	}
	port, err := strconv.Atoi(value[index+1:])
	if err != nil {
		return "", 0, fmt.Errorf("端口不正确: %w", err)
	}
	return value[:index], port, nil
}

func vmessToYAML(profile config.ProxyProfile) ([]byte, error) {
	node, err := parseVMess(profile.URL)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	writeProxyHeader(&out, profile.Name, "vmess")
	writeIndentedScalar(&out, "server", node.Server)
	writeIndentedInt(&out, "port", node.Port)
	writeIndentedScalar(&out, "uuid", node.UUID)
	if node.AlterID >= 0 {
		writeIndentedInt(&out, "alterId", node.AlterID)
	}
	writeIndentedScalar(&out, "cipher", node.Cipher)
	if node.TLS {
		writeIndentedBool(&out, "tls", true)
	}
	if node.Network != "" && node.Network != "tcp" {
		writeIndentedScalar(&out, "network", node.Network)
	}
	if node.ServerName != "" {
		writeIndentedScalar(&out, "servername", node.ServerName)
	}
	if node.Network == "ws" && (node.Path != "" || node.Host != "") {
		out.WriteString("    ws-opts:\n")
		if node.Path != "" {
			out.WriteString("      path: ")
			writeInlineString(&out, node.Path)
			out.WriteByte('\n')
		}
		if node.Host != "" {
			out.WriteString("      headers:\n        Host: ")
			writeInlineString(&out, node.Host)
			out.WriteByte('\n')
		}
	}
	return out.Bytes(), nil
}

type vmessNode struct {
	Server     string
	Port       int
	UUID       string
	AlterID    int
	Cipher     string
	TLS        bool
	Network    string
	Host       string
	Path       string
	ServerName string
}

func parseVMess(raw string) (vmessNode, error) {
	value := strings.TrimSpace(raw)
	if !strings.HasPrefix(strings.ToLower(value), "vmess://") {
		return vmessNode{}, fmt.Errorf("不是 vmess:// 链接")
	}
	encoded := value[len("vmess://"):]
	if i := strings.IndexAny(encoded, "#?"); i >= 0 {
		encoded = encoded[:i]
	}
	decoded, err := decodeBase64(encoded)
	if err != nil {
		return vmessNode{}, err
	}
	var doc struct {
		Server     string `json:"add"`
		Port       string `json:"port"`
		UUID       string `json:"id"`
		AlterID    any    `json:"aid"`
		Cipher     string `json:"scy"`
		TLS        string `json:"tls"`
		Network    string `json:"net"`
		Host       string `json:"host"`
		Path       string `json:"path"`
		ServerName string `json:"sni"`
	}
	if err := json.Unmarshal(decoded, &doc); err != nil {
		return vmessNode{}, fmt.Errorf("解析 vmess JSON 失败: %w", err)
	}
	port, err := strconv.Atoi(strings.TrimSpace(doc.Port))
	if err != nil {
		return vmessNode{}, fmt.Errorf("vmess 端口不正确: %w", err)
	}
	cipher := strings.TrimSpace(doc.Cipher)
	if cipher == "" {
		cipher = "auto"
	}
	network := strings.TrimSpace(doc.Network)
	if network == "" {
		network = "tcp"
	}
	return vmessNode{
		Server:     strings.TrimSpace(doc.Server),
		Port:       port,
		UUID:       strings.TrimSpace(doc.UUID),
		AlterID:    parseAlterID(doc.AlterID),
		Cipher:     cipher,
		TLS:        doc.TLS != "" && doc.TLS != "none",
		Network:    network,
		Host:       strings.TrimSpace(doc.Host),
		Path:       strings.TrimSpace(doc.Path),
		ServerName: strings.TrimSpace(doc.ServerName),
	}, nil
}

func parseAlterID(value any) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err == nil {
			return n
		}
	}
	return 0
}

func decodeMaybeBase64(value string) string {
	if decoded, err := decodeBase64(value); err == nil {
		return string(decoded)
	}
	decoded, err := url.QueryUnescape(value)
	if err == nil {
		return decoded
	}
	return value
}

func decodeBase64(value string) ([]byte, error) {
	value = strings.Join(strings.Fields(value), "")
	if value == "" {
		return nil, fmt.Errorf("base64 内容为空")
	}
	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}
	for _, encoding := range encodings {
		decoded, err := encoding.DecodeString(value)
		if err == nil {
			return decoded, nil
		}
	}
	if remainder := len(value) % 4; remainder != 0 {
		padded := value + strings.Repeat("=", 4-remainder)
		for _, encoding := range []*base64.Encoding{base64.StdEncoding, base64.URLEncoding} {
			decoded, err := encoding.DecodeString(padded)
			if err == nil {
				return decoded, nil
			}
		}
	}
	return nil, fmt.Errorf("base64 解码失败")
}

func writeProxyHeader(out *bytes.Buffer, name, proxyType string) {
	out.WriteString("  - name: ")
	writeInlineString(out, name)
	out.WriteString("\n    type: ")
	writeInlineString(out, proxyType)
	out.WriteByte('\n')
}

func writeScalar(out *bytes.Buffer, key, value string) {
	out.WriteString(key)
	out.WriteString(": ")
	writeInlineString(out, value)
	out.WriteByte('\n')
}

func writeBool(out *bytes.Buffer, key string, value bool) {
	out.WriteString(key)
	out.WriteString(": ")
	if value {
		out.WriteString("true\n")
	} else {
		out.WriteString("false\n")
	}
}

func writeInt(out *bytes.Buffer, key string, value int) {
	out.WriteString(key)
	out.WriteString(": ")
	out.WriteString(strconv.Itoa(value))
	out.WriteByte('\n')
}

func writeIndentedScalar(out *bytes.Buffer, key, value string) {
	out.WriteString("    ")
	out.WriteString(key)
	out.WriteString(": ")
	writeInlineString(out, value)
	out.WriteByte('\n')
}

func writeIndentedBool(out *bytes.Buffer, key string, value bool) {
	out.WriteString("    ")
	out.WriteString(key)
	out.WriteString(": ")
	if value {
		out.WriteString("true\n")
	} else {
		out.WriteString("false\n")
	}
}

func writeIndentedInt(out *bytes.Buffer, key string, value int) {
	out.WriteString("    ")
	out.WriteString(key)
	out.WriteString(": ")
	out.WriteString(strconv.Itoa(value))
	out.WriteByte('\n')
}

func writeListItem(out *bytes.Buffer, value string) {
	out.WriteString("      - ")
	writeInlineString(out, value)
	out.WriteByte('\n')
}

func writeInlineString(out *bytes.Buffer, value string) {
	encoded, _ := json.Marshal(value)
	out.Write(encoded)
}
