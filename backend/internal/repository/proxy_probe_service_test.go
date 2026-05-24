package repository

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ProxyProbeServiceSuite struct {
	suite.Suite
	ctx      context.Context
	proxySrv *httptest.Server
	prober   *proxyProbeService
}

func (s *ProxyProbeServiceSuite) SetupTest() {
	s.ctx = context.Background()
	s.prober = &proxyProbeService{
		allowPrivateHosts: true,
	}
}

func (s *ProxyProbeServiceSuite) TearDownTest() {
	if s.proxySrv != nil {
		s.proxySrv.Close()
		s.proxySrv = nil
	}
}

func (s *ProxyProbeServiceSuite) setupProxyServer(handler http.HandlerFunc) {
	s.proxySrv = newLocalTestServer(s.T(), handler)
}

func (s *ProxyProbeServiceSuite) setupTLSServer(handler http.HandlerFunc) {
	s.proxySrv = httptest.NewTLSServer(handler)
}

func (s *ProxyProbeServiceSuite) TestProbeProxy_InvalidProxyURL() {
	_, _, err := s.prober.ProbeProxy(s.ctx, "://bad")
	require.Error(s.T(), err)
	require.ErrorContains(s.T(), err, "failed to create proxy client")
}

func (s *ProxyProbeServiceSuite) TestProbeProxy_UnsupportedProxyScheme() {
	_, _, err := s.prober.ProbeProxy(s.ctx, "ftp://127.0.0.1:1")
	require.Error(s.T(), err)
	require.ErrorContains(s.T(), err, "failed to create proxy client")
}

func (s *ProxyProbeServiceSuite) TestProbeURLsPreferChatGPTCodexResponses() {
	require.NotEmpty(s.T(), probeURLs)
	require.Equal(s.T(), "https://chatgpt.com/backend-api/codex/responses", probeURLs[0].url)
	for _, probe := range probeURLs {
		require.NotContains(s.T(), probe.url, "api.openai.com/v1/models")
	}
}

func (s *ProxyProbeServiceSuite) TestProbeProxy_Success_ChatGPTCodexReachability() {
	s.setupTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead && r.URL.Path == "/backend-api/codex/responses" {
			w.WriteHeader(http.StatusOK)
			return
		}
		// 其他请求返回错误
		w.WriteHeader(http.StatusServiceUnavailable)
	}))

	info, latencyMs, err := s.prober.probeWithURL(s.ctx, s.proxySrv.Client(), s.proxySrv.URL+"/backend-api/codex/responses", http.MethodHead, "chatgpt_codex")
	require.NoError(s.T(), err, "ProbeProxy")
	require.GreaterOrEqual(s.T(), latencyMs, int64(0), "unexpected latency")
	require.Equal(s.T(), "", info.IP)
	require.Equal(s.T(), "chatgpt_codex", info.Country)
}

func (s *ProxyProbeServiceSuite) TestProbeProxy_DoesNotUsePublicIPProbeFallback() {
	s.setupTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.RequestURI, "ip-api.com") || strings.Contains(r.RequestURI, "httpbin.org") {
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"origin": "5.6.7.8"}`)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))

	_, _, err := s.prober.ProbeProxy(s.ctx, s.proxySrv.URL)
	require.Error(s.T(), err)
	require.NotContains(s.T(), err.Error(), "httpbin.org")
	require.NotContains(s.T(), err.Error(), "ip-api.com")
}

func (s *ProxyProbeServiceSuite) TestProbeProxy_AllFailed() {
	s.setupProxyServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))

	_, _, err := s.prober.ProbeProxy(s.ctx, s.proxySrv.URL)
	require.Error(s.T(), err)
	require.ErrorContains(s.T(), err, "all probe URLs failed")
}

func (s *ProxyProbeServiceSuite) TestProbeProxy_ChatGPTCodexReachabilityAcceptsUnauthorized() {
	s.setupTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"error":{"message":"missing api key"}}`)
	}))

	info, latencyMs, err := s.prober.probeWithURL(s.ctx, s.proxySrv.Client(), s.proxySrv.URL+"/backend-api/codex/responses", http.MethodHead, "chatgpt_codex")
	require.NoError(s.T(), err)
	require.GreaterOrEqual(s.T(), latencyMs, int64(0), "unexpected latency")
	require.Equal(s.T(), "chatgpt_codex", info.Country)
}

func (s *ProxyProbeServiceSuite) TestProbeProxy_ChatGPTCodexReachabilityAcceptsMethodAndRateLimit4xx() {
	for _, statusCode := range []int{http.StatusMethodNotAllowed, http.StatusTooManyRequests} {
		s.Run(http.StatusText(statusCode), func() {
			info, latencyMs, err := s.prober.probeWithURL(
				s.ctx,
				&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					require.Equal(s.T(), http.MethodHead, req.Method)
					return &http.Response{
						StatusCode: statusCode,
						Body:       io.NopCloser(strings.NewReader("")),
						Header:     make(http.Header),
					}, nil
				})},
				"https://chatgpt.com/backend-api/codex/responses",
				http.MethodHead,
				"chatgpt_codex",
			)
			require.NoError(s.T(), err)
			require.GreaterOrEqual(s.T(), latencyMs, int64(0), "unexpected latency")
			require.Equal(s.T(), "chatgpt_codex", info.Country)
		})
	}
}

func (s *ProxyProbeServiceSuite) TestProbeProxy_ProxyServerClosed() {
	s.setupProxyServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	s.proxySrv.Close()

	_, _, err := s.prober.ProbeProxy(s.ctx, s.proxySrv.URL)
	require.Error(s.T(), err, "expected error when proxy server is closed")
}

func (s *ProxyProbeServiceSuite) TestParseIPAPI_Success() {
	body := []byte(`{"status":"success","query":"1.2.3.4","city":"Beijing","regionName":"Beijing","country":"China","countryCode":"CN"}`)
	info, latencyMs, err := s.prober.parseIPAPI(body, 100)
	require.NoError(s.T(), err)
	require.Equal(s.T(), int64(100), latencyMs)
	require.Equal(s.T(), "1.2.3.4", info.IP)
	require.Equal(s.T(), "Beijing", info.City)
	require.Equal(s.T(), "Beijing", info.Region)
	require.Equal(s.T(), "China", info.Country)
	require.Equal(s.T(), "CN", info.CountryCode)
}

func (s *ProxyProbeServiceSuite) TestParseIPAPI_Failure() {
	body := []byte(`{"status":"fail","message":"rate limited"}`)
	_, _, err := s.prober.parseIPAPI(body, 100)
	require.Error(s.T(), err)
	require.ErrorContains(s.T(), err, "rate limited")
}

func (s *ProxyProbeServiceSuite) TestParseHTTPBin_Success() {
	body := []byte(`{"origin": "9.8.7.6"}`)
	info, latencyMs, err := s.prober.parseHTTPBin(body, 50)
	require.NoError(s.T(), err)
	require.Equal(s.T(), int64(50), latencyMs)
	require.Equal(s.T(), "9.8.7.6", info.IP)
}

func (s *ProxyProbeServiceSuite) TestParseHTTPBin_NoIP() {
	body := []byte(`{"origin": ""}`)
	_, _, err := s.prober.parseHTTPBin(body, 50)
	require.Error(s.T(), err)
	require.ErrorContains(s.T(), err, "no IP found")
}

func TestProxyProbeServiceSuite(t *testing.T) {
	suite.Run(t, new(ProxyProbeServiceSuite))
}
