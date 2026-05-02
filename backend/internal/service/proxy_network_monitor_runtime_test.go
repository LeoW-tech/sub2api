package service

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type proxyNetworkMonitorRuntimeSettingRepo struct {
	values map[string]string
	err    error
}

func newProxyNetworkMonitorRuntimeSettingRepo(values map[string]string) *proxyNetworkMonitorRuntimeSettingRepo {
	if values == nil {
		values = map[string]string{}
	}
	return &proxyNetworkMonitorRuntimeSettingRepo{values: values}
}

func (s *proxyNetworkMonitorRuntimeSettingRepo) Get(ctx context.Context, key string) (*Setting, error) {
	value, err := s.GetValue(ctx, key)
	if err != nil {
		return nil, err
	}
	return &Setting{Key: key, Value: value}, nil
}

func (s *proxyNetworkMonitorRuntimeSettingRepo) GetValue(ctx context.Context, key string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	value, ok := s.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return value, nil
}

func (s *proxyNetworkMonitorRuntimeSettingRepo) Set(ctx context.Context, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = value
	return nil
}

func (s *proxyNetworkMonitorRuntimeSettingRepo) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	if s.err != nil {
		return nil, s.err
	}
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *proxyNetworkMonitorRuntimeSettingRepo) SetMultiple(ctx context.Context, settings map[string]string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *proxyNetworkMonitorRuntimeSettingRepo) GetAll(ctx context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *proxyNetworkMonitorRuntimeSettingRepo) Delete(ctx context.Context, key string) error {
	delete(s.values, key)
	return nil
}

func TestSettingService_GetProxyNetworkMonitorRuntime_DefaultsEnabled(t *testing.T) {
	settingService := NewSettingService(newProxyNetworkMonitorRuntimeSettingRepo(nil), &config.Config{})

	runtime := settingService.GetProxyNetworkMonitorRuntime(context.Background())

	require.True(t, runtime.Enabled)
}

func TestSettingService_GetProxyNetworkMonitorRuntime_ReadFailureDefaultsEnabled(t *testing.T) {
	repo := newProxyNetworkMonitorRuntimeSettingRepo(nil)
	repo.err = errors.New("database unavailable")
	settingService := NewSettingService(repo, &config.Config{})

	runtime := settingService.GetProxyNetworkMonitorRuntime(context.Background())

	require.True(t, runtime.Enabled)
}

func TestSettingService_SetProxyNetworkMonitorEnabled(t *testing.T) {
	repo := newProxyNetworkMonitorRuntimeSettingRepo(nil)
	settingService := NewSettingService(repo, &config.Config{})

	require.NoError(t, settingService.SetProxyNetworkMonitorEnabled(context.Background(), false))
	require.Equal(t, "false", repo.values[SettingKeyProxyNetworkMonitorEnabled])

	require.NoError(t, settingService.SetProxyNetworkMonitorEnabled(context.Background(), true))
	require.Equal(t, "true", repo.values[SettingKeyProxyNetworkMonitorEnabled])
}

type proxyNetworkMonitorRuntimeProxyRepo struct {
	listCalls int
	proxies   []Proxy
}

func (s *proxyNetworkMonitorRuntimeProxyRepo) Create(ctx context.Context, proxy *Proxy) error {
	panic("unexpected Create call")
}

func (s *proxyNetworkMonitorRuntimeProxyRepo) GetByID(ctx context.Context, id int64) (*Proxy, error) {
	return nil, ErrProxyNotFound
}

func (s *proxyNetworkMonitorRuntimeProxyRepo) ListByIDs(ctx context.Context, ids []int64) ([]Proxy, error) {
	panic("unexpected ListByIDs call")
}

func (s *proxyNetworkMonitorRuntimeProxyRepo) ListIPOptions(ctx context.Context) ([]ProxyIPOption, error) {
	panic("unexpected ListIPOptions call")
}

func (s *proxyNetworkMonitorRuntimeProxyRepo) Update(ctx context.Context, proxy *Proxy) error {
	return nil
}

func (s *proxyNetworkMonitorRuntimeProxyRepo) Delete(ctx context.Context, id int64) error {
	panic("unexpected Delete call")
}

func (s *proxyNetworkMonitorRuntimeProxyRepo) List(ctx context.Context, params pagination.PaginationParams) ([]Proxy, *pagination.PaginationResult, error) {
	s.listCalls++
	return s.proxies, &pagination.PaginationResult{Total: int64(len(s.proxies)), Page: params.Page, PageSize: params.PageSize}, nil
}

func (s *proxyNetworkMonitorRuntimeProxyRepo) ListWithFilters(ctx context.Context, params pagination.PaginationParams, protocol, status, search string) ([]Proxy, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}

func (s *proxyNetworkMonitorRuntimeProxyRepo) ListWithFiltersAndAccountCount(ctx context.Context, params pagination.PaginationParams, protocol, status, search string) ([]ProxyWithAccountCount, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFiltersAndAccountCount call")
}

func (s *proxyNetworkMonitorRuntimeProxyRepo) ListActive(ctx context.Context) ([]Proxy, error) {
	panic("unexpected ListActive call")
}

func (s *proxyNetworkMonitorRuntimeProxyRepo) ListActiveWithAccountCount(ctx context.Context) ([]ProxyWithAccountCount, error) {
	panic("unexpected ListActiveWithAccountCount call")
}

func (s *proxyNetworkMonitorRuntimeProxyRepo) ExistsByHostPortAuth(ctx context.Context, host string, port int, username, password string) (bool, error) {
	return false, nil
}

func (s *proxyNetworkMonitorRuntimeProxyRepo) CountAccountsByProxyID(ctx context.Context, proxyID int64) (int64, error) {
	return 0, nil
}

func (s *proxyNetworkMonitorRuntimeProxyRepo) ListAccountSummariesByProxyID(ctx context.Context, proxyID int64) ([]ProxyAccountSummary, error) {
	return nil, nil
}

type proxyNetworkMonitorRuntimeAdminService struct {
	AdminService
}

func (s *proxyNetworkMonitorRuntimeAdminService) TestProxy(ctx context.Context, proxyID int64) (*ProxyTestResult, error) {
	return &ProxyTestResult{Success: true}, nil
}

func (s *proxyNetworkMonitorRuntimeAdminService) ListAccounts(ctx context.Context, page, pageSize int, platform, accountType, status, search string, groupID int64, privacyMode, networkStatus, exitIP, capacityStatus string, sortBy, sortOrder string) ([]Account, int64, error) {
	return nil, 0, nil
}

type blockingProxyNetworkMonitorAdminService struct {
	AdminService

	started chan struct{}
	release chan struct{}
	once    sync.Once
	calls   atomic.Int32
}

func newBlockingProxyNetworkMonitorAdminService() *blockingProxyNetworkMonitorAdminService {
	return &blockingProxyNetworkMonitorAdminService{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
}

func (s *blockingProxyNetworkMonitorAdminService) TestProxy(ctx context.Context, proxyID int64) (*ProxyTestResult, error) {
	s.calls.Add(1)
	s.once.Do(func() { close(s.started) })
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-s.release:
		return &ProxyTestResult{Success: true}, nil
	}
}

func (s *blockingProxyNetworkMonitorAdminService) ListAccounts(ctx context.Context, page, pageSize int, platform, accountType, status, search string, groupID int64, privacyMode, networkStatus, exitIP, capacityStatus string, sortBy, sortOrder string) ([]Account, int64, error) {
	return nil, 0, nil
}

func TestProvideProxyNetworkMonitorService_StartsWhenEnabled(t *testing.T) {
	settingService := NewSettingService(newProxyNetworkMonitorRuntimeSettingRepo(map[string]string{
		SettingKeyProxyNetworkMonitorEnabled: "true",
	}), &config.Config{})
	proxyRepo := &proxyNetworkMonitorRuntimeProxyRepo{}

	svc := ProvideProxyNetworkMonitorService(&proxyNetworkMonitorRuntimeAdminService{}, proxyRepo, nil, settingService)
	t.Cleanup(svc.Stop)

	require.True(t, svc.IsRunning())
	require.Eventually(t, func() bool { return proxyRepo.listCalls > 0 }, time.Second, 10*time.Millisecond)
}

func TestProvideProxyNetworkMonitorService_DoesNotStartWhenDisabled(t *testing.T) {
	settingService := NewSettingService(newProxyNetworkMonitorRuntimeSettingRepo(map[string]string{
		SettingKeyProxyNetworkMonitorEnabled: "false",
	}), &config.Config{})
	proxyRepo := &proxyNetworkMonitorRuntimeProxyRepo{}

	svc := ProvideProxyNetworkMonitorService(&proxyNetworkMonitorRuntimeAdminService{}, proxyRepo, nil, settingService)
	t.Cleanup(svc.Stop)

	require.False(t, svc.IsRunning())
	require.Zero(t, proxyRepo.listCalls)
}

func TestProxyNetworkMonitorService_StartStopRestart(t *testing.T) {
	svc := NewProxyNetworkMonitorService(
		&proxyNetworkMonitorRuntimeAdminService{},
		&proxyNetworkMonitorRuntimeProxyRepo{},
		nil,
	)
	t.Cleanup(svc.Stop)

	require.Equal(t, 300, svc.IntervalSeconds())
	require.False(t, svc.IsRunning())

	svc.Start()
	require.True(t, svc.IsRunning())

	svc.Start()
	require.True(t, svc.IsRunning())

	svc.Stop()
	require.False(t, svc.IsRunning())

	svc.Start()
	require.True(t, svc.IsRunning())
}

func TestProxyNetworkMonitorService_StopCancelsRunningScanAndAllowsRestart(t *testing.T) {
	adminSvc := newBlockingProxyNetworkMonitorAdminService()
	proxyRepo := &proxyNetworkMonitorRuntimeProxyRepo{
		proxies: []Proxy{{ID: 1, Name: "proxy-1", Protocol: "http", Host: "127.0.0.1", Port: 8081}},
	}
	svc := NewProxyNetworkMonitorService(adminSvc, proxyRepo, nil)
	t.Cleanup(svc.Stop)

	svc.Start()
	require.True(t, svc.IsRunning())
	require.Eventually(t, func() bool {
		select {
		case <-adminSvc.started:
			return true
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)
	require.True(t, svc.IsScanRunning())

	stopDone := make(chan struct{})
	go func() {
		svc.Stop()
		close(stopDone)
	}()

	require.Eventually(t, func() bool {
		select {
		case <-stopDone:
			return true
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)
	require.False(t, svc.IsRunning())
	require.False(t, svc.IsScanRunning())

	callsBeforeRestart := adminSvc.calls.Load()
	close(adminSvc.release)
	svc.Start()
	require.True(t, svc.IsRunning())
	require.Eventually(t, func() bool {
		return adminSvc.calls.Load() > callsBeforeRestart
	}, time.Second, 10*time.Millisecond)
}
