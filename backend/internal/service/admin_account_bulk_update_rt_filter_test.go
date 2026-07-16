package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type bulkUpdateRTFilterAccountRepo struct {
	AccountRepository
	rtStatus string
}

func (r *bulkUpdateRTFilterAccountRepo) ListWithFilters(
	_ context.Context,
	_ pagination.PaginationParams,
	_, _, _, _ string,
	_ int64,
	_, _, _ string,
	rtStatus string,
	_ string,
	_ []int64,
) ([]Account, *pagination.PaginationResult, error) {
	r.rtStatus = rtStatus
	return []Account{{ID: 7}}, &pagination.PaginationResult{Total: 1}, nil
}

func (r *bulkUpdateRTFilterAccountRepo) BulkUpdate(_ context.Context, ids []int64, _ AccountBulkUpdate) (int64, error) {
	return int64(len(ids)), nil
}

func TestAdminServiceBulkUpdateAccountsPreservesRTStatusFilter(t *testing.T) {
	repo := &bulkUpdateRTFilterAccountRepo{}
	svc := &adminServiceImpl{accountRepo: repo}
	schedulable := true

	result, err := svc.BulkUpdateAccounts(context.Background(), &BulkUpdateAccountsInput{
		Filters:     &BulkUpdateAccountFilters{RTStatus: "has_rt"},
		Schedulable: &schedulable,
	})

	require.NoError(t, err)
	require.Equal(t, "has_rt", repo.rtStatus)
	require.Equal(t, []int64{7}, result.SuccessIDs)
}
