//go:build unit

package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type proxyNetworkMonitorProxyRepoStub struct {
	proxies       map[int64]*Proxy
	updateFailIDs map[int64]error
	updated       []*Proxy
}

func (s *proxyNetworkMonitorProxyRepoStub) Create(ctx context.Context, proxy *Proxy) error {
	panic("unexpected Create call")
}

func (s *proxyNetworkMonitorProxyRepoStub) GetByID(ctx context.Context, id int64) (*Proxy, error) {
	proxy, ok := s.proxies[id]
	if !ok {
		return nil, ErrProxyNotFound
	}
	cp := *proxy
	return &cp, nil
}

func (s *proxyNetworkMonitorProxyRepoStub) ListByIDs(ctx context.Context, ids []int64) ([]Proxy, error) {
	panic("unexpected ListByIDs call")
}

func (s *proxyNetworkMonitorProxyRepoStub) Update(ctx context.Context, proxy *Proxy) error {
	if err, ok := s.updateFailIDs[proxy.ID]; ok {
		return err
	}
	cp := *proxy
	s.updated = append(s.updated, &cp)
	if s.proxies == nil {
		s.proxies = map[int64]*Proxy{}
	}
	s.proxies[proxy.ID] = &cp
	return nil
}

func (s *proxyNetworkMonitorProxyRepoStub) Delete(ctx context.Context, id int64) error {
	panic("unexpected Delete call")
}

func (s *proxyNetworkMonitorProxyRepoStub) List(ctx context.Context, params pagination.PaginationParams) ([]Proxy, *pagination.PaginationResult, error) {
	items := make([]Proxy, 0, len(s.proxies))
	for id := int64(1); id <= int64(len(s.proxies)); id++ {
		if proxy, ok := s.proxies[id]; ok {
			items = append(items, *proxy)
		}
	}
	return items, &pagination.PaginationResult{
		Total:    int64(len(items)),
		Page:     params.Page,
		PageSize: params.PageSize,
	}, nil
}

func (s *proxyNetworkMonitorProxyRepoStub) ListWithFilters(ctx context.Context, params pagination.PaginationParams, protocol, status, search string) ([]Proxy, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}

func (s *proxyNetworkMonitorProxyRepoStub) ListWithFiltersAndAccountCount(ctx context.Context, params pagination.PaginationParams, protocol, status, search string) ([]ProxyWithAccountCount, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFiltersAndAccountCount call")
}

func (s *proxyNetworkMonitorProxyRepoStub) ListActive(ctx context.Context) ([]Proxy, error) {
	panic("unexpected ListActive call")
}

func (s *proxyNetworkMonitorProxyRepoStub) ListActiveWithAccountCount(ctx context.Context) ([]ProxyWithAccountCount, error) {
	panic("unexpected ListActiveWithAccountCount call")
}

func (s *proxyNetworkMonitorProxyRepoStub) ExistsByHostPortAuth(ctx context.Context, host string, port int, username, password string) (bool, error) {
	panic("unexpected ExistsByHostPortAuth call")
}

func (s *proxyNetworkMonitorProxyRepoStub) CountAccountsByProxyID(ctx context.Context, proxyID int64) (int64, error) {
	panic("unexpected CountAccountsByProxyID call")
}

func (s *proxyNetworkMonitorProxyRepoStub) ListAccountSummariesByProxyID(ctx context.Context, proxyID int64) ([]ProxyAccountSummary, error) {
	panic("unexpected ListAccountSummariesByProxyID call")
}

type proxyNetworkMonitorProberStub struct{}

func (p *proxyNetworkMonitorProberStub) ProbeProxy(ctx context.Context, proxyURL string) (*ProxyExitInfo, int64, error) {
	switch {
	case strings.Contains(proxyURL, "127.0.0.1"):
		return &ProxyExitInfo{IP: "1.1.1.1", Country: "China", CountryCode: "CN"}, 111, nil
	case strings.Contains(proxyURL, "127.0.0.2"):
		return nil, 222, errors.New("network unreachable")
	default:
		return &ProxyExitInfo{IP: "8.8.8.8", Country: "United States", CountryCode: "US"}, 333, nil
	}
}

func TestProxyNetworkMonitorService_RunFullScan(t *testing.T) {
	accountRepo := &proxyNetworkAccountRepoStub{}
	proxyRepo := &proxyNetworkMonitorProxyRepoStub{
		proxies: map[int64]*Proxy{
			1: &Proxy{ID: 1, Name: "proxy-1", Protocol: "http", Host: "127.0.0.1", Port: 8081},
			2: &Proxy{ID: 2, Name: "proxy-2", Protocol: "http", Host: "127.0.0.2", Port: 8082},
			3: &Proxy{ID: 3, Name: "proxy-3", Protocol: "http", Host: "127.0.0.3", Port: 8083},
		},
		updateFailIDs: map[int64]error{
			3: fmt.Errorf("persist proxy state failed"),
		},
	}
	adminSvc := &adminServiceImpl{
		accountRepo: accountRepo,
		proxyRepo:   proxyRepo,
		proxyProber: &proxyNetworkMonitorProberStub{},
	}
	svc := NewProxyNetworkMonitorService(adminSvc, proxyRepo)

	summary, err := svc.RunFullScan(context.Background())
	require.NoError(t, err)
	require.NotNil(t, summary)
	require.Equal(t, 3, summary.Total)
	require.Equal(t, 1, summary.Online)
	require.Equal(t, 1, summary.Offline)
	require.Equal(t, 1, summary.Errors)
	require.Len(t, proxyRepo.updated, 2)
	require.Equal(t, []int64{1}, accountRepo.resumedProxyIDs)
	require.Equal(t, []int64{2}, accountRepo.pausedProxyIDs)

	lastSummary := svc.LastSummary()
	require.NotNil(t, lastSummary)
	require.Equal(t, summary.Total, lastSummary.Total)
	require.False(t, summary.StartedAt.IsZero())
	require.False(t, summary.FinishedAt.IsZero())
}

func TestProxyNetworkMonitorService_RunFullScan_PreventsOverlap(t *testing.T) {
	accountRepo := &proxyNetworkAccountRepoStub{}
	proxyRepo := &proxyNetworkMonitorProxyRepoStub{
		proxies: map[int64]*Proxy{
			1: &Proxy{ID: 1, Name: "proxy-1", Protocol: "http", Host: "127.0.0.1", Port: 8081},
		},
	}
	svc := NewProxyNetworkMonitorService(&adminServiceImpl{
		accountRepo: accountRepo,
		proxyRepo:   proxyRepo,
		proxyProber: &proxyNetworkMonitorProberStub{},
	}, proxyRepo)
	svc.scanRunning.Store(true)

	summary, err := svc.RunFullScan(context.Background())
	require.ErrorIs(t, err, ErrProxyNetworkScanRunning)
	require.Nil(t, summary)
}
