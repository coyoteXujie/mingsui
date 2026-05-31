package subscription

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
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
