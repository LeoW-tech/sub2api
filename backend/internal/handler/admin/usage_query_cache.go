package admin

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
)

var adminUsageStatsCache = newSnapshotCache(30 * time.Second)

type adminUsageStatsCacheKey struct {
	UserID      int64  `json:"user_id"`
	APIKeyID    int64  `json:"api_key_id"`
	AccountID   int64  `json:"account_id"`
	GroupID     int64  `json:"group_id"`
	Model       string `json:"model"`
	BillingMode string `json:"billing_mode"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	RequestType *int16 `json:"request_type"`
	Stream      *bool  `json:"stream"`
	BillingType *int8  `json:"billing_type"`
}

func (h *UsageHandler) getStatsCached(
	ctx context.Context,
	filters usagestats.UsageLogFilters,
) (*usagestats.UsageStats, bool, error) {
	startTime := ""
	if filters.StartTime != nil {
		startTime = filters.StartTime.UTC().Format(time.RFC3339)
	}
	endTime := ""
	if filters.EndTime != nil {
		endTime = filters.EndTime.UTC().Format(time.RFC3339)
	}

	key := mustMarshalDashboardCacheKey(adminUsageStatsCacheKey{
		UserID:      filters.UserID,
		APIKeyID:    filters.APIKeyID,
		AccountID:   filters.AccountID,
		GroupID:     filters.GroupID,
		Model:       filters.Model,
		BillingMode: filters.BillingMode,
		StartTime:   startTime,
		EndTime:     endTime,
		RequestType: filters.RequestType,
		Stream:      filters.Stream,
		BillingType: filters.BillingType,
	})

	entry, hit, err := adminUsageStatsCache.GetOrLoad(key, func() (any, error) {
		return h.usageService.GetStatsWithFilters(ctx, filters)
	})
	if err != nil {
		return nil, hit, err
	}

	stats, err := snapshotPayloadAs[*usagestats.UsageStats](entry.Payload)
	return stats, hit, err
}
