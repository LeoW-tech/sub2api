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

	pausedProxyIDs   []int64
	resumedProxyIDs  []int64
	pausedAccountIDs []int64
	restoredIDs      []int64
	clearedIDs       []int64

	pauseProxyErr   error
	resumeProxyErr  error
	pauseAccountErr error
	restoreErr      error
	clearErr        error
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

func (s *proxyNetworkAccountRepoStub) PauseAccountByNetwork(ctx context.Context, accountID int64) (bool, error) {
	s.pausedAccountIDs = append(s.pausedAccountIDs, accountID)
	return true, s.pauseAccountErr
}

func (s *proxyNetworkAccountRepoStub) RestoreAccountFromNetworkPause(ctx context.Context, accountID int64) (bool, error) {
	s.restoredIDs = append(s.restoredIDs, accountID)
	return true, s.restoreErr
}

func (s *proxyNetworkAccountRepoStub) ClearNetworkAutoPause(ctx context.Context, accountID int64) error {
	s.clearedIDs = append(s.clearedIDs, accountID)
	return s.clearErr
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
	exitInfo   *ProxyExitInfo
	latencyMs  int64
	err        error
	proxyURLs  []string
}

func (s *proxyExitInfoProberStub) ProbeProxy(ctx context.Context, proxyURL string) (*ProxyExitInfo, int64, error) {
	s.proxyURLs = append(s.proxyURLs, proxyURL)
	return s.exitInfo, s.latencyMs, s.err
}

func TestAdminService_TestProxy_OfflinePausesAccounts(t *testing.T) {
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

	require.Len(t, proxyRepo.updatedProxies, 1)
	updated := proxyRepo.updatedProxies[0]
	require.Equal(t, ProxyNetworkStatusOffline, updated.NetworkStatus)
	require.NotNil(t, updated.NetworkCheckedAt)
	require.Equal(t, "dial tcp timeout", updated.NetworkErrorMessage)
	require.Equal(t, []int64{7}, accountRepo.pausedProxyIDs)
	require.Empty(t, accountRepo.resumedProxyIDs)
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

func TestAdminService_ReconcileAccountProxyNetwork(t *testing.T) {
	t.Run("离线代理会暂停当前账号调度", func(t *testing.T) {
		accountRepo := &proxyNetworkAccountRepoStub{}
		proxyRepo := &proxyNetworkProxyRepoStub{
			proxies: map[int64]*Proxy{
				5: &Proxy{ID: 5, NetworkStatus: ProxyNetworkStatusOffline},
			},
		}
		svc := &adminServiceImpl{
			accountRepo: accountRepo,
			proxyRepo:   proxyRepo,
		}

		err := svc.reconcileAccountProxyNetwork(context.Background(), &Account{ID: 88, ProxyID: int64Ptr(5)}, nil)
		require.NoError(t, err)
		require.Equal(t, []int64{88}, accountRepo.pausedAccountIDs)
		require.Empty(t, accountRepo.restoredIDs)
		require.Empty(t, accountRepo.clearedIDs)
	})

	t.Run("在线代理会恢复此前因网络自动暂停的账号", func(t *testing.T) {
		accountRepo := &proxyNetworkAccountRepoStub{}
		proxyRepo := &proxyNetworkProxyRepoStub{
			proxies: map[int64]*Proxy{
				6: &Proxy{ID: 6, NetworkStatus: ProxyNetworkStatusOnline},
			},
		}
		svc := &adminServiceImpl{
			accountRepo: accountRepo,
			proxyRepo:   proxyRepo,
		}

		err := svc.reconcileAccountProxyNetwork(context.Background(), &Account{ID: 99, ProxyID: int64Ptr(6), NetworkAutoPaused: true}, nil)
		require.NoError(t, err)
		require.Equal(t, []int64{99}, accountRepo.restoredIDs)
		require.Empty(t, accountRepo.pausedAccountIDs)
		require.Empty(t, accountRepo.clearedIDs)
	})

	t.Run("无代理账号会清理自动暂停残留标记", func(t *testing.T) {
		accountRepo := &proxyNetworkAccountRepoStub{}
		svc := &adminServiceImpl{
			accountRepo: accountRepo,
		}

		err := svc.reconcileAccountProxyNetwork(context.Background(), &Account{ID: 100}, nil)
		require.NoError(t, err)
		require.Equal(t, []int64{100}, accountRepo.clearedIDs)
	})
}

func int64Ptr(v int64) *int64 {
	return &v
}
