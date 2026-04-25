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
	proxyNetworkMonitorInterval    = 5 * time.Minute
	proxyNetworkMonitorPageSize    = 200
	proxyNetworkMonitorConcurrency = 5
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

type ProxyNetworkMonitorService struct {
	adminService AdminService
	proxyRepo    ProxyRepository
	notifier     proxyNetworkMonitorNotifier

	stopCh   chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup

	scanRunning atomic.Bool
	lastSummary atomic.Pointer[ProxyNetworkScanSummary]
}

func NewProxyNetworkMonitorService(adminService AdminService, proxyRepo ProxyRepository, notifier proxyNetworkMonitorNotifier) *ProxyNetworkMonitorService {
	return &ProxyNetworkMonitorService{
		adminService: adminService,
		proxyRepo:    proxyRepo,
		notifier:     notifier,
		stopCh:       make(chan struct{}),
	}
}

func (s *ProxyNetworkMonitorService) Start() {
	if s == nil || s.adminService == nil || s.proxyRepo == nil {
		return
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runLoop()
	}()
}

func (s *ProxyNetworkMonitorService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

func (s *ProxyNetworkMonitorService) LastSummary() *ProxyNetworkScanSummary {
	if s == nil {
		return nil
	}
	return s.lastSummary.Load()
}

func (s *ProxyNetworkMonitorService) RunFullScan(ctx context.Context) (*ProxyNetworkScanSummary, error) {
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
	summary.Total = len(proxies)

	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)
	sem := make(chan struct{}, proxyNetworkMonitorConcurrency)

	for _, proxy := range proxies {
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

			result, testErr := s.adminService.TestProxy(ctx, proxyID)

			mu.Lock()
			defer mu.Unlock()
			if testErr != nil {
				summary.Errors++
				return
			}
			if result != nil && result.Success {
				summary.Online++
				return
			}
			summary.Offline++
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

func (s *ProxyNetworkMonitorService) runLoop() {
	run := func(trigger string) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		if _, err := s.RunFullScan(ctx); err != nil && !errors.Is(err, ErrProxyNetworkScanRunning) {
			slog.Warn("proxy_network_monitor.full_scan_failed", "trigger", trigger, "error", err)
		}
	}

	run("startup")

	ticker := time.NewTicker(proxyNetworkMonitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			run("interval")
		case <-s.stopCh:
			return
		}
	}
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
		accounts, total, err := s.adminService.ListAccounts(ctx, page, proxyNetworkMonitorPageSize, "", "", "", "", 0, "", ProxyNetworkStatusOffline, "", "", "id", "asc")
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
