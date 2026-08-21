package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func NewProxyExitInfoProber(cfg *config.Config) service.ProxyExitInfoProber {
	insecure := false
	allowPrivate := false
	validateResolvedIP := true
	maxResponseBytes := defaultProxyProbeResponseMaxBytes
	if cfg != nil {
		insecure = cfg.Security.ProxyProbe.InsecureSkipVerify
		allowPrivate = cfg.Security.URLAllowlist.AllowPrivateHosts
		validateResolvedIP = cfg.Security.URLAllowlist.Enabled
		if cfg.Gateway.ProxyProbeResponseReadMaxBytes > 0 {
			maxResponseBytes = cfg.Gateway.ProxyProbeResponseReadMaxBytes
		}
	}
	if insecure {
		log.Printf("[ProxyProbe] Warning: insecure_skip_verify is not allowed and will cause probe failure.")
	}
	// 构建探测 URL 列表：配置存在时覆盖内置默认列表。
	var configuredTargets []configuredProbeTarget
	if cfg != nil && len(cfg.Security.ProxyProbe.URLs) > 0 {
		configuredTargets = make([]configuredProbeTarget, 0, len(cfg.Security.ProxyProbe.URLs))
		for _, u := range cfg.Security.ProxyProbe.URLs {
			configuredTargets = append(configuredTargets, configuredProbeTarget{
				url:    u.URL,
				parser: u.Parser,
			})
		}
	}

	return &proxyProbeService{
		insecureSkipVerify:  insecure,
		allowPrivateHosts:   allowPrivate,
		validateResolvedIP:  validateResolvedIP,
		maxResponseBytes:    maxResponseBytes,
		configuredProbeURLs: configuredTargets,
	}
}

const (
	defaultProxyProbeTimeout          = 10 * time.Second
	defaultProxyProbeResponseMaxBytes = int64(1024 * 1024)
)

// probeURLs 按优先级排列的探测 URL 列表。
// 代理只服务 AI 上游账号时，在线判断必须探测真实业务目标，不能使用
// ip-api/httpbin/gstatic 等无关站点，否则会把“只允许 OpenAI 的代理”误判离线。
var probeURLs = []struct {
	method string
	url    string
	parser string // "chatgpt_codex" or legacy parser names kept for unit-level parsers
}{
	{http.MethodHead, "https://chatgpt.com/backend-api/codex/responses", "chatgpt_codex"},
}

type configuredProbeTarget struct {
	url    string
	parser string
}

type proxyProbeService struct {
	insecureSkipVerify  bool
	allowPrivateHosts   bool
	validateResolvedIP  bool
	maxResponseBytes    int64
	configuredProbeURLs []configuredProbeTarget
}

func (s *proxyProbeService) ProbeProxy(ctx context.Context, proxyURL string) (*service.ProxyExitInfo, int64, error) {
	client, err := httpclient.GetClient(httpclient.Options{
		ProxyURL:           proxyURL,
		Timeout:            defaultProxyProbeTimeout,
		InsecureSkipVerify: s.insecureSkipVerify,
		ValidateResolvedIP: s.validateResolvedIP,
		AllowPrivateHosts:  s.allowPrivateHosts,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create proxy client: %w", err)
	}

	var lastErr error
	if len(s.configuredProbeURLs) > 0 {
		for _, probe := range s.configuredProbeURLs {
			exitInfo, latencyMs, err := s.probeWithURL(ctx, client, probe.url, http.MethodGet, probe.parser)
			if err == nil {
				return exitInfo, latencyMs, nil
			}
			lastErr = err
		}
		return nil, 0, fmt.Errorf("all probe URLs failed, last error: %w", lastErr)
	}

	for _, probe := range probeURLs {
		exitInfo, latencyMs, err := s.probeWithURL(ctx, client, probe.url, probe.method, probe.parser)
		if err == nil {
			return exitInfo, latencyMs, nil
		}
		lastErr = err
	}

	return nil, 0, fmt.Errorf("all probe URLs failed, last error: %w", lastErr)
}

func (s *proxyProbeService) probeWithURL(ctx context.Context, client *http.Client, url string, method string, parser string) (*service.ProxyExitInfo, int64, error) {
	startTime := time.Now()
	if method == "" {
		method = http.MethodGet
	}
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		if parser == "chatgpt_codex" || parser == "openai" {
			return nil, 0, formatCodexProbeError(method, url, err)
		}
		return nil, 0, fmt.Errorf("proxy connection failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	latencyMs := time.Since(startTime).Milliseconds()

	if parser == "chatgpt_codex" || parser == "openai" {
		// 这里判断的是出口到真实业务目标的 reachability，而不是携带账号凭据的
		// 完整 availability。无认证探测遇到 401/403/404/405/429 等 4xx
		// 仍说明 TLS、代理、目标域名和边缘服务链路已经打通。
		if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusInternalServerError {
			return &service.ProxyExitInfo{Country: parser}, latencyMs, nil
		}
		return nil, latencyMs, formatCodexProbeHTTPStatusError(method, url, resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, latencyMs, fmt.Errorf("request failed with status: %d", resp.StatusCode)
	}

	maxResponseBytes := s.maxResponseBytes
	if maxResponseBytes <= 0 {
		maxResponseBytes = defaultProxyProbeResponseMaxBytes
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, latencyMs, fmt.Errorf("failed to read response: %w", err)
	}
	if int64(len(body)) > maxResponseBytes {
		return nil, latencyMs, fmt.Errorf("proxy probe response exceeds limit: %d", maxResponseBytes)
	}

	switch parser {
	case "ip-api":
		return s.parseIPAPI(body, latencyMs)
	case "ipify":
		return s.parseIPify(body, latencyMs)
	case "chatgpt-trace":
		return s.parseChatGPTTrace(body, latencyMs)
	default:
		return nil, latencyMs, fmt.Errorf("unknown parser: %s", parser)
	}
}

func formatCodexProbeError(method string, rawURL string, err error) error {
	return fmt.Errorf(
		"codex probe target unreachable via proxy: method=%s target=%s error_type=%s detail=%v: %w",
		method,
		probeTargetHost(rawURL),
		classifyProxyProbeError(err),
		err,
		err,
	)
}

func formatCodexProbeHTTPStatusError(method string, rawURL string, statusCode int) error {
	return fmt.Errorf(
		"codex probe target unreachable via proxy: method=%s target=%s error_type=http_status failed with status: %d",
		method,
		probeTargetHost(rawURL),
		statusCode,
	)
}

func classifyProxyProbeError(err error) string {
	if err == nil {
		return "proxy_connect_failed"
	}
	message := strings.ToLower(err.Error())
	if errors.Is(err, io.EOF) || strings.Contains(message, "eof") {
		return "upstream_eof"
	}
	if strings.Contains(message, "tls handshake timeout") {
		return "tls_handshake_timeout"
	}
	if strings.Contains(message, "i/o timeout") || strings.Contains(message, "timeout") || errors.Is(err, context.DeadlineExceeded) {
		return "io_timeout"
	}
	return "proxy_connect_failed"
}

func probeTargetHost(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err == nil && parsed.Hostname() != "" {
		return parsed.Hostname()
	}
	return rawURL
}

func (s *proxyProbeService) parseIPAPI(body []byte, latencyMs int64) (*service.ProxyExitInfo, int64, error) {
	var ipInfo struct {
		Status      string `json:"status"`
		Message     string `json:"message"`
		Query       string `json:"query"`
		City        string `json:"city"`
		Region      string `json:"region"`
		RegionName  string `json:"regionName"`
		Country     string `json:"country"`
		CountryCode string `json:"countryCode"`
	}

	if err := json.Unmarshal(body, &ipInfo); err != nil {
		preview := string(body)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return nil, latencyMs, fmt.Errorf("failed to parse response: %w (body: %s)", err, preview)
	}
	if strings.ToLower(ipInfo.Status) != "success" {
		if ipInfo.Message == "" {
			ipInfo.Message = "ip-api request failed"
		}
		return nil, latencyMs, fmt.Errorf("ip-api request failed: %s", ipInfo.Message)
	}

	region := ipInfo.RegionName
	if region == "" {
		region = ipInfo.Region
	}
	return &service.ProxyExitInfo{
		IP:          ipInfo.Query,
		City:        ipInfo.City,
		Region:      region,
		Country:     ipInfo.Country,
		CountryCode: ipInfo.CountryCode,
	}, latencyMs, nil
}

func (s *proxyProbeService) parseIPify(body []byte, latencyMs int64) (*service.ProxyExitInfo, int64, error) {
	var result struct {
		IP string `json:"ip"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, latencyMs, fmt.Errorf("failed to parse ipify response: %w", err)
	}
	if result.IP == "" {
		return nil, latencyMs, fmt.Errorf("ipify: no IP found in response")
	}
	return &service.ProxyExitInfo{
		IP: result.IP,
	}, latencyMs, nil
}

// parseChatGPTTrace 解析 Cloudflare trace 端点（如 chatgpt.com/cdn-cgi/trace）的纯文本响应。
// 响应按行给出键值对，其中 ip= 为出口 IP，loc= 为国家代码。
func (s *proxyProbeService) parseChatGPTTrace(body []byte, latencyMs int64) (*service.ProxyExitInfo, int64, error) {
	var ip, loc string
	for _, line := range strings.Split(string(body), "\n") {
		key, value, found := strings.Cut(strings.TrimSpace(line), "=")
		if !found {
			continue
		}
		switch key {
		case "ip":
			ip = strings.TrimSpace(value)
		case "loc":
			loc = strings.TrimSpace(value)
		}
	}
	if ip == "" {
		preview := string(body)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return nil, latencyMs, fmt.Errorf("chatgpt-trace: no ip= found in response (body: %s)", preview)
	}
	info := &service.ProxyExitInfo{
		IP: ip,
	}
	if loc != "" {
		info.CountryCode = loc
	}
	return info, latencyMs, nil
}
