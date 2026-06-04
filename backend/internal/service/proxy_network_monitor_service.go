package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	proxyNetworkMonitorInterval          = 5 * time.Minute
	proxyNetworkMonitorFailureRetryDelay = 2 * time.Minute
	proxyNetworkMonitorFailureThreshold  = 3
	proxyNetworkMonitorPageSize          = 200
	proxyNetworkMonitorConcurrency       = 5
)

var ErrProxyNetworkScanRunning = errors.New("proxy network scan already running")

type ProxyNetworkScanSummary struct {
	StartedAt  time.Time
	FinishedAt time.Time
	Total      int
	Online     int
	Offline    int
	Errors     int
}

type proxyNetworkMonitorNotifier interface {
	SendText(ctx context.Context, text string) error
}

type proxyNetworkMonitorAdminService interface {
	TestProxyForNetworkMonitor(ctx context.Context, id int64, pauseOnFailure bool) (*ProxyTestResult, error)
	ListAccounts(ctx context.Context, page, pageSize int, platform, accountType, status, search string, groupID int64, privacyMode, networkStatus, exitIP, rtStatus, capacityStatus string, sortBy, sortOrder string) ([]Account, int64, error)
}

type proxyNetworkMonitorFailureState struct {
	ConsecutiveFailures int
	LastError           string
	LastFailureAt       time.Time
	LastSuccessAt       time.Time
	NextRetryAt         time.Time
}

type ProxyNetworkMonitorService struct {
	adminService proxyNetworkMonitorAdminService
	proxyRepo    ProxyRepository
	notifier     proxyNetworkMonitorNotifier

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}

	running     atomic.Bool
	scanRunning atomic.Bool
	lastSummary atomic.Pointer[ProxyNetworkScanSummary]

	stateMu       sync.Mutex
	failureStates map[int64]proxyNetworkMonitorFailureState
}

func NewProxyNetworkMonitorService(adminService proxyNetworkMonitorAdminService, proxyRepo ProxyRepository, notifier proxyNetworkMonitorNotifier) *ProxyNetworkMonitorService {
	return &ProxyNetworkMonitorService{
		adminService:  adminService,
		proxyRepo:     proxyRepo,
		notifier:      notifier,
		failureStates: map[int64]proxyNetworkMonitorFailureState{},
	}
}

func (s *ProxyNetworkMonitorService) Start() {
	if s == nil || s.adminService == nil || s.proxyRepo == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running.Load() {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	s.cancel = cancel
	s.done = done
	s.running.Store(true)

	go func() {
		defer close(done)
		defer s.running.Store(false)
		s.runLoop(ctx)
	}()
}

func (s *ProxyNetworkMonitorService) Stop() {
	if s == nil {
		return
	}

	s.mu.Lock()
	cancel := s.cancel
	done := s.done
	if cancel == nil || done == nil {
		s.mu.Unlock()
		return
	}
	s.cancel = nil
	s.done = nil
	cancel()
	<-done
	s.mu.Unlock()
}

func (s *ProxyNetworkMonitorService) LastSummary() *ProxyNetworkScanSummary {
	if s == nil {
		return nil
	}
	return s.lastSummary.Load()
}

func (s *ProxyNetworkMonitorService) IsRunning() bool {
	if s == nil {
		return false
	}
	return s.running.Load()
}

func (s *ProxyNetworkMonitorService) IsScanRunning() bool {
	if s == nil {
		return false
	}
	return s.scanRunning.Load()
}

func (s *ProxyNetworkMonitorService) IntervalSeconds() int {
	return int(proxyNetworkMonitorInterval / time.Second)
}

func (s *ProxyNetworkMonitorService) RunFullScan(ctx context.Context) (*ProxyNetworkScanSummary, error) {
	return s.runFullScan(ctx, false, time.Now().UTC(), true)
}

func (s *ProxyNetworkMonitorService) runFullScan(ctx context.Context, startupGrace bool, now time.Time, fullScan bool) (*ProxyNetworkScanSummary, error) {
	if s == nil || s.adminService == nil || s.proxyRepo == nil {
		return nil, nil
	}
	if !s.scanRunning.CompareAndSwap(false, true) {
		return nil, ErrProxyNetworkScanRunning
	}
	defer s.scanRunning.Store(false)

	summary := &ProxyNetworkScanSummary{
		StartedAt: time.Now().UTC(),
	}

	proxies, err := s.listAllProxies(ctx)
	if err != nil {
		return nil, err
	}
	eligibleProxies := s.filterProxiesForScan(proxies, now, startupGrace || fullScan)
	summary.Total = len(eligibleProxies)

	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)
	sem := make(chan struct{}, proxyNetworkMonitorConcurrency)

	for _, proxy := range eligibleProxies {
		proxyID := proxy.ID
		wg.Add(1)
		go func() {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				mu.Lock()
				summary.Errors++
				mu.Unlock()
				return
			}
			defer func() { <-sem }()

			pauseOnFailure := !startupGrace && s.shouldPauseOnNextFailure(proxyID)
			result, testErr := s.adminService.TestProxyForNetworkMonitor(ctx, proxyID, pauseOnFailure)

			mu.Lock()
			defer mu.Unlock()
			if testErr != nil {
				summary.Errors++
				return
			}
			if result != nil && result.Success {
				s.clearFailureState(proxyID, now)
				summary.Online++
				return
			}

			errorMessage := "proxy network probe failed"
			if result != nil && result.Message != "" {
				errorMessage = result.Message
			}
			state := s.recordFailure(proxyID, errorMessage, now)
			if pauseOnFailure && state.ConsecutiveFailures >= proxyNetworkMonitorFailureThreshold {
				summary.Offline++
				return
			}
			summary.Errors++
		}()
	}

	wg.Wait()

	summary.FinishedAt = time.Now().UTC()
	s.lastSummary.Store(summary)
	slog.Info(
		"proxy_network_monitor.full_scan_completed",
		"started_at", summary.StartedAt,
		"finished_at", summary.FinishedAt,
		"total", summary.Total,
		"online", summary.Online,
		"offline", summary.Offline,
		"errors", summary.Errors,
	)
	s.notifySummary(ctx, summary)
	return summary, nil
}

func (s *ProxyNetworkMonitorService) runLoop(ctx context.Context) {
	run := func(trigger string, startupGrace bool, fullScan bool) {
		if ctx.Err() != nil {
			return
		}
		scanCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
		defer cancel()
		if _, err := s.runFullScan(scanCtx, startupGrace, time.Now().UTC(), fullScan); err != nil && !errors.Is(err, ErrProxyNetworkScanRunning) {
			slog.Warn("proxy_network_monitor.full_scan_failed", "trigger", trigger, "error", err)
		}
	}

	run("startup", true, true)
	nextFullScanAt := time.Now().UTC().Add(proxyNetworkMonitorInterval)

	for {
		now := time.Now().UTC()
		delay := s.nextScanDelay(now, nextFullScanAt)
		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
			runAt := time.Now().UTC()
			fullScan := !runAt.Before(nextFullScanAt)
			run("interval", false, fullScan)
			if fullScan {
				nextFullScanAt = time.Now().UTC().Add(proxyNetworkMonitorInterval)
			}
		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}

func (s *ProxyNetworkMonitorService) filterProxiesForScan(proxies []Proxy, now time.Time, force bool) []Proxy {
	if force {
		return proxies
	}
	out := make([]Proxy, 0, len(proxies))
	for _, proxy := range proxies {
		if s.shouldScanProxy(proxy.ID, now) {
			out = append(out, proxy)
		}
	}
	return out
}

func (s *ProxyNetworkMonitorService) shouldScanProxy(proxyID int64, now time.Time) bool {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	state, ok := s.failureStates[proxyID]
	if !ok || state.ConsecutiveFailures == 0 {
		return false
	}
	return !now.Before(state.NextRetryAt)
}

func (s *ProxyNetworkMonitorService) shouldPauseOnNextFailure(proxyID int64) bool {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	state := s.failureStates[proxyID]
	return state.ConsecutiveFailures+1 >= proxyNetworkMonitorFailureThreshold
}

func (s *ProxyNetworkMonitorService) recordFailure(proxyID int64, message string, now time.Time) proxyNetworkMonitorFailureState {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if s.failureStates == nil {
		s.failureStates = map[int64]proxyNetworkMonitorFailureState{}
	}
	state := s.failureStates[proxyID]
	state.ConsecutiveFailures++
	state.LastError = message
	state.LastFailureAt = now
	state.NextRetryAt = now.Add(proxyNetworkMonitorFailureRetryDelay)
	s.failureStates[proxyID] = state
	return state
}

func (s *ProxyNetworkMonitorService) clearFailureState(proxyID int64, now time.Time) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if s.failureStates == nil {
		return
	}
	state := s.failureStates[proxyID]
	state.ConsecutiveFailures = 0
	state.LastError = ""
	state.LastSuccessAt = now
	state.NextRetryAt = time.Time{}
	s.failureStates[proxyID] = state
}

func (s *ProxyNetworkMonitorService) nextScanDelay(now time.Time, nextFullScanAt time.Time) time.Duration {
	next := nextFullScanAt
	if next.IsZero() {
		next = now.Add(proxyNetworkMonitorInterval)
	}
	s.stateMu.Lock()
	for _, state := range s.failureStates {
		if state.ConsecutiveFailures <= 0 || state.NextRetryAt.IsZero() {
			continue
		}
		if state.NextRetryAt.Before(next) {
			next = state.NextRetryAt
		}
	}
	s.stateMu.Unlock()
	if !next.After(now) {
		return 0
	}
	return next.Sub(now)
}

func (s *ProxyNetworkMonitorService) listAllProxies(ctx context.Context) ([]Proxy, error) {
	page := 1
	out := make([]Proxy, 0, proxyNetworkMonitorPageSize)

	for {
		items, pageInfo, err := s.proxyRepo.List(ctx, pagination.PaginationParams{
			Page:      page,
			PageSize:  proxyNetworkMonitorPageSize,
			SortBy:    "id",
			SortOrder: "desc",
		})
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			break
		}
		out = append(out, items...)
		if pageInfo != nil && int64(len(out)) >= pageInfo.Total {
			break
		}
		if len(items) < proxyNetworkMonitorPageSize {
			break
		}
		page++
	}

	return out, nil
}

func (s *ProxyNetworkMonitorService) countNetworkPausedOfflineAccounts(ctx context.Context) (int, error) {
	if s == nil || s.adminService == nil {
		return 0, nil
	}

	page := 1
	totalCount := 0
	for {
		accounts, total, err := s.adminService.ListAccounts(ctx, page, proxyNetworkMonitorPageSize, "", "", "", "", 0, "", ProxyNetworkStatusOffline, "", "", "", "id", "asc")
		if err != nil {
			return 0, err
		}
		if len(accounts) == 0 {
			break
		}
		for i := range accounts {
			if accounts[i].NetworkAutoPaused && !accounts[i].Schedulable {
				totalCount++
			}
		}
		if int64(page*proxyNetworkMonitorPageSize) >= total {
			break
		}
		page++
	}
	return totalCount, nil
}

func (s *ProxyNetworkMonitorService) notifySummary(ctx context.Context, summary *ProxyNetworkScanSummary) {
	if s == nil || s.notifier == nil || summary == nil {
		return
	}
	pausedCount, err := s.countNetworkPausedOfflineAccounts(ctx)
	if err != nil {
		slog.Warn("proxy_network_monitor.count_paused_offline_accounts_failed", "error", err)
		return
	}
	if pausedCount == 0 {
		return
	}

	message := fmt.Sprintf(
		"网络检查完成\n代理总数：%d\n在线：%d\n离线：%d\n错误：%d\n网络异常且保持关闭调度的账号：%d",
		summary.Total,
		summary.Online,
		summary.Offline,
		summary.Errors,
		pausedCount,
	)
	if err := s.notifier.SendText(ctx, message); err != nil {
		slog.Warn("proxy_network_monitor.telegram_notify_failed", "error", err)
	}
}
