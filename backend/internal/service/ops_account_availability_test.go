package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type opsAvailabilityAccountRepoStub struct {
	AccountRepository
	accounts []Account
}

func (r opsAvailabilityAccountRepoStub) ListWithFilters(ctx context.Context, params pagination.PaginationParams, platform, accountType, status, search string, groupID int64, privacyMode, networkStatus, exitIP, rtStatus, capacityStatus string, accountIDs []int64) ([]Account, *pagination.PaginationResult, error) {
	filtered := make([]Account, 0, len(r.accounts))
	for _, acc := range r.accounts {
		if platform != "" && acc.Platform != platform {
			continue
		}
		filtered = append(filtered, acc)
	}
	return filtered, &pagination.PaginationResult{Total: int64(len(filtered)), Page: 1, PageSize: len(filtered), Pages: 1}, nil
}

func TestOpsAccountAvailabilityCountsQuotaLimitedSeparately(t *testing.T) {
	now := time.Now().UTC()
	accounts := make([]Account, 0, 17)
	for i := 0; i < 17; i++ {
		accounts = append(accounts, Account{
			ID:          int64(i + 1),
			Name:        "quota",
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Extra: map[string]any{
				"codex_5h_used_percent": 99.0,
				"codex_5h_reset_at":     now.Add(time.Hour).Format(time.RFC3339),
			},
		})
	}
	svc := &OpsService{
		cfg:         &config.Config{Ops: config.OpsConfig{Enabled: true}},
		accountRepo: opsAvailabilityAccountRepoStub{accounts: accounts},
	}
	svc.storeAdvancedSettingsSnapshot(&OpsAdvancedSettings{OpenAIAccountQuotaAutoPause: OpsOpenAIAccountQuotaAutoPauseSettings{DefaultThreshold5h: 0.95}})

	platform, _, account, _, err := svc.GetAccountAvailabilityStats(context.Background(), PlatformOpenAI, nil)
	require.NoError(t, err)
	require.Equal(t, int64(17), platform[PlatformOpenAI].TotalAccounts)
	require.Equal(t, int64(0), platform[PlatformOpenAI].AvailableCount)
	require.Equal(t, int64(17), platform[PlatformOpenAI].QuotaLimitedCount)
	require.Len(t, account, 17)
	for _, item := range account {
		require.True(t, item.IsQuotaLimited)
		require.Equal(t, []string{"5h"}, item.QuotaLimitWindows)
	}
}

func TestOpsAccountAvailabilityRateLimitWinsQuotaCount(t *testing.T) {
	now := time.Now().UTC()
	rateReset := now.Add(30 * time.Minute)
	svc := &OpsService{
		cfg: &config.Config{Ops: config.OpsConfig{Enabled: true}},
		accountRepo: opsAvailabilityAccountRepoStub{accounts: []Account{{
			ID:               1,
			Name:             "rate-and-quota",
			Platform:         PlatformOpenAI,
			Type:             AccountTypeAPIKey,
			Status:           StatusActive,
			Schedulable:      true,
			RateLimitResetAt: &rateReset,
			Extra: map[string]any{
				"codex_7d_used_percent": 99.0,
				"codex_7d_reset_at":     now.Add(48 * time.Hour).Format(time.RFC3339),
			},
		}}},
	}
	svc.storeAdvancedSettingsSnapshot(&OpsAdvancedSettings{OpenAIAccountQuotaAutoPause: OpsOpenAIAccountQuotaAutoPauseSettings{DefaultThreshold7d: 0.95}})

	platform, _, account, _, err := svc.GetAccountAvailabilityStats(context.Background(), PlatformOpenAI, nil)
	require.NoError(t, err)
	require.Equal(t, int64(0), platform[PlatformOpenAI].AvailableCount)
	require.Equal(t, int64(1), platform[PlatformOpenAI].RateLimitCount)
	require.Equal(t, int64(0), platform[PlatformOpenAI].QuotaLimitedCount)
	require.True(t, account[1].IsRateLimited)
	require.True(t, account[1].IsQuotaLimited)
	require.Equal(t, []string{"7d"}, account[1].QuotaLimitWindows)
}
