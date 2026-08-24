//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type proxyNetworkAccountRepoStub struct {
	accountRepoStub

	pausedProxyIDs  []int64
	resumedProxyIDs []int64

	pauseProxyErr  error
	resumeProxyErr error
}

func (s *proxyNetworkAccountRepoStub) PauseAccountsByProxyNetwork(ctx context.Context, proxyID int64) ([]int64, error) {
	s.pausedProxyIDs = append(s.pausedProxyIDs, proxyID)
	if s.pauseProxyErr != nil {
		return nil, s.pauseProxyErr
	}
	return []int64{101, 102}, nil
}

func (s *proxyNetworkAccountRepoStub) ResumeAccountsByProxyNetwork(ctx context.Context, proxyID int64) ([]int64, error) {
	s.resumedProxyIDs = append(s.resumedProxyIDs, proxyID)
	if s.resumeProxyErr != nil {
		return nil, s.resumeProxyErr
	}
	return []int64{101}, nil
}

func (s *proxyNetworkAccountRepoStub) ResumeAllAccountsPausedByProxyNetwork(ctx context.Context) ([]int64, error) {
	if s.resumeProxyErr != nil {
		return nil, s.resumeProxyErr
	}
	return []int64{101, 102}, nil
}

type proxyNetworkProxyRepoStub struct {
	proxyRepoStub

	proxies        map[int64]*Proxy
	updatedProxies []*Proxy
	updateErr      error
	getErr         error
}

func (s *proxyNetworkProxyRepoStub) GetByID(ctx context.Context, id int64) (*Proxy, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	proxy, ok := s.proxies[id]
	if !ok {
		return nil, ErrProxyNotFound
	}
	cp := *proxy
	return &cp, nil
}

func (s *proxyNetworkProxyRepoStub) Update(ctx context.Context, proxy *Proxy) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	cp := *proxy
	s.updatedProxies = append(s.updatedProxies, &cp)
	if s.proxies == nil {
		s.proxies = map[int64]*Proxy{}
	}
	s.proxies[proxy.ID] = &cp
	return nil
}

type proxyExitInfoProberStub struct {
	exitInfo  *ProxyExitInfo
	latencyMs int64
	err       error
	proxyURLs []string
}

func (s *proxyExitInfoProberStub) ProbeProxy(ctx context.Context, proxyURL string) (*ProxyExitInfo, int64, error) {
	s.proxyURLs = append(s.proxyURLs, proxyURL)
	return s.exitInfo, s.latencyMs, s.err
}

func TestAdminService_TestProxy_OfflineDoesNotPauseAccounts(t *testing.T) {
	accountRepo := &proxyNetworkAccountRepoStub{}
	proxyRepo := &proxyNetworkProxyRepoStub{
		proxies: map[int64]*Proxy{
			7: &Proxy{
				ID:       7,
				Name:     "proxy-7",
				Protocol: "http",
				Host:     "127.0.0.1",
				Port:     8080,
			},
		},
	}
	svc := &adminServiceImpl{
		accountRepo: accountRepo,
		proxyRepo:   proxyRepo,
		proxyProber: &proxyExitInfoProberStub{err: errors.New("dial tcp timeout")},
	}

	result, err := svc.TestProxy(context.Background(), 7)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.Success)
	require.Equal(t, "dial tcp timeout", result.Message)

	require.Empty(t, proxyRepo.updatedProxies)
	require.Empty(t, accountRepo.pausedProxyIDs)
	require.Empty(t, accountRepo.resumedProxyIDs)
}

func TestAdminService_TestProxyForNetworkMonitor_OfflineCanPauseAccounts(t *testing.T) {
	accountRepo := &proxyNetworkAccountRepoStub{}
	proxyRepo := &proxyNetworkProxyRepoStub{
		proxies: map[int64]*Proxy{
			7: &Proxy{
				ID:       7,
				Name:     "proxy-7",
				Protocol: "http",
				Host:     "127.0.0.1",
				Port:     8080,
			},
		},
	}
	svc := &adminServiceImpl{
		accountRepo: accountRepo,
		proxyRepo:   proxyRepo,
		proxyProber: &proxyExitInfoProberStub{err: errors.New("dial tcp timeout")},
	}

	result, err := svc.TestProxyForNetworkMonitor(context.Background(), 7, true)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.Success)
	require.Equal(t, "dial tcp timeout", result.Message)
	require.Equal(t, []int64{7}, accountRepo.pausedProxyIDs)
}

func TestAdminService_TestProxy_OnlineResumesAutoPausedAccounts(t *testing.T) {
	accountRepo := &proxyNetworkAccountRepoStub{}
	proxyRepo := &proxyNetworkProxyRepoStub{
		proxies: map[int64]*Proxy{
			9: &Proxy{
				ID:       9,
				Name:     "proxy-9",
				Protocol: "socks5",
				Host:     "10.0.0.9",
				Port:     1080,
			},
		},
	}
	svc := &adminServiceImpl{
		accountRepo: accountRepo,
		proxyRepo:   proxyRepo,
		proxyProber: &proxyExitInfoProberStub{
			exitInfo: &ProxyExitInfo{
				IP:          "1.1.1.1",
				City:        "Shanghai",
				Region:      "Shanghai",
				Country:     "China",
				CountryCode: "CN",
			},
			latencyMs: 123,
		},
	}

	result, err := svc.TestProxy(context.Background(), 9)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Success)
	require.Equal(t, int64(123), result.LatencyMs)
	require.Equal(t, "1.1.1.1", result.IPAddress)

	require.Len(t, proxyRepo.updatedProxies, 1)
	updated := proxyRepo.updatedProxies[0]
	require.Equal(t, ProxyNetworkStatusOnline, updated.NetworkStatus)
	require.NotNil(t, updated.NetworkCheckedAt)
	require.Equal(t, "", updated.NetworkErrorMessage)
	require.Equal(t, "1.1.1.1", updated.ExitIP)
	require.NotNil(t, updated.ExitIPCheckedAt)
	require.Equal(t, []int64{9}, accountRepo.resumedProxyIDs)
	require.Empty(t, accountRepo.pausedProxyIDs)
}

func int64Ptr(v int64) *int64 {
	return &v
}
