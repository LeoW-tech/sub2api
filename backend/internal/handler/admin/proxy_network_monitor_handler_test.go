package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type proxyNetworkMonitorHandlerResponse struct {
	Code int `json:"code"`
	Data struct {
		Enabled         bool                                `json:"enabled"`
		Running         bool                                `json:"running"`
		ScanRunning     bool                                `json:"scan_running"`
		IntervalSeconds int                                 `json:"interval_seconds"`
		LastSummary     *proxyNetworkMonitorSummaryResponse `json:"last_summary"`
	} `json:"data"`
}

type proxyNetworkMonitorFake struct {
	running     bool
	scanRunning bool
	lastSummary *service.ProxyNetworkScanSummary
	startCalls  int
	stopCalls   int
}

func (s *proxyNetworkMonitorFake) Start() {
	s.startCalls++
	s.running = true
}

func (s *proxyNetworkMonitorFake) Stop() {
	s.stopCalls++
	s.running = false
	s.scanRunning = false
}

func (s *proxyNetworkMonitorFake) IsRunning() bool {
	return s.running
}

func (s *proxyNetworkMonitorFake) IsScanRunning() bool {
	return s.scanRunning
}

func (s *proxyNetworkMonitorFake) IntervalSeconds() int {
	return 300
}

func (s *proxyNetworkMonitorFake) LastSummary() *service.ProxyNetworkScanSummary {
	return s.lastSummary
}

type proxyNetworkMonitorResumerFake struct {
	*stubAdminService
	resumeCalls int
	resumeIDs   []int64
	err         error
}

func (s *proxyNetworkMonitorResumerFake) ResumeAllAccountsPausedByProxyNetwork(ctx context.Context) ([]int64, error) {
	s.resumeCalls++
	if s.err != nil {
		return nil, s.err
	}
	s.resumeIDs = []int64{101, 102}
	return s.resumeIDs, nil
}

func newProxyNetworkMonitorTestRouter(repo *testSettingRepo, monitor *proxyNetworkMonitorFake) *gin.Engine {
	return newProxyNetworkMonitorTestRouterWithAdmin(repo, monitor, nil)
}

func newProxyNetworkMonitorTestRouterWithAdmin(repo *testSettingRepo, monitor *proxyNetworkMonitorFake, adminService service.AdminService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	settingService := service.NewSettingService(repo, &config.Config{})
	handler := NewProxyHandler(adminService)
	handler.SetNetworkMonitorDependencies(settingService, monitor)

	router.GET("/api/v1/admin/proxies/network-monitor", handler.GetNetworkMonitorStatus)
	router.PUT("/api/v1/admin/proxies/network-monitor", handler.UpdateNetworkMonitorStatus)
	return router
}

func TestProxyHandler_GetNetworkMonitorStatus(t *testing.T) {
	repo := newTestSettingRepo()
	repo.values[service.SettingKeyProxyNetworkMonitorEnabled] = "true"
	startedAt := time.Date(2026, 5, 2, 8, 0, 0, 0, time.UTC)
	finishedAt := startedAt.Add(3 * time.Second)
	monitor := &proxyNetworkMonitorFake{
		running:     true,
		scanRunning: true,
		lastSummary: &service.ProxyNetworkScanSummary{
			StartedAt:  startedAt,
			FinishedAt: finishedAt,
			Total:      3,
			Online:     2,
			Offline:    1,
			Errors:     0,
		},
	}
	router := newProxyNetworkMonitorTestRouter(repo, monitor)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/proxies/network-monitor", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp proxyNetworkMonitorHandlerResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.True(t, resp.Data.Enabled)
	require.True(t, resp.Data.Running)
	require.True(t, resp.Data.ScanRunning)
	require.Equal(t, 300, resp.Data.IntervalSeconds)
	require.NotNil(t, resp.Data.LastSummary)
	require.Equal(t, 3, resp.Data.LastSummary.Total)
	require.Equal(t, 2, resp.Data.LastSummary.Online)
	require.Equal(t, 1, resp.Data.LastSummary.Offline)
}

func TestProxyHandler_UpdateNetworkMonitorStatus_StartsMonitor(t *testing.T) {
	repo := newTestSettingRepo()
	monitor := &proxyNetworkMonitorFake{}
	router := newProxyNetworkMonitorTestRouter(repo, monitor)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/proxies/network-monitor", bytes.NewBufferString(`{"enabled":true}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "true", repo.values[service.SettingKeyProxyNetworkMonitorEnabled])
	require.Equal(t, 1, monitor.startCalls)
	require.Zero(t, monitor.stopCalls)
	require.True(t, monitor.running)
}

func TestProxyHandler_UpdateNetworkMonitorStatus_StopsMonitor(t *testing.T) {
	repo := newTestSettingRepo()
	monitor := &proxyNetworkMonitorFake{running: true, scanRunning: true}
	resumer := &proxyNetworkMonitorResumerFake{stubAdminService: newStubAdminService()}
	router := newProxyNetworkMonitorTestRouterWithAdmin(repo, monitor, resumer)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/proxies/network-monitor", bytes.NewBufferString(`{"enabled":false}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "false", repo.values[service.SettingKeyProxyNetworkMonitorEnabled])
	require.Zero(t, monitor.startCalls)
	require.Equal(t, 1, monitor.stopCalls)
	require.False(t, monitor.running)
	require.False(t, monitor.scanRunning)
	require.Equal(t, 1, resumer.resumeCalls)
	require.Equal(t, []int64{101, 102}, resumer.resumeIDs)
}

func TestProxyHandler_GetNetworkMonitorStatus_MissingSettingDefaultsEnabled(t *testing.T) {
	repo := newTestSettingRepo()
	monitor := &proxyNetworkMonitorFake{}
	router := newProxyNetworkMonitorTestRouter(repo, monitor)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/proxies/network-monitor", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp proxyNetworkMonitorHandlerResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.True(t, resp.Data.Enabled)
}

func TestProxyHandler_UpdateNetworkMonitorStatus_RequiresEnabled(t *testing.T) {
	repo := newTestSettingRepo()
	router := newProxyNetworkMonitorTestRouter(repo, &proxyNetworkMonitorFake{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/proxies/network-monitor", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

var _ proxyNetworkMonitorController = (*proxyNetworkMonitorFake)(nil)
