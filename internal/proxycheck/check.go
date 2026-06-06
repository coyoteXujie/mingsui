package proxycheck

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/coyoteXujie/mingsui/internal/config"
	"github.com/coyoteXujie/mingsui/internal/mihomo"
)

const (
	DefaultTargetURL = "https://www.gstatic.com/generate_204"
	DefaultTimeout   = 10 * time.Second
)

var (
	ErrNoCandidates  = errors.New("没有可测速的国外机场节点")
	ErrNoHealthyNode = errors.New("没有检测到可用的国外机场节点")
)

type ProbeFunc func(ctx context.Context, proxyAddr, targetURL string) error

type Options struct {
	TargetURL                string
	Timeout                  time.Duration
	Limit                    int
	WorkDir                  string
	Probe                    ProbeFunc
	IncludeNonAutoSelectable bool
}

type Report struct {
	TargetURL string   `json:"target_url"`
	BestName  string   `json:"best,omitempty"`
	Selected  string   `json:"selected,omitempty"`
	Error     string   `json:"error,omitempty"`
	Results   []Result `json:"results"`
}

type Result struct {
	Name           string `json:"name"`
	Protocol       string `json:"protocol"`
	Exportable     bool   `json:"exportable"`
	AutoSelectable bool   `json:"auto_selectable"`
	Tested         bool   `json:"tested"`
	OK             bool   `json:"ok"`
	LatencyMS      int64  `json:"latency_ms,omitempty"`
	SkipReason     string `json:"skip_reason,omitempty"`
	Error          string `json:"error,omitempty"`
}

func Check(ctx context.Context, cfg config.ClientConfig, opts Options) (Report, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	targetURL := strings.TrimSpace(opts.TargetURL)
	if targetURL == "" {
		targetURL = DefaultTargetURL
	}
	if err := validateTargetURL(targetURL); err != nil {
		return Report{TargetURL: targetURL, Error: err.Error()}, err
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	probe := opts.Probe
	if probe == nil {
		probe = probeViaHTTPProxy
	}

	report := Report{TargetURL: targetURL, Results: make([]Result, 0, len(cfg.ProxyProfiles))}
	tested := 0
	for _, profile := range cfg.ProxyProfiles {
		result := Result{
			Name:           profile.Name,
			Protocol:       profile.Protocol,
			Exportable:     mihomo.CanExportProfile(profile),
			AutoSelectable: mihomo.CanAutoSelectProfile(profile),
		}
		switch {
		case !result.Exportable:
			result.SkipReason = "暂不支持直接连接"
		case !result.AutoSelectable && !opts.IncludeNonAutoSelectable:
			result.SkipReason = "国内节点不自动选择"
		case opts.Limit > 0 && tested >= opts.Limit:
			result.SkipReason = "超过检测数量上限"
		default:
			tested++
			result.Tested = true
			latency, err := checkProfile(ctx, cfg, profile, targetURL, timeout, opts.WorkDir, probe)
			if err != nil {
				result.Error = err.Error()
			} else {
				result.OK = true
				result.LatencyMS = latency.Milliseconds()
			}
			if ctx.Err() != nil {
				report.Results = append(report.Results, result)
				report.Error = ctx.Err().Error()
				return report, ctx.Err()
			}
		}
		report.Results = append(report.Results, result)
	}

	if tested == 0 {
		report.Error = ErrNoCandidates.Error()
		return report, ErrNoCandidates
	}
	if best, ok := report.Best(); ok {
		report.BestName = best.Name
		return report, nil
	}
	report.Error = ErrNoHealthyNode.Error()
	return report, ErrNoHealthyNode
}

func (r Report) Best() (Result, bool) {
	var best Result
	ok := false
	for _, result := range r.Results {
		if !result.OK {
			continue
		}
		if !ok || result.LatencyMS < best.LatencyMS {
			best = result
			ok = true
		}
	}
	return best, ok
}

func checkProfile(ctx context.Context, cfg config.ClientConfig, profile config.ProxyProfile, targetURL string, timeout time.Duration, baseWorkDir string, probe ProbeFunc) (time.Duration, error) {
	testCfg := cfg.Clone()
	testCfg.ActiveProfile = ""
	testCfg.ActiveProxyProfile = profile.Name

	localAddr, err := freeLoopbackAddr()
	if err != nil {
		return 0, err
	}
	httpAddr, err := freeLoopbackAddr()
	if err != nil {
		return 0, err
	}
	testCfg.LocalAddr = localAddr
	testCfg.HTTPAddr = httpAddr

	workDir, err := os.MkdirTemp(baseWorkDir, "mingsui-proxy-check-*")
	if err != nil {
		return 0, fmt.Errorf("创建测速工作目录失败: %w", err)
	}
	defer os.RemoveAll(workDir)

	nodeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	controller := mihomo.NewController(testCfg, mihomo.Options{WorkDir: workDir})
	if err := controller.Start(nodeCtx); err != nil {
		return 0, err
	}
	defer func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer stopCancel()
		_ = controller.Stop(stopCtx)
	}()

	if err := waitTCP(nodeCtx, testCfg.HTTPAddr); err != nil {
		return 0, err
	}
	start := time.Now()
	if err := probe(nodeCtx, testCfg.HTTPAddr, targetURL); err != nil {
		return 0, err
	}
	return time.Since(start), nil
}

func validateTargetURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("测速 URL 不正确: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("测速 URL 必须是 http 或 https")
	}
	if parsed.Host == "" {
		return errors.New("测速 URL 缺少主机名")
	}
	return nil
}

func freeLoopbackAddr() (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("分配本地端口失败: %w", err)
	}
	defer listener.Close()
	return listener.Addr().String(), nil
}

func waitTCP(ctx context.Context, addr string) error {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	var lastErr error
	for {
		dialCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
		conn, err := (&net.Dialer{}).DialContext(dialCtx, "tcp", addr)
		cancel()
		if err == nil {
			_ = conn.Close()
			return nil
		}
		lastErr = err

		select {
		case <-ctx.Done():
			if lastErr != nil {
				return fmt.Errorf("等待本地 HTTP 代理就绪超时: %w", lastErr)
			}
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func probeViaHTTPProxy(ctx context.Context, proxyAddr, targetURL string) error {
	proxyURL := &url.URL{Scheme: "http", Host: proxyAddr}
	transport := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	defer transport.CloseIdleConnections()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "mingsui-proxy-check")
	resp, err := (&http.Client{Transport: transport}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 500 {
		return fmt.Errorf("测速 URL 返回 HTTP %d", resp.StatusCode)
	}
	return nil
}
