package subscription

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/coyoteXujie/mingsui/internal/config"
)

const defaultMaxSubscriptionBytes int64 = 1 << 20

type Document struct {
	Version  int                   `json:"version"`
	Profiles []config.RelayProfile `json:"profiles"`
}

type Loader struct {
	HTTPClient *http.Client
	MaxBytes   int64
}

func LoadRelayProfiles(ctx context.Context, source string, stdin io.Reader) ([]config.RelayProfile, error) {
	return Loader{}.LoadRelayProfiles(ctx, source, stdin)
}

func LoadSource(ctx context.Context, source string, stdin io.Reader) ([]byte, error) {
	return Loader{}.LoadSource(ctx, source, stdin)
}

func (l Loader) LoadRelayProfiles(ctx context.Context, source string, stdin io.Reader) ([]config.RelayProfile, error) {
	data, err := l.LoadSource(ctx, source, stdin)
	if err != nil {
		return nil, err
	}
	return ParseRelayProfiles(data)
}

func (l Loader) LoadProxyProfiles(ctx context.Context, source string, stdin io.Reader) ([]config.ProxyProfile, error) {
	data, err := l.LoadSource(ctx, source, stdin)
	if err != nil {
		return nil, err
	}
	return ParseProxyProfiles(data)
}

func ParseRelayProfiles(data []byte) ([]config.RelayProfile, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, fmt.Errorf("订阅内容为空")
	}

	var profiles []config.RelayProfile
	if data[0] == '[' {
		if err := json.Unmarshal(data, &profiles); err != nil {
			return nil, relayProfileParseError(data, fmt.Errorf("解析 profile 列表失败: %w", err))
		}
	} else {
		var doc Document
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, relayProfileParseError(data, fmt.Errorf("解析订阅失败: %w", err))
		}
		profiles = doc.Profiles
	}
	if len(profiles) == 0 {
		return nil, fmt.Errorf("订阅中没有 relay profile")
	}

	cfg := config.DefaultClient()
	cfg.Profiles = nil
	if err := cfg.ImportRelayProfiles(profiles, false); err != nil {
		return nil, err
	}
	return cfg.Profiles, nil
}

func ParseProxyProfiles(data []byte) ([]config.ProxyProfile, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, fmt.Errorf("订阅内容为空")
	}

	candidates := [][]byte{data}
	if decoded, ok := decodeSubscriptionBase64(data); ok {
		candidates = append([][]byte{bytes.TrimSpace(decoded)}, candidates...)
	}
	for _, candidate := range candidates {
		profiles, err := parseProxyProfileLines(candidate)
		if err != nil {
			return nil, err
		}
		if len(profiles) > 0 {
			return profiles, nil
		}
		profiles, recognized, err := parseClashProxyProfiles(candidate)
		if err != nil {
			return nil, err
		}
		if len(profiles) > 0 {
			return profiles, nil
		}
		if recognized {
			return nil, fmt.Errorf("Clash/Mihomo 订阅中没有识别到支持的机场节点")
		}
	}
	return nil, fmt.Errorf("没有识别到支持的机场节点")
}

func parseProxyProfileLines(data []byte) ([]config.ProxyProfile, error) {
	var profiles []config.ProxyProfile
	nameCounts := make(map[string]int)
	for _, raw := range strings.Fields(string(data)) {
		profile, ok, err := parseProxyProfile(raw)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		if isInformationalProxyName(profile.Name) {
			continue
		}
		profile.Name = uniqueProxyProfileName(profile.Name, nameCounts)
		profiles = append(profiles, profile)
	}
	return profiles, nil
}

func parseProxyProfile(raw string) (config.ProxyProfile, bool, error) {
	raw = strings.TrimSpace(raw)
	scheme, _, ok := strings.Cut(raw, "://")
	if !ok {
		return config.ProxyProfile{}, false, nil
	}
	protocol := strings.ToLower(strings.TrimSpace(scheme))
	if !isStandardProxyScheme(protocol) {
		return config.ProxyProfile{}, false, nil
	}

	name := proxyNameFromURL(raw)
	if name == "" && protocol == "vmess" {
		name = proxyNameFromVMess(raw)
	}
	if name == "" {
		name = protocol
	}
	profile := config.ProxyProfile{
		Name:     name,
		Protocol: protocol,
		URL:      raw,
	}
	cfg := config.DefaultClient()
	if err := cfg.ImportProxyProfiles([]config.ProxyProfile{profile}, false); err != nil {
		return config.ProxyProfile{}, false, err
	}
	return cfg.ProxyProfiles[0], true, nil
}

func parseClashProxyProfiles(data []byte) ([]config.ProxyProfile, bool, error) {
	nodes, recognized := extractClashProxyNodes(data)
	if !recognized {
		return nil, false, nil
	}

	profiles := make([]config.ProxyProfile, 0, len(nodes))
	nameCounts := make(map[string]int)
	for _, node := range nodes {
		profile, ok, err := clashNodeToProxyProfile(node)
		if err != nil {
			return nil, true, err
		}
		if !ok || isInformationalProxyName(profile.Name) {
			continue
		}
		profile.Name = uniqueProxyProfileName(profile.Name, nameCounts)
		profiles = append(profiles, profile)
	}
	if len(profiles) == 0 {
		return nil, true, nil
	}

	cfg := config.DefaultClient()
	if err := cfg.ImportProxyProfiles(profiles, false); err != nil {
		return nil, true, err
	}
	return cfg.ProxyProfiles, true, nil
}

type yamlKeyLevel struct {
	Indent int
	Key    string
}

func extractClashProxyNodes(data []byte) ([]map[string]string, bool) {
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	inProxies := false
	proxiesIndent := -1
	itemIndent := -1
	var current map[string]string
	var stack []yamlKeyLevel
	var nodes []map[string]string

	flush := func() {
		if len(current) > 0 {
			nodes = append(nodes, current)
		}
		current = nil
		stack = nil
		itemIndent = -1
	}

	for _, raw := range lines {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		indent := countLeadingSpaces(raw)
		trimmed := strings.TrimSpace(trimYAMLComment(raw))
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if !inProxies {
			key, value, ok := splitYAMLKeyValue(trimmed)
			if indent == 0 && ok && normalizeYAMLKey(key) == "proxies" && strings.TrimSpace(value) == "" {
				inProxies = true
				proxiesIndent = indent
			}
			continue
		}

		if indent <= proxiesIndent && !strings.HasPrefix(trimmed, "-") {
			break
		}
		if strings.HasPrefix(trimmed, "-") && indent > proxiesIndent {
			flush()
			current = make(map[string]string)
			itemIndent = indent
			rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
			if rest == "" {
				continue
			}
			if strings.HasPrefix(rest, "{") && strings.HasSuffix(rest, "}") {
				for key, value := range parseInlineYAMLMap(rest) {
					current[key] = value
				}
				continue
			}
			parseYAMLField(current, rest, indent+2, &stack)
			continue
		}
		if current != nil && indent > itemIndent {
			parseYAMLField(current, trimmed, indent, &stack)
		}
	}
	flush()
	return nodes, inProxies
}

func parseYAMLField(fields map[string]string, line string, indent int, stack *[]yamlKeyLevel) {
	key, value, ok := splitYAMLKeyValue(line)
	if !ok {
		return
	}
	key = normalizeYAMLKey(key)
	if key == "" {
		return
	}
	levels := (*stack)[:0]
	for _, item := range *stack {
		if item.Indent < indent {
			levels = append(levels, item)
		}
	}
	*stack = levels

	fullKey := key
	if len(levels) > 0 {
		parts := make([]string, 0, len(levels)+1)
		for _, item := range levels {
			parts = append(parts, item.Key)
		}
		parts = append(parts, key)
		fullKey = strings.Join(parts, ".")
	}

	if strings.TrimSpace(value) == "" {
		*stack = append(*stack, yamlKeyLevel{Indent: indent, Key: key})
		return
	}
	fields[fullKey] = parseYAMLScalar(value)
}

func parseInlineYAMLMap(value string) map[string]string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "{")
	value = strings.TrimSuffix(value, "}")
	fields := make(map[string]string)
	for _, part := range splitInlineYAMLFields(value) {
		key, val, ok := splitYAMLKeyValue(part)
		if !ok {
			continue
		}
		key = normalizeYAMLKey(key)
		if key == "" {
			continue
		}
		fields[key] = parseYAMLScalar(val)
	}
	return fields
}

func splitInlineYAMLFields(value string) []string {
	var parts []string
	start := 0
	quote := rune(0)
	depth := 0
	for i, r := range value {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			}
		case r == '\'' || r == '"':
			quote = r
		case r == '{' || r == '[':
			depth++
		case r == '}' || r == ']':
			if depth > 0 {
				depth--
			}
		case r == ',' && depth == 0:
			if part := strings.TrimSpace(value[start:i]); part != "" {
				parts = append(parts, part)
			}
			start = i + 1
		}
	}
	if part := strings.TrimSpace(value[start:]); part != "" {
		parts = append(parts, part)
	}
	return parts
}

func splitYAMLKeyValue(line string) (string, string, bool) {
	line = strings.TrimSpace(trimYAMLComment(line))
	quote := rune(0)
	for i, r := range line {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			}
		case r == '\'' || r == '"':
			quote = r
		case r == ':':
			if i == 0 {
				return "", "", false
			}
			return strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:]), true
		}
	}
	return "", "", false
}

func trimYAMLComment(line string) string {
	quote := rune(0)
	for i, r := range line {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			}
		case r == '\'' || r == '"':
			quote = r
		case r == '#':
			if i == 0 || isYAMLWhitespace(rune(line[i-1])) {
				return strings.TrimSpace(line[:i])
			}
		}
	}
	return strings.TrimSpace(line)
}

func parseYAMLScalar(value string) string {
	value = strings.TrimSpace(trimYAMLComment(value))
	value = strings.TrimSuffix(value, ",")
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			if unquoted, err := strconv.Unquote(value); err == nil {
				return strings.TrimSpace(unquoted)
			}
			return strings.TrimSpace(value[1 : len(value)-1])
		}
	}
	return strings.TrimSpace(value)
}

func normalizeYAMLKey(key string) string {
	key = parseYAMLScalar(key)
	key = strings.ToLower(strings.TrimSpace(key))
	key = strings.ReplaceAll(key, "_", "-")
	return key
}

func countLeadingSpaces(value string) int {
	count := 0
	for _, r := range value {
		if r != ' ' {
			break
		}
		count++
	}
	return count
}

func isYAMLWhitespace(r rune) bool {
	return r == ' ' || r == '\t'
}

func clashNodeToProxyProfile(node map[string]string) (config.ProxyProfile, bool, error) {
	name := nodeValue(node, "name")
	protocol := normalizeProxyProtocol(nodeValue(node, "type"))
	if name == "" {
		name = protocol
	}
	if protocol == "" {
		return config.ProxyProfile{}, false, nil
	}

	var raw string
	var err error
	switch protocol {
	case "ss":
		raw, err = clashSSToShareURL(node, name)
	case "vmess":
		raw, err = clashVMessToShareURL(node, name)
	case "trojan":
		raw, err = clashTrojanToShareURL(node, name)
	case "vless":
		raw, err = clashVLESSToShareURL(node, name)
	case "hysteria2", "hy2":
		raw, err = clashHysteria2ToShareURL(node, name)
		protocol = "hysteria2"
	default:
		return config.ProxyProfile{}, false, nil
	}
	if err != nil {
		return config.ProxyProfile{}, true, fmt.Errorf("解析 Clash/Mihomo 节点 %q 失败: %w", name, err)
	}
	return config.ProxyProfile{Name: name, Protocol: protocol, URL: raw}, true, nil
}

func clashSSToShareURL(node map[string]string, name string) (string, error) {
	server, port, err := clashServerPort(node)
	if err != nil {
		return "", err
	}
	cipher := nodeValue(node, "cipher", "method")
	password := nodeValue(node, "password")
	if cipher == "" || password == "" {
		return "", fmt.Errorf("ss 节点缺少 cipher/password")
	}
	credential := base64.RawURLEncoding.EncodeToString([]byte(cipher + ":" + password))
	return "ss://" + credential + "@" + net.JoinHostPort(server, strconv.Itoa(port)) + "#" + url.QueryEscape(name), nil
}

func clashVMessToShareURL(node map[string]string, name string) (string, error) {
	server, port, err := clashServerPort(node)
	if err != nil {
		return "", err
	}
	uuid := nodeValue(node, "uuid", "id")
	if uuid == "" {
		return "", fmt.Errorf("vmess 节点缺少 uuid")
	}
	cipher := nodeValue(node, "cipher", "scy")
	if cipher == "" {
		cipher = "auto"
	}
	network := nodeValue(node, "network")
	if network == "" {
		network = "tcp"
	}
	tlsValue := ""
	if yamlBool(nodeValue(node, "tls")) {
		tlsValue = "tls"
	} else if nodeValue(node, "tls") != "" {
		tlsValue = "none"
	}
	doc := map[string]string{
		"v":    "2",
		"ps":   name,
		"add":  server,
		"port": strconv.Itoa(port),
		"id":   uuid,
		"aid":  nodeValueDefault(node, "0", "alterid", "alter-id"),
		"scy":  cipher,
		"net":  network,
		"type": "none",
		"host": nodeValue(node, "host", "ws-opts.headers.host"),
		"path": nodeValue(node, "path", "ws-opts.path"),
		"tls":  tlsValue,
		"sni":  nodeValue(node, "sni", "servername", "server-name"),
	}
	data, err := json.Marshal(doc)
	if err != nil {
		return "", err
	}
	return "vmess://" + base64.StdEncoding.EncodeToString(data) + "#" + url.QueryEscape(name), nil
}

func clashTrojanToShareURL(node map[string]string, name string) (string, error) {
	server, port, err := clashServerPort(node)
	if err != nil {
		return "", err
	}
	password := nodeValue(node, "password")
	if password == "" {
		return "", fmt.Errorf("trojan 节点缺少 password")
	}
	query := clashCommonProxyQuery(node)
	u := url.URL{Scheme: "trojan", User: url.User(password), Host: net.JoinHostPort(server, strconv.Itoa(port)), RawQuery: query.Encode(), Fragment: name}
	return u.String(), nil
}

func clashVLESSToShareURL(node map[string]string, name string) (string, error) {
	server, port, err := clashServerPort(node)
	if err != nil {
		return "", err
	}
	uuid := nodeValue(node, "uuid", "id")
	if uuid == "" {
		return "", fmt.Errorf("vless 节点缺少 uuid")
	}
	query := clashCommonProxyQuery(node)
	security := strings.ToLower(nodeValue(node, "security"))
	if security == "" {
		switch {
		case nodeValue(node, "reality-opts.public-key", "reality-opts.short-id") != "":
			security = "reality"
		case yamlBool(nodeValue(node, "tls")):
			security = "tls"
		}
	}
	if security != "" {
		query.Set("security", security)
	}
	if value := nodeValue(node, "flow"); value != "" {
		query.Set("flow", value)
	}
	if value := nodeValue(node, "client-fingerprint", "fingerprint"); value != "" {
		query.Set("fp", value)
	}
	if value := nodeValue(node, "reality-opts.public-key", "pbk", "public-key"); value != "" {
		query.Set("pbk", value)
	}
	if value := nodeValue(node, "reality-opts.short-id", "sid", "short-id"); value != "" {
		query.Set("sid", value)
	}
	u := url.URL{Scheme: "vless", User: url.User(uuid), Host: net.JoinHostPort(server, strconv.Itoa(port)), RawQuery: query.Encode(), Fragment: name}
	return u.String(), nil
}

func clashHysteria2ToShareURL(node map[string]string, name string) (string, error) {
	server, port, err := clashServerPort(node)
	if err != nil {
		return "", err
	}
	password := nodeValue(node, "password", "auth", "auth-str")
	if password == "" {
		return "", fmt.Errorf("hysteria2 节点缺少 password/auth")
	}
	query := url.Values{}
	if value := nodeValue(node, "sni", "servername", "server-name"); value != "" {
		query.Set("sni", value)
	}
	if yamlBool(nodeValue(node, "skip-cert-verify")) {
		query.Set("insecure", "1")
	}
	if value := nodeValue(node, "obfs"); value != "" {
		query.Set("obfs", value)
	}
	if value := nodeValue(node, "obfs-password", "obfs_password"); value != "" {
		query.Set("obfs-password", value)
	}
	u := url.URL{Scheme: "hysteria2", User: url.User(password), Host: net.JoinHostPort(server, strconv.Itoa(port)), RawQuery: query.Encode(), Fragment: name}
	return u.String(), nil
}

func clashServerPort(node map[string]string) (string, int, error) {
	server := nodeValue(node, "server", "address", "addr")
	if server == "" {
		return "", 0, fmt.Errorf("节点缺少 server")
	}
	portText := nodeValue(node, "port")
	if portText == "" {
		return "", 0, fmt.Errorf("节点缺少 port")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port <= 0 || port > 65535 {
		return "", 0, fmt.Errorf("端口不正确: %s", portText)
	}
	return server, port, nil
}

func clashCommonProxyQuery(node map[string]string) url.Values {
	query := url.Values{}
	if value := nodeValue(node, "sni", "servername", "server-name"); value != "" {
		query.Set("sni", value)
	}
	if yamlBool(nodeValue(node, "skip-cert-verify")) {
		query.Set("allowInsecure", "1")
	}
	if value := nodeValue(node, "network"); value != "" {
		query.Set("type", value)
	}
	if value := nodeValue(node, "host", "ws-opts.headers.host"); value != "" {
		query.Set("host", value)
	}
	if value := nodeValue(node, "path", "ws-opts.path"); value != "" {
		query.Set("path", value)
	}
	if value := nodeValue(node, "service-name", "grpc-opts.grpc-service-name"); value != "" {
		query.Set("serviceName", value)
	}
	return query
}

func normalizeProxyProtocol(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "shadowsocks":
		return "ss"
	case "hysteria2", "hy2":
		return "hysteria2"
	default:
		return value
	}
}

func nodeValue(node map[string]string, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(node[normalizeYAMLKey(key)]); value != "" {
			return value
		}
	}
	return ""
}

func nodeValueDefault(node map[string]string, fallback string, keys ...string) string {
	if value := nodeValue(node, keys...); value != "" {
		return value
	}
	return fallback
}

func yamlBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func proxyNameFromURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	name := strings.TrimSpace(parsed.Fragment)
	if name == "" {
		return ""
	}
	if decoded, err := url.QueryUnescape(name); err == nil {
		name = decoded
	}
	return strings.TrimSpace(name)
}

func proxyNameFromVMess(raw string) string {
	_, encoded, ok := strings.Cut(raw, "://")
	if !ok {
		return ""
	}
	if i := strings.IndexAny(encoded, "#?"); i >= 0 {
		encoded = encoded[:i]
	}
	decoded, ok := decodeSubscriptionBase64([]byte(encoded))
	if !ok {
		return ""
	}
	var doc struct {
		Name string `json:"ps"`
	}
	if err := json.Unmarshal(decoded, &doc); err != nil {
		return ""
	}
	return strings.TrimSpace(doc.Name)
}

func uniqueProxyProfileName(name string, counts map[string]int) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "proxy"
	}
	count := counts[name]
	counts[name] = count + 1
	if count == 0 {
		return name
	}
	return fmt.Sprintf("%s-%d", name, count+1)
}

func isInformationalProxyName(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return false
	}
	markers := []string{
		"剩余流量",
		"套餐到期",
		"到期",
		"过期",
		"官网",
		"入口",
		"电报",
		"群组",
		"traffic",
		"expire",
		"expired",
		"subscription",
	}
	for _, marker := range markers {
		if strings.Contains(name, marker) {
			return true
		}
	}
	return false
}

type proxySubscriptionInspection struct {
	Total  int
	Counts map[string]int
}

func relayProfileParseError(data []byte, parseErr error) error {
	inspection := inspectProxySubscription(data)
	if inspection.Total == 0 {
		return parseErr
	}
	return fmt.Errorf("识别到真实机场订阅（%s，共 %d 个节点）；当前命令只导入明隧 relay profile，请使用 mingsui import 导入机场节点", formatSchemeCounts(inspection.Counts), inspection.Total)
}

func inspectProxySubscription(data []byte) proxySubscriptionInspection {
	candidates := [][]byte{bytes.TrimSpace(data)}
	if decoded, ok := decodeSubscriptionBase64(data); ok {
		candidates = append([][]byte{bytes.TrimSpace(decoded)}, candidates...)
	}

	for _, candidate := range candidates {
		counts := countProxySchemes(candidate)
		total := 0
		for _, count := range counts {
			total += count
		}
		if total > 0 {
			return proxySubscriptionInspection{Total: total, Counts: counts}
		}
	}
	return proxySubscriptionInspection{}
}

func decodeSubscriptionBase64(data []byte) ([]byte, bool) {
	text := strings.Join(strings.Fields(string(data)), "")
	if text == "" {
		return nil, false
	}
	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}
	for _, encoding := range encodings {
		decoded, err := encoding.DecodeString(text)
		if err == nil && len(bytes.TrimSpace(decoded)) > 0 {
			return decoded, true
		}
	}
	if remainder := len(text) % 4; remainder != 0 {
		padded := text + strings.Repeat("=", 4-remainder)
		for _, encoding := range []*base64.Encoding{base64.StdEncoding, base64.URLEncoding} {
			decoded, err := encoding.DecodeString(padded)
			if err == nil && len(bytes.TrimSpace(decoded)) > 0 {
				return decoded, true
			}
		}
	}
	return nil, false
}

func countProxySchemes(data []byte) map[string]int {
	counts := make(map[string]int)
	for _, line := range strings.Fields(string(data)) {
		scheme, _, ok := strings.Cut(line, "://")
		if !ok {
			continue
		}
		scheme = strings.ToLower(strings.TrimSpace(scheme))
		if isStandardProxyScheme(scheme) {
			counts[scheme]++
		}
	}
	return counts
}

func isStandardProxyScheme(scheme string) bool {
	switch scheme {
	case "ss", "ssr", "vmess", "vless", "trojan", "hysteria", "hysteria2", "hy2", "tuic":
		return true
	default:
		return false
	}
}

func formatSchemeCounts(counts map[string]int) string {
	schemes := make([]string, 0, len(counts))
	for scheme := range counts {
		schemes = append(schemes, scheme)
	}
	sort.Strings(schemes)

	parts := make([]string, 0, len(schemes))
	for _, scheme := range schemes {
		parts = append(parts, fmt.Sprintf("%s: %d", scheme, counts[scheme]))
	}
	return strings.Join(parts, "，")
}

func (l Loader) LoadSource(ctx context.Context, source string, stdin io.Reader) ([]byte, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil, fmt.Errorf("订阅来源不能为空")
	}
	limit := l.MaxBytes
	if limit <= 0 {
		limit = defaultMaxSubscriptionBytes
	}
	if source == "-" {
		if stdin == nil {
			stdin = os.Stdin
		}
		return readLimited(stdin, limit)
	}
	if isHTTPURL(source) {
		return l.readHTTP(ctx, source, limit)
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("订阅内容超过 %d 字节", limit)
	}
	return data, nil
}

func (l Loader) readHTTP(ctx context.Context, source string, limit int64) ([]byte, error) {
	client := l.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("订阅请求失败: HTTP %d", resp.StatusCode)
	}
	return readLimited(resp.Body, limit)
}

func readLimited(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("订阅内容超过 %d 字节", limit)
	}
	return data, nil
}

func isHTTPURL(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}
