//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type updatingProxyRepoStub struct {
	*proxyRepoStub
	proxy       *Proxy
	created     *Proxy
	createCalls int
	updateCalls int
}

func (s *updatingProxyRepoStub) Create(_ context.Context, proxy *Proxy) error {
	s.createCalls++
	copy := *proxy
	s.created = &copy
	return nil
}

func (s *updatingProxyRepoStub) GetByID(context.Context, int64) (*Proxy, error) {
	copy := *s.proxy
	return &copy, nil
}

func (s *updatingProxyRepoStub) Update(_ context.Context, proxy *Proxy) error {
	s.updateCalls++
	copy := *proxy
	s.proxy = &copy
	return nil
}

func TestBothProxyUpdateServicesUseRepositoryUpdateBoundary(t *testing.T) {
	t.Run("ProxyService", func(t *testing.T) {
		repo := &updatingProxyRepoStub{
			proxyRepoStub: &proxyRepoStub{},
			proxy:         &Proxy{ID: 9, Protocol: "http", Host: "old.example", Port: 8080, Status: StatusActive},
		}
		svc := NewProxyService(repo)
		host := "new.example"

		_, err := svc.Update(context.Background(), 9, UpdateProxyRequest{Host: &host})

		require.NoError(t, err)
		require.Equal(t, 1, repo.updateCalls)
		require.Equal(t, host, repo.proxy.Host)
	})

	t.Run("adminService", func(t *testing.T) {
		repo := &updatingProxyRepoStub{
			proxyRepoStub: &proxyRepoStub{},
			proxy: &Proxy{
				ID:             9,
				Protocol:       "http",
				Host:           "old.example",
				Port:           8080,
				Status:         StatusActive,
				FallbackMode:   FallbackModeNone,
				ExpiryWarnDays: 7,
			},
		}
		svc := &adminServiceImpl{proxyRepo: repo}

		_, err := svc.UpdateProxy(context.Background(), 9, &UpdateProxyInput{
			Host:           "new.example",
			FallbackMode:   FallbackModeNone,
			ExpiryWarnDays: 7,
		})

		require.NoError(t, err)
		require.Equal(t, 1, repo.updateCalls)
		require.Equal(t, "new.example", repo.proxy.Host)
	})
}

func TestAdminProxyExternalKeyIsPersisted(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		repo := &updatingProxyRepoStub{proxyRepoStub: &proxyRepoStub{}}
		svc := &adminServiceImpl{proxyRepo: repo}

		_, err := svc.CreateProxy(context.Background(), &CreateProxyInput{
			Name:           "egress proxy",
			ExternalKey:    "egress-control:sub2api:test",
			Protocol:       "http",
			Host:           "host.docker.internal",
			Port:           65172,
			FallbackMode:   FallbackModeNone,
			ExpiryWarnDays: 7,
		})

		require.NoError(t, err)
		require.Equal(t, 1, repo.createCalls)
		require.Equal(t, "egress-control:sub2api:test", repo.created.ExternalKey)
	})

	t.Run("update", func(t *testing.T) {
		repo := &updatingProxyRepoStub{
			proxyRepoStub: &proxyRepoStub{},
			proxy: &Proxy{
				ID:             9,
				Protocol:       "http",
				Host:           "host.docker.internal",
				Port:           65172,
				Status:         StatusActive,
				FallbackMode:   FallbackModeNone,
				ExpiryWarnDays: 7,
			},
		}
		svc := &adminServiceImpl{proxyRepo: repo}
		externalKey := "egress-control:sub2api:test"

		_, err := svc.UpdateProxy(context.Background(), 9, &UpdateProxyInput{
			ExternalKey:    &externalKey,
			FallbackMode:   FallbackModeNone,
			ExpiryWarnDays: 7,
		})

		require.NoError(t, err)
		require.Equal(t, 1, repo.updateCalls)
		require.Equal(t, externalKey, repo.proxy.ExternalKey)
	})
}
