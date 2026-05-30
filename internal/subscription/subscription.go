package subscription

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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

func (l Loader) LoadRelayProfiles(ctx context.Context, source string, stdin io.Reader) ([]config.RelayProfile, error) {
	data, err := l.readSource(ctx, source, stdin)
	if err != nil {
		return nil, err
	}
	return ParseRelayProfiles(data)
}

func ParseRelayProfiles(data []byte) ([]config.RelayProfile, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, fmt.Errorf("订阅内容为空")
	}

	var profiles []config.RelayProfile
	if data[0] == '[' {
		if err := json.Unmarshal(data, &profiles); err != nil {
			return nil, fmt.Errorf("解析 profile 列表失败: %w", err)
		}
	} else {
		var doc Document
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("解析订阅失败: %w", err)
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

func (l Loader) readSource(ctx context.Context, source string, stdin io.Reader) ([]byte, error) {
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
