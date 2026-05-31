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
		return nil, fmt.Errorf("没有可导出到 Mihomo 的机场节点，当前支持 ss、vmess、trojan、vless 和 hysteria2")
	}
	if selected == "" {
		name, ok := FirstAutoSelectableProfileName(cfg.ProxyProfiles)
		if !ok {
			return nil, fmt.Errorf("没有可自动选择的国外节点；请手动选择非国内节点")
		}
		selected = name
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

func CanAutoSelectProfile(profile config.ProxyProfile) bool {
	return CanExportProfile(profile) && !LikelyMainlandChinaProfile(profile)
}

func FirstExportableProfileName(profiles []config.ProxyProfile) (string, bool) {
	for _, profile := range profiles {
		if CanExportProfile(profile) {
			return profile.Name, true
		}
	}
	return "", false
}

func FirstAutoSelectableProfileName(profiles []config.ProxyProfile) (string, bool) {
	for _, profile := range profiles {
		if CanAutoSelectProfile(profile) {
			return profile.Name, true
		}
	}
	return "", false
}

func LikelyMainlandChinaProfile(profile config.ProxyProfile) bool {
	label := normalizeProxyLocationText(profile.Name)
	if label == "" {
		return false
	}
	if containsAny(label, mainlandChinaStrongLocationMarkers) {
		return true
	}
	if containsAny(label, overseasLocationMarkers) {
		return false
	}
	if containsAny(label, mainlandChinaLocationMarkers) {
		return true
	}
	for _, token := range strings.Fields(label) {
		switch token {
		case "cn", "china", "mainland":
			return true
		}
	}
	return false
}

func normalizeProxyLocationText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"|", " ",
		"/", " ",
		"\\", " ",
		"-", " ",
		"_", " ",
		".", " ",
		",", " ",
		"(", " ",
		")", " ",
		"[", " ",
		"]", " ",
		"{", " ",
		"}", " ",
		"【", " ",
		"】", " ",
		"「", " ",
		"」", " ",
		"·", " ",
		"•", " ",
		"+", " ",
		"#", " ",
		":", " ",
		"：", " ",
	)
	return strings.Join(strings.Fields(replacer.Replace(value)), " ")
}

func containsAny(value string, markers []string) bool {
	for _, marker := range markers {
		if shortASCIIMarker(marker) {
			for _, token := range strings.Fields(value) {
				if token == marker {
					return true
				}
			}
			continue
		}
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}

func shortASCIIMarker(value string) bool {
	if len(value) == 0 || len(value) > 3 {
		return false
	}
	for _, r := range value {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

var overseasLocationMarkers = []string{
	"香港", "港", "hong kong", "hk", "hkg",
	"台湾", "台灣", "taiwan", "tw", "tpe",
	"日本", "东京", "東京", "大阪", "japan", "tokyo", "osaka", "jp", "jpn",
	"新加坡", "狮城", "singapore", "sg", "sgp",
	"美国", "美國", "洛杉矶", "洛杉磯", "纽约", "紐約", "西雅图", "西雅圖", "america", "united states", "usa", "us", "la", "lax", "nyc", "sfo", "sjc", "sea",
	"英国", "英國", "伦敦", "倫敦", "united kingdom", "britain", "uk", "gb", "london",
	"德国", "德國", "法兰克福", "法蘭克福", "germany", "de", "fra", "frankfurt",
	"法国", "法國", "巴黎", "france", "fr", "paris",
	"荷兰", "荷蘭", "阿姆斯特丹", "netherlands", "nl", "ams", "amsterdam",
	"韩国", "韓國", "首尔", "首爾", "korea", "kr", "kor", "seoul",
	"加拿大", "多伦多", "多倫多", "温哥华", "溫哥華", "canada", "ca", "toronto", "vancouver",
	"澳大利亚", "澳大利亞", "澳洲", "悉尼", "墨尔本", "墨爾本", "australia", "au", "sydney", "melbourne",
	"俄罗斯", "俄羅斯", "莫斯科", "russia", "ru", "moscow",
	"印度", "孟买", "孟買", "india", "in", "mumbai",
	"泰国", "泰國", "曼谷", "thailand", "th", "bangkok",
	"越南", "河内", "胡志明", "vietnam", "vn",
	"马来西亚", "馬來西亞", "吉隆坡", "malaysia", "my",
	"菲律宾", "菲律賓", "philippines", "ph",
	"印尼", "印度尼西亚", "印度尼西亞", "indonesia", "id",
	"土耳其", "turkey", "tr",
	"迪拜", "阿联酋", "阿聯酋", "dubai", "uae", "ae",
	"巴西", "brazil", "br",
	"阿根廷", "argentina", "ar",
}

var mainlandChinaLocationMarkers = []string{
	"中国大陆", "中國大陸", "大陆", "大陸", "内地", "內地", "国内", "國內", "回国", "回國", "中国", "中國",
	"北京", "上海", "广州", "廣州", "深圳", "杭州", "南京", "苏州", "蘇州", "成都", "重庆", "重慶", "天津",
	"武汉", "武漢", "西安", "郑州", "鄭州", "长沙", "長沙", "青岛", "青島", "厦门", "廈門", "福州", "济南", "濟南",
	"沈阳", "瀋陽", "大连", "大連", "哈尔滨", "哈爾濱", "合肥", "昆明", "南宁", "南寧", "贵阳", "貴陽", "海口",
	"乌鲁木齐", "烏魯木齊", "拉萨", "拉薩", "石家庄", "石家莊", "太原", "兰州", "蘭州", "银川", "銀川", "呼和浩特",
}

var mainlandChinaStrongLocationMarkers = []string{
	"中国大陆", "中國大陸", "大陆", "大陸", "内地", "內地", "国内", "國內", "回国", "回國", "mainland",
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
	case "trojan":
		return trojanToYAML(profile)
	case "vless":
		return vlessToYAML(profile)
	case "hysteria2", "hy2":
		return hysteria2ToYAML(profile)
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
	writeTransportOptions(&out, transportOptions{Network: node.Network, Host: node.Host, Path: node.Path})
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

func trojanToYAML(profile config.ProxyProfile) ([]byte, error) {
	node, err := parseTrojan(profile.URL)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	writeProxyHeader(&out, profile.Name, "trojan")
	writeIndentedScalar(&out, "server", node.Server)
	writeIndentedInt(&out, "port", node.Port)
	writeIndentedScalar(&out, "password", node.Password)
	writeIndentedBool(&out, "udp", true)
	writeOptionalTLSFields(&out, node.TLS)
	writeTransportOptions(&out, node.Transport)
	return out.Bytes(), nil
}

type trojanNode struct {
	Server    string
	Port      int
	Password  string
	TLS       tlsOptions
	Transport transportOptions
}

func parseTrojan(raw string) (trojanNode, error) {
	parsed, err := parseURL(raw, "trojan")
	if err != nil {
		return trojanNode{}, err
	}
	server, port, err := serverPortFromURL(parsed)
	if err != nil {
		return trojanNode{}, err
	}
	password := ""
	if parsed.User != nil {
		password = parsed.User.Username()
	}
	if password == "" {
		return trojanNode{}, fmt.Errorf("trojan 链接缺少 password")
	}
	query := parsed.Query()
	return trojanNode{
		Server:    server,
		Port:      port,
		Password:  password,
		TLS:       tlsOptionsFromQuery(query, true),
		Transport: transportOptionsFromQuery(query),
	}, nil
}

func vlessToYAML(profile config.ProxyProfile) ([]byte, error) {
	node, err := parseVLESS(profile.URL)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	writeProxyHeader(&out, profile.Name, "vless")
	writeIndentedScalar(&out, "server", node.Server)
	writeIndentedInt(&out, "port", node.Port)
	writeIndentedScalar(&out, "uuid", node.UUID)
	writeIndentedBool(&out, "udp", true)
	if node.Flow != "" {
		writeIndentedScalar(&out, "flow", node.Flow)
	}
	writeOptionalTLSFields(&out, node.TLS)
	if node.ClientFingerprint != "" {
		writeIndentedScalar(&out, "client-fingerprint", node.ClientFingerprint)
	}
	if node.Reality.PublicKey != "" || node.Reality.ShortID != "" {
		out.WriteString("    reality-opts:\n")
		if node.Reality.PublicKey != "" {
			out.WriteString("      public-key: ")
			writeInlineString(&out, node.Reality.PublicKey)
			out.WriteByte('\n')
		}
		if node.Reality.ShortID != "" {
			out.WriteString("      short-id: ")
			writeInlineString(&out, node.Reality.ShortID)
			out.WriteByte('\n')
		}
	}
	writeTransportOptions(&out, node.Transport)
	return out.Bytes(), nil
}

type vlessNode struct {
	Server            string
	Port              int
	UUID              string
	Flow              string
	ClientFingerprint string
	TLS               tlsOptions
	Reality           realityOptions
	Transport         transportOptions
}

type realityOptions struct {
	PublicKey string
	ShortID   string
}

func parseVLESS(raw string) (vlessNode, error) {
	parsed, err := parseURL(raw, "vless")
	if err != nil {
		return vlessNode{}, err
	}
	server, port, err := serverPortFromURL(parsed)
	if err != nil {
		return vlessNode{}, err
	}
	uuid := ""
	if parsed.User != nil {
		uuid = parsed.User.Username()
	}
	if uuid == "" {
		return vlessNode{}, fmt.Errorf("vless 链接缺少 uuid")
	}
	query := parsed.Query()
	security := strings.ToLower(strings.TrimSpace(query.Get("security")))
	tls := tlsOptionsFromQuery(query, security == "tls" || security == "reality")
	return vlessNode{
		Server:            server,
		Port:              port,
		UUID:              uuid,
		Flow:              firstQueryValue(query, "flow"),
		ClientFingerprint: firstQueryValue(query, "fp", "client-fingerprint", "client_fingerprint"),
		TLS:               tls,
		Reality: realityOptions{
			PublicKey: firstQueryValue(query, "pbk", "public-key", "public_key"),
			ShortID:   firstQueryValue(query, "sid", "short-id", "short_id"),
		},
		Transport: transportOptionsFromQuery(query),
	}, nil
}

func hysteria2ToYAML(profile config.ProxyProfile) ([]byte, error) {
	node, err := parseHysteria2(profile.URL)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	writeProxyHeader(&out, profile.Name, "hysteria2")
	writeIndentedScalar(&out, "server", node.Server)
	writeIndentedInt(&out, "port", node.Port)
	writeIndentedScalar(&out, "password", node.Password)
	if node.SNI != "" {
		writeIndentedScalar(&out, "sni", node.SNI)
	}
	if node.SkipCertVerify {
		writeIndentedBool(&out, "skip-cert-verify", true)
	}
	if node.Obfs != "" {
		writeIndentedScalar(&out, "obfs", node.Obfs)
	}
	if node.ObfsPassword != "" {
		writeIndentedScalar(&out, "obfs-password", node.ObfsPassword)
	}
	return out.Bytes(), nil
}

type hysteria2Node struct {
	Server         string
	Port           int
	Password       string
	SNI            string
	SkipCertVerify bool
	Obfs           string
	ObfsPassword   string
}

func parseHysteria2(raw string) (hysteria2Node, error) {
	parsed, err := parseURL(raw, "hysteria2", "hy2")
	if err != nil {
		return hysteria2Node{}, err
	}
	server, port, err := serverPortFromURL(parsed)
	if err != nil {
		return hysteria2Node{}, err
	}
	password := ""
	if parsed.User != nil {
		password = parsed.User.Username()
	}
	if password == "" {
		return hysteria2Node{}, fmt.Errorf("hysteria2 链接缺少 password")
	}
	query := parsed.Query()
	return hysteria2Node{
		Server:         server,
		Port:           port,
		Password:       password,
		SNI:            firstQueryValue(query, "sni", "peer"),
		SkipCertVerify: queryBool(query, "insecure", "allowInsecure", "skip-cert-verify", "skip_cert_verify"),
		Obfs:           firstQueryValue(query, "obfs"),
		ObfsPassword:   firstQueryValue(query, "obfs-password", "obfs_password", "obfsPassword"),
	}, nil
}

type tlsOptions struct {
	Enabled        bool
	ServerName     string
	SkipCertVerify bool
	ALPN           []string
}

type transportOptions struct {
	Network     string
	Host        string
	Path        string
	ServiceName string
}

func parseURL(raw string, schemes ...string) (*url.URL, error) {
	value := strings.TrimSpace(raw)
	parsed, err := url.Parse(value)
	if err != nil {
		return nil, err
	}
	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	for _, want := range schemes {
		if scheme == strings.ToLower(want) {
			return parsed, nil
		}
	}
	return nil, fmt.Errorf("不是 %s 链接", strings.Join(schemes, "/"))
}

func serverPortFromURL(parsed *url.URL) (string, int, error) {
	server := strings.Trim(parsed.Hostname(), "[]")
	if server == "" {
		return "", 0, fmt.Errorf("节点地址缺少 server")
	}
	portText := parsed.Port()
	if portText == "" {
		return "", 0, fmt.Errorf("节点地址缺少 port")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port <= 0 || port > 65535 {
		return "", 0, fmt.Errorf("端口不正确: %s", portText)
	}
	return server, port, nil
}

func tlsOptionsFromQuery(query url.Values, enabled bool) tlsOptions {
	return tlsOptions{
		Enabled:        enabled,
		ServerName:     firstQueryValue(query, "sni", "peer", "servername", "serverName"),
		SkipCertVerify: queryBool(query, "allowInsecure", "insecure", "skip-cert-verify", "skip_cert_verify"),
		ALPN:           queryList(query, "alpn"),
	}
}

func transportOptionsFromQuery(query url.Values) transportOptions {
	return transportOptions{
		Network:     strings.ToLower(firstQueryValue(query, "type", "network")),
		Host:        firstQueryValue(query, "host"),
		Path:        firstQueryValue(query, "path"),
		ServiceName: firstQueryValue(query, "serviceName", "service_name", "service-name"),
	}
}

func writeOptionalTLSFields(out *bytes.Buffer, opts tlsOptions) {
	if opts.Enabled {
		writeIndentedBool(out, "tls", true)
	}
	if opts.ServerName != "" {
		writeIndentedScalar(out, "servername", opts.ServerName)
	}
	if opts.SkipCertVerify {
		writeIndentedBool(out, "skip-cert-verify", true)
	}
	if len(opts.ALPN) > 0 {
		out.WriteString("    alpn:\n")
		for _, item := range opts.ALPN {
			out.WriteString("      - ")
			writeInlineString(out, item)
			out.WriteByte('\n')
		}
	}
}

func writeTransportOptions(out *bytes.Buffer, opts transportOptions) {
	if opts.Network == "" || opts.Network == "tcp" {
		return
	}
	writeIndentedScalar(out, "network", opts.Network)
	switch opts.Network {
	case "ws":
		if opts.Path == "" && opts.Host == "" {
			return
		}
		out.WriteString("    ws-opts:\n")
		if opts.Path != "" {
			out.WriteString("      path: ")
			writeInlineString(out, opts.Path)
			out.WriteByte('\n')
		}
		if opts.Host != "" {
			out.WriteString("      headers:\n        Host: ")
			writeInlineString(out, opts.Host)
			out.WriteByte('\n')
		}
	case "grpc":
		if opts.ServiceName == "" {
			return
		}
		out.WriteString("    grpc-opts:\n      grpc-service-name: ")
		writeInlineString(out, opts.ServiceName)
		out.WriteByte('\n')
	}
}

func firstQueryValue(query url.Values, names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(query.Get(name)); value != "" {
			return value
		}
	}
	return ""
}

func queryBool(query url.Values, names ...string) bool {
	for _, name := range names {
		value := strings.ToLower(strings.TrimSpace(query.Get(name)))
		switch value {
		case "1", "true", "yes", "y":
			return true
		}
	}
	return false
}

func queryList(query url.Values, name string) []string {
	value := strings.TrimSpace(query.Get(name))
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			items = append(items, part)
		}
	}
	return items
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
