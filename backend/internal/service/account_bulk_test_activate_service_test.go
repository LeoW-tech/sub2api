//go:build unit

package service

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

func TestAccountBulkTestActivateService_ExecutionConfig_UsesLockedParameters(t *testing.T) {
	svc := NewAccountBulkTestActivateService(nil, nil, nil, nil)

	manualCfg := svc.executionConfig(AccountBulkTestActivateTriggerManual)
	require.Equal(t, accountBulkTestActivateConcurrency, manualCfg.concurrency)
	require.Equal(t, accountBulkTestActivateManualTimeout, manualCfg.timeout)
	require.True(t, manualCfg.detachCaller)

	scheduledCfg := svc.executionConfig(AccountBulkTestActivateTriggerSchedule)
	require.Equal(t, accountBulkTestActivateConcurrency, scheduledCfg.concurrency)
	require.Equal(t, accountBulkTestActivateScheduledTimeout, scheduledCfg.timeout)
	require.False(t, scheduledCfg.detachCaller)
}

func TestAccountBulkTestActivateService_Execute_ActivatesSuccessesWithoutDeactivatingFailures(t *testing.T) {
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
	require.Equal(t, 3, summary.RequestedTotal)
	require.Equal(t, 3, summary.Total)
	require.Equal(t, 3, summary.Processed)
	require.Equal(t, 0, summary.Remaining)
	require.Equal(t, 1, summary.Success)
	require.Equal(t, 2, summary.Failed)
	require.Equal(t, 0, summary.Skipped)
	require.Equal(t, []int64{1}, summary.ActivatedIDs)
	require.Empty(t, summary.DeactivatedIDs)
	require.Len(t, accountRepo.bulkUpdates, 1)
	require.Equal(t, StatusActive, *accountRepo.bulkUpdates[0].status)
	require.Equal(t, []int64{1}, accountRepo.bulkUpdates[0].ids)
}

func TestAccountBulkTestActivateService_Execute_RetriesRetryableFailuresWithinRun(t *testing.T) {
	accountRepo := &accountRepoStubForBulkTestActivate{
		accountsByID: map[int64]*Account{
			1: {ID: 1, Status: "inactive"},
		},
	}
	accountTestSvc := &accountTestServiceStubForBulkTestActivate{
		resultSequences: map[int64][]*ScheduledTestResult{
			1: {
				{Status: "failed", ErrorMessage: "API returned 503: upstream unavailable"},
				{Status: "success"},
			},
		},
	}

	svc := NewAccountBulkTestActivateService(accountRepo, accountTestSvc, nil, nil)
	svc.retryDelayOverride = -1
	summary, err := svc.Execute(context.Background(), []int64{1}, AccountBulkTestActivateTriggerManual)

	require.NoError(t, err)
	require.Equal(t, 1, summary.Success)
	require.Equal(t, 0, summary.Failed)
	require.Equal(t, []int64{1}, summary.SuccessIDs)
	require.Equal(t, []int64{1}, summary.ActivatedIDs)
	require.Equal(t, []int64{1, 1}, accountTestSvc.calls())
	require.Len(t, summary.Results, 1)
	require.Equal(t, int64(1), summary.Results[0].AccountID)
	require.Equal(t, "success", summary.Results[0].Status)
	require.Equal(t, 2, summary.Results[0].Attempts)
	require.Equal(t, "transient", summary.Results[0].FailureCategory)
	require.Equal(t, "activate", summary.Results[0].Action)
}

func TestAccountBulkTestActivateService_Execute_RetriesOverloaded529FailuresWithinRun(t *testing.T) {
	accountRepo := &accountRepoStubForBulkTestActivate{
		accountsByID: map[int64]*Account{
			1: {ID: 1, Status: "inactive"},
		},
	}
	accountTestSvc := &accountTestServiceStubForBulkTestActivate{
		resultSequences: map[int64][]*ScheduledTestResult{
			1: {
				{Status: "failed", ErrorMessage: "API returned 529: overloaded, please retry later"},
				{Status: "success"},
			},
		},
	}

	svc := NewAccountBulkTestActivateService(accountRepo, accountTestSvc, nil, nil)
	svc.retryDelayOverride = -1
	summary, err := svc.Execute(context.Background(), []int64{1}, AccountBulkTestActivateTriggerManual)

	require.NoError(t, err)
	require.Equal(t, 1, summary.Success)
	require.Equal(t, 0, summary.Failed)
	require.Equal(t, []int64{1, 1}, accountTestSvc.calls())
	require.Len(t, summary.Results, 1)
	require.Equal(t, "transient", summary.Results[0].FailureCategory)
	require.Equal(t, 2, summary.Results[0].Attempts)
	require.Equal(t, []int64{1}, summary.ActivatedIDs)
}

func TestAccountBulkTestActivateService_Execute_DoesNotRetryAuthFailures(t *testing.T) {
	accountRepo := &accountRepoStubForBulkTestActivate{
		accountsByID: map[int64]*Account{
			1: {ID: 1, Status: "inactive"},
		},
	}
	accountTestSvc := &accountTestServiceStubForBulkTestActivate{
		resultSequences: map[int64][]*ScheduledTestResult{
			1: {
				{Status: "failed", ErrorMessage: "Authentication failed (401): token_invalidated"},
				{Status: "success"},
			},
		},
	}

	svc := NewAccountBulkTestActivateService(accountRepo, accountTestSvc, nil, nil)
	summary, err := svc.Execute(context.Background(), []int64{1}, AccountBulkTestActivateTriggerManual)

	require.NoError(t, err)
	require.Equal(t, 0, summary.Success)
	require.Equal(t, 1, summary.Failed)
	require.Equal(t, []int64{1}, summary.FailedIDs)
	require.Equal(t, []int64{1}, accountTestSvc.calls())
	require.Len(t, summary.Results, 1)
	require.Equal(t, "auth", summary.Results[0].FailureCategory)
	require.Equal(t, 1, summary.Results[0].Attempts)
	require.Equal(t, "noop", summary.Results[0].Action)
	require.Empty(t, accountRepo.bulkUpdates)
}

func TestAccountBulkTestActivateService_Execute_DoesNotRetryRateLimitFailures(t *testing.T) {
	accountRepo := &accountRepoStubForBulkTestActivate{
		accountsByID: map[int64]*Account{
			1: {ID: 1, Status: StatusActive},
		},
	}
	accountTestSvc := &accountTestServiceStubForBulkTestActivate{
		resultSequences: map[int64][]*ScheduledTestResult{
			1: {
				{Status: "failed", ErrorMessage: "API returned 429: temporarily unavailable"},
				{Status: "success"},
			},
		},
	}
	svc := NewAccountBulkTestActivateService(accountRepo, accountTestSvc, nil, nil)

	summary, err := svc.Execute(context.Background(), []int64{1}, AccountBulkTestActivateTriggerManual)
	require.NoError(t, err)
	require.Equal(t, 1, summary.Failed)
	require.Equal(t, []int64{1}, accountTestSvc.calls())
}

func TestAccountBulkTestActivateService_Execute_RecordsAuditRunAndResults(t *testing.T) {
	accountRepo := &accountRepoStubForBulkTestActivate{
		accountsByID: map[int64]*Account{
			1: {ID: 1, Status: "inactive"},
			2: {ID: 2, Status: StatusActive},
		},
	}
	accountTestSvc := &accountTestServiceStubForBulkTestActivate{
		results: map[int64]*ScheduledTestResult{
			1: {Status: "success"},
			2: {Status: "failed", ErrorMessage: "Request failed: unexpected EOF"},
		},
	}
	auditRepo := &bulkTestActivateAuditRepoStub{}

	svc := NewAccountBulkTestActivateService(accountRepo, accountTestSvc, nil, nil)
	svc.auditRepo = auditRepo
	svc.retryDelayOverride = -1

	summary, err := svc.Execute(context.Background(), []int64{1, 2}, AccountBulkTestActivateTriggerManual)

	require.NoError(t, err)
	require.NotZero(t, summary.RunID)
	require.Equal(t, summary.RunID, auditRepo.createdRun.ID)
	require.Equal(t, "manual", auditRepo.createdRun.Trigger)
	require.Len(t, auditRepo.savedResults, 2)
	require.Equal(t, summary.RunID, auditRepo.savedResults[0].RunID)
	require.Equal(t, summary.RunID, auditRepo.savedResults[1].RunID)
	require.ElementsMatch(t, []int64{1, 2}, []int64{auditRepo.savedResults[0].AccountID, auditRepo.savedResults[1].AccountID})
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
	require.Contains(t, telegram.lastMessage(), "批量测试激活完成")
	require.Contains(t, telegram.lastMessage(), "请求总数：1")
	require.Contains(t, telegram.lastMessage(), "已处理：1")
}

func TestAccountBulkTestActivateService_Execute_PreservesAccountTestStatusMutation(t *testing.T) {
	accountRepo := &accountRepoStubForBulkTestActivate{
		accountsByID: map[int64]*Account{
			1: {ID: 1, Status: StatusActive},
		},
	}
	accountTestSvc := &accountTestServiceStubForBulkTestActivate{
		runFn: func(ctx context.Context, accountID int64, modelID string) (*ScheduledTestResult, error) {
			require.Equal(t, int64(1), accountID)
			require.Equal(t, "gpt-5.6-sol", modelID)
			require.False(t, accountTestShouldSuppressStatusMutation(ctx))
			return &ScheduledTestResult{Status: "failed", ErrorMessage: "Authentication failed (401): token_invalidated"}, nil
		},
	}

	svc := NewAccountBulkTestActivateService(accountRepo, accountTestSvc, nil, nil)
	summary, err := svc.Execute(context.Background(), []int64{1}, AccountBulkTestActivateTriggerManual)

	require.NoError(t, err)
	require.Equal(t, 1, summary.Failed)
	require.Empty(t, accountRepo.bulkUpdates)
}

func TestAccountBulkTestActivateService_Execute_RecoversSuccessfulAccounts(t *testing.T) {
	accountRepo := &accountRepoStubForBulkTestActivate{
		accountsByID: map[int64]*Account{1: {ID: 1, Status: StatusError}},
	}
	accountTestSvc := &accountTestServiceStubForBulkTestActivate{
		results: map[int64]*ScheduledTestResult{1: {Status: "success"}},
	}
	recoverer := &accountBulkTestActivateRecoveryStub{}
	svc := NewAccountBulkTestActivateService(accountRepo, accountTestSvc, nil, nil)
	svc.rateLimitRecovery = recoverer

	summary, err := svc.Execute(context.Background(), []int64{1}, AccountBulkTestActivateTriggerManual)
	require.NoError(t, err)
	require.Equal(t, 1, summary.Success)
	require.Equal(t, []int64{1}, recoverer.accountIDs)
}

func TestAccountBulkTestActivateService_Execute_SkipsSoftDeletedAndFailsMissingIDs(t *testing.T) {
	accountRepo := &accountRepoStubForBulkTestActivate{
		accountsByID: map[int64]*Account{
			1: {ID: 1, Status: "inactive"},
		},
		softDeletedIDs: map[int64]struct{}{
			2: {},
		},
	}
	accountTestSvc := &accountTestServiceStubForBulkTestActivate{
		results: map[int64]*ScheduledTestResult{
			1: {Status: "success"},
		},
	}

	svc := NewAccountBulkTestActivateService(accountRepo, accountTestSvc, nil, nil)
	summary, err := svc.Execute(context.Background(), []int64{1, 2, 4}, AccountBulkTestActivateTriggerManual)
	require.NoError(t, err)
	require.Equal(t, 3, summary.RequestedTotal)
	require.Equal(t, 3, summary.Total)
	require.Equal(t, 1, summary.Processed)
	require.Equal(t, 1, summary.Success)
	require.Equal(t, 1, summary.Failed)
	require.Equal(t, 1, summary.Skipped)
	require.Equal(t, 0, summary.Remaining)
	require.Equal(t, []int64{4}, summary.FailedIDs)
	require.Equal(t, []int64{1}, summary.SuccessIDs)
	require.Equal(t, []int64{1}, summary.ActivatedIDs)
	require.Equal(t, []int64{1}, accountTestSvc.calls())
}

func TestAccountBulkTestActivateService_Execute_ScheduleFiltersSoftDeletedCandidates(t *testing.T) {
	accountRepo := &accountRepoStubForBulkTestActivate{
		listAccounts: []Account{
			{ID: 1, Status: "inactive"},
			{ID: 2, Status: "inactive"},
		},
		softDeletedIDs: map[int64]struct{}{
			2: {},
		},
	}
	accountTestSvc := &accountTestServiceStubForBulkTestActivate{
		results: map[int64]*ScheduledTestResult{
			1: {Status: "success"},
			2: {Status: "success"},
		},
	}

	svc := NewAccountBulkTestActivateService(accountRepo, accountTestSvc, nil, nil)
	summary, err := svc.Execute(context.Background(), nil, AccountBulkTestActivateTriggerSchedule)
	require.NoError(t, err)
	require.Equal(t, 2, summary.RequestedTotal)
	require.Equal(t, 2, summary.Total)
	require.Equal(t, 1, summary.Processed)
	require.Equal(t, 1, summary.Success)
	require.Equal(t, 0, summary.Failed)
	require.Equal(t, 1, summary.Skipped)
	require.Equal(t, 0, summary.Remaining)
	require.Equal(t, []int64{1}, accountTestSvc.calls())
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
		Trigger:        string(AccountBulkTestActivateTriggerManual),
		RequestedTotal: 1,
		Total:          1,
		Processed:      1,
		Success:        1,
		Activated:      1,
	})

	require.Equal(t, 1, telegram.callsCount())
}

func TestAccountBulkTestActivateService_Execute_TimeoutReturnsPartialSummaryAndStillNotifies(t *testing.T) {
	accountRepo := &accountRepoStubForBulkTestActivate{
		accountsByID: map[int64]*Account{
			1: {ID: 1, Status: "inactive"},
			2: {ID: 2, Status: StatusActive},
		},
	}
	accountTestSvc := &accountTestServiceStubForBulkTestActivate{
		runFn: func(ctx context.Context, accountID int64, modelID string) (*ScheduledTestResult, error) {
			require.Equal(t, accountBulkTestActivateDefaultModel, modelID)
			if accountID == 1 {
				return &ScheduledTestResult{Status: "success"}, nil
			}
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}
	telegram := &bulkTestActivateNotifierStub{}

	svc := NewAccountBulkTestActivateService(accountRepo, accountTestSvc, telegram, nil)
	svc.manualTimeoutOverride = 20 * time.Millisecond
	svc.concurrencyOverride = 2

	summary, err := svc.Execute(context.Background(), []int64{1, 2}, AccountBulkTestActivateTriggerManual)
	require.NoError(t, err)
	require.True(t, summary.TimedOut)
	require.Equal(t, 2, summary.Processed)
	require.Equal(t, 1, summary.Success)
	require.Equal(t, 1, summary.Failed)
	require.Equal(t, 0, summary.Skipped)
	require.Equal(t, 0, summary.Remaining)
	require.Contains(t, summary.ErrorMessage, context.DeadlineExceeded.Error())
	require.Equal(t, []int64{1}, summary.ActivatedIDs)
	require.Empty(t, summary.DeactivatedIDs)
	require.Len(t, accountRepo.bulkUpdates, 1)
	require.Contains(t, telegram.lastMessage(), "批量测试激活部分完成（超时）")
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

func TestAccountBulkTestActivateService_RunScheduled_SkipsWhenPreviousRunStillActive(t *testing.T) {
	accountRepo := &accountRepoStubForBulkTestActivate{}
	accountTestSvc := &accountTestServiceStubForBulkTestActivate{}
	telegram := &bulkTestActivateNotifierStub{}

	svc := NewAccountBulkTestActivateService(accountRepo, accountTestSvc, telegram, nil)
	svc.scheduledRunMu.Lock()
	svc.scheduledRunRunning = true
	svc.scheduledRunMu.Unlock()

	svc.runScheduled()

	require.Empty(t, accountTestSvc.calls())
	require.Contains(t, telegram.lastMessage(), "批量测试激活已跳过")
	require.Contains(t, telegram.lastMessage(), accountBulkTestActivateOverlapSkipReason)
}

type accountRepoStubForBulkTestActivate struct {
	accountRepoStub
	accountsByID    map[int64]*Account
	listAccounts    []Account
	softDeletedIDs  map[int64]struct{}
	bulkUpdates     []bulkStatusUpdate
	listWithFilters error
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

func (s *accountRepoStubForBulkTestActivate) ListWithFilters(_ context.Context, _ pagination.PaginationParams, _, _, _, _ string, _ int64, _, _, _, _, _ string, _ []int64) ([]Account, *pagination.PaginationResult, error) {
	if s.listWithFilters != nil {
		return nil, nil, s.listWithFilters
	}
	if len(s.listAccounts) > 0 {
		items := append([]Account(nil), s.listAccounts...)
		return items, &pagination.PaginationResult{Total: int64(len(items))}, nil
	}

	items := make([]Account, 0, len(s.accountsByID))
	for _, account := range s.accountsByID {
		if account != nil {
			items = append(items, *account)
		}
	}
	slices.SortFunc(items, func(a, b Account) int {
		switch {
		case a.ID < b.ID:
			return -1
		case a.ID > b.ID:
			return 1
		default:
			return 0
		}
	})
	return items, &pagination.PaginationResult{Total: int64(len(items))}, nil
}

func (s *accountRepoStubForBulkTestActivate) ListSoftDeletedIDs(_ context.Context, ids []int64) ([]int64, error) {
	out := make([]int64, 0)
	for _, id := range ids {
		if _, ok := s.softDeletedIDs[id]; ok {
			out = append(out, id)
		}
	}
	return out, nil
}

func (s *accountRepoStubForBulkTestActivate) BulkUpdate(_ context.Context, ids []int64, updates AccountBulkUpdate) (int64, error) {
	s.bulkUpdates = append(s.bulkUpdates, bulkStatusUpdate{
		ids:    append([]int64(nil), ids...),
		status: updates.Status,
	})
	return int64(len(ids)), nil
}

type accountTestServiceStubForBulkTestActivate struct {
	mu              sync.Mutex
	results         map[int64]*ScheduledTestResult
	resultSequences map[int64][]*ScheduledTestResult
	errors          map[int64]error
	runFn           func(ctx context.Context, accountID int64, modelID string) (*ScheduledTestResult, error)
	called          []int64
	callCounts      map[int64]int
}

type accountBulkTestActivateRecoveryStub struct {
	accountIDs []int64
}

func (s *accountBulkTestActivateRecoveryStub) RecoverAccountAfterSuccessfulTest(_ context.Context, accountID int64) (*SuccessfulTestRecoveryResult, error) {
	s.accountIDs = append(s.accountIDs, accountID)
	return &SuccessfulTestRecoveryResult{}, nil
}

func (s *accountTestServiceStubForBulkTestActivate) RunTestBackground(ctx context.Context, accountID int64, modelID string) (*ScheduledTestResult, error) {
	s.mu.Lock()
	s.called = append(s.called, accountID)
	if s.callCounts == nil {
		s.callCounts = make(map[int64]int)
	}
	callIndex := s.callCounts[accountID]
	s.callCounts[accountID]++
	s.mu.Unlock()

	if s.runFn != nil {
		return s.runFn(ctx, accountID, modelID)
	}
	if modelID != accountBulkTestActivateDefaultModel {
		return nil, errors.New("unexpected model id")
	}
	if err, ok := s.errors[accountID]; ok {
		return nil, err
	}
	if sequence := s.resultSequences[accountID]; len(sequence) > 0 {
		if callIndex < len(sequence) {
			return sequence[callIndex], nil
		}
		return sequence[len(sequence)-1], nil
	}
	if result, ok := s.results[accountID]; ok {
		return result, nil
	}
	return &ScheduledTestResult{Status: "failed"}, nil
}

func (s *accountTestServiceStubForBulkTestActivate) calls() []int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]int64(nil), s.called...)
}

type bulkTestActivateAuditRepoStub struct {
	createdRun   *AccountBulkTestActivateRun
	finishedRuns []*AccountBulkTestActivateRun
	savedResults []*AccountBulkTestActivateResult
}

func (s *bulkTestActivateAuditRepoStub) CreateRun(_ context.Context, run *AccountBulkTestActivateRun) (*AccountBulkTestActivateRun, error) {
	copyRun := *run
	copyRun.ID = 123
	s.createdRun = &copyRun
	return &copyRun, nil
}

func (s *bulkTestActivateAuditRepoStub) SaveResults(_ context.Context, results []*AccountBulkTestActivateResult) error {
	for _, result := range results {
		if result == nil {
			continue
		}
		copyResult := *result
		s.savedResults = append(s.savedResults, &copyResult)
	}
	return nil
}

func (s *bulkTestActivateAuditRepoStub) FinishRun(_ context.Context, run *AccountBulkTestActivateRun) error {
	copyRun := *run
	s.finishedRuns = append(s.finishedRuns, &copyRun)
	return nil
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

func (s *bulkTestActivateNotifierStub) callsCount() int {
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
