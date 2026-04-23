//go:build unit

package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

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

func TestAccountBulkTestActivateService_Execute_ManualDetachedFromCallerContext(t *testing.T) {
	accountRepo := &accountRepoStubForBulkTestActivate{
		accountsByID: map[int64]*Account{
			1: {ID: 1, Status: "inactive"},
		},
	}
	accountTestSvc := &accountTestServiceStubForBulkTestActivate{
		results: map[int64]*ScheduledTestResult{
			1: {Status: "success"},
		},
	}
	telegram := &bulkTestActivateNotifierStub{}

	svc := NewAccountBulkTestActivateService(accountRepo, accountTestSvc, telegram, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	summary, err := svc.Execute(ctx, []int64{1}, AccountBulkTestActivateTriggerManual)
	require.NoError(t, err)
	require.NotNil(t, summary)
	require.Equal(t, 1, summary.Success)
	require.Equal(t, []int64{1}, summary.ActivatedIDs)
	require.Len(t, accountRepo.bulkUpdates, 1)
	require.Equal(t, "批量测试激活完成\n触发方式：manual\n测试总数：1\n成功：1\n失败：0\n新增启用：1\n新增禁用：0", telegram.lastMessage())
}

func TestAccountBulkTestActivateService_NotifySummary_UsesDetachedTimeoutContext(t *testing.T) {
	svc := NewAccountBulkTestActivateService(nil, nil, nil, nil)
	telegram := &bulkTestActivateNotifierStub{
		sendTextFn: func(ctx context.Context, text string) error {
			require.NoError(t, ctx.Err())
			deadline, ok := ctx.Deadline()
			require.True(t, ok)
			require.WithinDuration(t, time.Now().Add(accountBulkTestActivateNotifyTimeout), deadline, 2*time.Second)
			require.Contains(t, text, "批量测试激活完成")
			return nil
		},
	}
	svc.telegram = telegram

	parent, cancel := context.WithCancel(context.Background())
	cancel()

	svc.notifySummary(parent, &AccountBulkTestActivateSummary{
		Trigger:     string(AccountBulkTestActivateTriggerManual),
		Total:       1,
		Success:     1,
		Failed:      0,
		Activated:   1,
		Deactivated: 0,
	})

	require.Equal(t, 1, telegram.calls())
}

func TestAccountBulkTestActivateService_Execute_ScheduleKeepsCallerContext(t *testing.T) {
	accountRepo := &accountRepoScheduleContextStub{}
	accountTestSvc := &accountTestServiceStubForBulkTestActivate{}
	svc := NewAccountBulkTestActivateService(accountRepo, accountTestSvc, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	summary, err := svc.Execute(ctx, []int64{1}, AccountBulkTestActivateTriggerSchedule)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, summary)
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

type bulkTestActivateNotifierStub struct {
	mu         sync.Mutex
	messages   []string
	sendTextFn func(ctx context.Context, text string) error
}

func (s *bulkTestActivateNotifierStub) SendText(ctx context.Context, text string) error {
	s.mu.Lock()
	s.messages = append(s.messages, text)
	s.mu.Unlock()
	if s.sendTextFn != nil {
		return s.sendTextFn(ctx, text)
	}
	return nil
}

func (s *bulkTestActivateNotifierStub) calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.messages)
}

func (s *bulkTestActivateNotifierStub) lastMessage() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.messages) == 0 {
		return ""
	}
	return s.messages[len(s.messages)-1]
}

type accountRepoScheduleContextStub struct {
	accountRepoStub
}

func (s *accountRepoScheduleContextStub) GetByIDs(ctx context.Context, ids []int64) ([]*Account, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, nil
}
