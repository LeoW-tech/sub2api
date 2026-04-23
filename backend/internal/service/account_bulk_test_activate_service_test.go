//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

func TestAccountBulkTestActivateService_Execute_UpdatesStatusesSymmetrically(t *testing.T) {
	accountRepo := &accountRepoStubForBulkTestActivate{
		accountsByID: map[int64]*Account{
			1: {ID: 1, Status: "inactive"},
			2: {ID: 2, Status: StatusActive},
			3: {ID: 3, Status: StatusError},
		},
	}
	accountTestSvc := &accountTestServiceStubForBulkTestActivate{
		results: map[int64]*ScheduledTestResult{
			1: {Status: "success"},
			2: {Status: "failed"},
			3: {Status: "failed"},
		},
	}

	svc := NewAccountBulkTestActivateService(accountRepo, accountTestSvc, nil, nil)
	summary, err := svc.Execute(context.Background(), []int64{1, 2, 3}, AccountBulkTestActivateTriggerManual)
	require.NoError(t, err)
	require.Equal(t, 3, summary.Total)
	require.Equal(t, 1, summary.Success)
	require.Equal(t, 2, summary.Failed)
	require.Equal(t, []int64{1}, summary.ActivatedIDs)
	require.Equal(t, []int64{2}, summary.DeactivatedIDs)
	require.Len(t, accountRepo.bulkUpdates, 2)
	require.Equal(t, StatusActive, *accountRepo.bulkUpdates[0].status)
	require.Equal(t, []int64{1}, accountRepo.bulkUpdates[0].ids)
	require.Equal(t, "inactive", *accountRepo.bulkUpdates[1].status)
	require.Equal(t, []int64{2}, accountRepo.bulkUpdates[1].ids)
}

type accountRepoStubForBulkTestActivate struct {
	accountRepoStub
	accountsByID map[int64]*Account
	bulkUpdates  []bulkStatusUpdate
}

type bulkStatusUpdate struct {
	ids    []int64
	status *string
}

func (s *accountRepoStubForBulkTestActivate) GetByIDs(_ context.Context, ids []int64) ([]*Account, error) {
	result := make([]*Account, 0, len(ids))
	for _, id := range ids {
		if account, ok := s.accountsByID[id]; ok {
			result = append(result, account)
		}
	}
	return result, nil
}

func (s *accountRepoStubForBulkTestActivate) ListWithFilters(_ context.Context, _ pagination.PaginationParams, _, _, _, _ string, _ int64, _, _, _ string) ([]Account, *pagination.PaginationResult, error) {
	items := make([]Account, 0, len(s.accountsByID))
	for _, account := range s.accountsByID {
		if account != nil {
			items = append(items, *account)
		}
	}
	return items, &pagination.PaginationResult{Total: int64(len(items))}, nil
}

func (s *accountRepoStubForBulkTestActivate) BulkUpdate(_ context.Context, ids []int64, updates AccountBulkUpdate) (int64, error) {
	s.bulkUpdates = append(s.bulkUpdates, bulkStatusUpdate{
		ids:    append([]int64(nil), ids...),
		status: updates.Status,
	})
	return int64(len(ids)), nil
}

type accountTestServiceStubForBulkTestActivate struct {
	results map[int64]*ScheduledTestResult
	errors  map[int64]error
}

func (s *accountTestServiceStubForBulkTestActivate) RunTestBackground(_ context.Context, accountID int64, modelID string) (*ScheduledTestResult, error) {
	if modelID != accountBulkTestActivateDefaultModel {
		return nil, errors.New("unexpected model id")
	}
	if err, ok := s.errors[accountID]; ok {
		return nil, err
	}
	if result, ok := s.results[accountID]; ok {
		return result, nil
	}
	return &ScheduledTestResult{Status: "failed"}, nil
}
