//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type initialProbeTaskRepoStub struct {
	created       []*AccountInitialProbeTask
	createErr     error
	createCreated bool
	claimQueue    []*AccountInitialProbeTask
	claimErr      error
	markSucceeded []initialProbeMarkCall
	markFailed    []initialProbeMarkCall
	markSkipped   []initialProbeMarkCall
}

type initialProbeMarkCall struct {
	taskID       int64
	attemptCount int
	lastError    string
}

func (s *initialProbeTaskRepoStub) CreateTaskIfAbsent(_ context.Context, task *AccountInitialProbeTask) (bool, error) {
	if task != nil {
		cloned := *task
		s.created = append(s.created, &cloned)
	}
	if s.createErr != nil {
		return false, s.createErr
	}
	return s.createCreated, nil
}

func (s *initialProbeTaskRepoStub) ClaimNextPendingTask(_ context.Context, _ int64) (*AccountInitialProbeTask, error) {
	if s.claimErr != nil {
		return nil, s.claimErr
	}
	if len(s.claimQueue) == 0 {
		return nil, nil
	}
	task := s.claimQueue[0]
	s.claimQueue = s.claimQueue[1:]
	return task, nil
}

func (s *initialProbeTaskRepoStub) MarkTaskSucceeded(_ context.Context, taskID int64, attemptCount int) error {
	s.markSucceeded = append(s.markSucceeded, initialProbeMarkCall{taskID: taskID, attemptCount: attemptCount})
	return nil
}

func (s *initialProbeTaskRepoStub) MarkTaskFailed(_ context.Context, taskID int64, attemptCount int, lastError string) error {
	s.markFailed = append(s.markFailed, initialProbeMarkCall{taskID: taskID, attemptCount: attemptCount, lastError: lastError})
	return nil
}

func (s *initialProbeTaskRepoStub) MarkTaskSkipped(_ context.Context, taskID int64, attemptCount int, lastError string) error {
	s.markSkipped = append(s.markSkipped, initialProbeMarkCall{taskID: taskID, attemptCount: attemptCount, lastError: lastError})
	return nil
}

type initialProbeRunnerStub struct {
	results []*ScheduledTestResult
	errs    []error
	calls   []initialProbeRunCall
}

type initialProbeRunCall struct {
	accountID int64
	modelID   string
}

func (s *initialProbeRunnerStub) RunTestBackground(_ context.Context, accountID int64, modelID string) (*ScheduledTestResult, error) {
	s.calls = append(s.calls, initialProbeRunCall{accountID: accountID, modelID: modelID})
	var result *ScheduledTestResult
	if len(s.results) > 0 {
		result = s.results[0]
		s.results = s.results[1:]
	}
	var err error
	if len(s.errs) > 0 {
		err = s.errs[0]
		s.errs = s.errs[1:]
	}
	return result, err
}

type initialProbeEnqueuerStub struct {
	calls []initialProbeEnqueueCall
	err   error
}

type initialProbeEnqueueCall struct {
	accountID     int64
	platform      string
	triggerSource string
}

func (s *initialProbeEnqueuerStub) EnqueueAccountInitialProbe(_ context.Context, accountID int64, platform, triggerSource string) error {
	s.calls = append(s.calls, initialProbeEnqueueCall{
		accountID:     accountID,
		platform:      platform,
		triggerSource: triggerSource,
	})
	return s.err
}

type accountRepoStubForInitialProbe struct {
	mockAccountRepoForGemini
	createID         int64
	created          *Account
	bulkUpdatedIDs   []int64
	bulkUpdatedState *string
}

func (s *accountRepoStubForInitialProbe) Create(_ context.Context, account *Account) error {
	cloned := *account
	if s.createID == 0 {
		s.createID = 1
	}
	account.ID = s.createID
	cloned.ID = s.createID
	s.created = &cloned
	return nil
}

func (s *accountRepoStubForInitialProbe) BulkUpdate(_ context.Context, ids []int64, updates AccountBulkUpdate) (int64, error) {
	s.bulkUpdatedIDs = append([]int64(nil), ids...)
	if updates.Status != nil {
		v := *updates.Status
		s.bulkUpdatedState = &v
	}
	return int64(len(ids)), nil
}

type crsSyncTestAccountRepoWithIDs struct {
	*crsSyncTestAccountRepo
	nextID int64
}

func (r *crsSyncTestAccountRepoWithIDs) Create(ctx context.Context, account *Account) error {
	r.nextID++
	account.ID = r.nextID
	return r.crsSyncTestAccountRepo.Create(ctx, account)
}

func TestAccountInitialProbeServiceEnqueueUsesExpectedModel(t *testing.T) {
	t.Run("openai uses gpt-5.4", func(t *testing.T) {
		repo := &initialProbeTaskRepoStub{createCreated: true}
		svc := NewAccountInitialProbeService(repo, nil, nil, nil)

		err := svc.EnqueueAccountInitialProbe(context.Background(), 101, PlatformOpenAI, "admin_create")
		require.NoError(t, err)
		require.Len(t, repo.created, 1)
		require.Equal(t, int64(101), repo.created[0].AccountID)
		require.Equal(t, "gpt-5.4", repo.created[0].ModelID)
		require.Equal(t, "admin_create", repo.created[0].TriggerSource)
		require.Equal(t, AccountInitialProbeStatusPending, repo.created[0].Status)
	})

	t.Run("non-openai keeps platform default", func(t *testing.T) {
		repo := &initialProbeTaskRepoStub{createCreated: true}
		svc := NewAccountInitialProbeService(repo, nil, nil, nil)

		err := svc.EnqueueAccountInitialProbe(context.Background(), 202, PlatformGemini, "admin_create")
		require.NoError(t, err)
		require.Len(t, repo.created, 1)
		require.Equal(t, "", repo.created[0].ModelID)
	})
}

func TestAdminServiceCreateAccountEnqueuesInitialProbe(t *testing.T) {
	accountRepo := &accountRepoStubForInitialProbe{createID: 42}
	enqueuer := &initialProbeEnqueuerStub{}
	svc := &adminServiceImpl{
		accountRepo:          accountRepo,
		initialProbeEnqueuer: enqueuer,
	}

	account, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:                 "new-openai-account",
		Platform:             PlatformOpenAI,
		Type:                 AccountTypeAPIKey,
		Credentials:          map[string]any{"api_key": "sk-test"},
		Concurrency:          3,
		SkipDefaultGroupBind: true,
	})
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Len(t, enqueuer.calls, 1)
	require.Equal(t, int64(42), enqueuer.calls[0].accountID)
	require.Equal(t, PlatformOpenAI, enqueuer.calls[0].platform)
	require.Equal(t, "admin_create", enqueuer.calls[0].triggerSource)
}

func TestCRSSyncServiceSyncFromCRSCreatedAccountEnqueuesInitialProbe(t *testing.T) {
	accountRepo := &crsSyncTestAccountRepoWithIDs{
		crsSyncTestAccountRepo: &crsSyncTestAccountRepo{},
	}
	proxyRepo := &crsSyncTestProxyRepo{}
	enqueuer := &initialProbeEnqueuerStub{}

	server := newCRSTestServer(t, map[string]any{
		"success": true,
		"data": map[string]any{
			"exportedAt":              time.Now().UTC().Format(time.RFC3339),
			"claudeAccounts":          []any{},
			"claudeConsoleAccounts":   []any{},
			"openaiOAuthAccounts":     []any{},
			"openaiResponsesAccounts": []map[string]any{{"kind": "openai-responses", "id": "crs-openai-1", "name": "openai-upstream", "platform": "openai", "isActive": true, "schedulable": true, "priority": 5, "status": "active", "credentials": map[string]any{"api_key": "sk-crs", "base_url": "https://api.openai.com"}}},
			"geminiOAuthAccounts":     []any{},
			"geminiApiKeyAccounts":    []any{},
		},
	})
	defer server.Close()

	svc := NewCRSSyncService(accountRepo, proxyRepo, nil, nil, nil, &config.Config{
		Security: config.SecurityConfig{
			URLAllowlist: config.URLAllowlistConfig{
				AllowInsecureHTTP: true,
			},
		},
	})
	svc.SetInitialProbeEnqueuer(enqueuer)

	result, err := svc.SyncFromCRS(context.Background(), SyncFromCRSInput{
		BaseURL:     server.URL,
		Username:    "admin",
		Password:    "secret",
		SyncProxies: false,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 1, result.Created)
	require.Len(t, enqueuer.calls, 1)
	require.Equal(t, int64(1), enqueuer.calls[0].accountID)
	require.Equal(t, PlatformOpenAI, enqueuer.calls[0].platform)
	require.Equal(t, "crs_sync", enqueuer.calls[0].triggerSource)
}

func TestAccountInitialProbeServiceRunOnceDisablesAccountAfterRetryableFailures(t *testing.T) {
	repo := &initialProbeTaskRepoStub{
		claimQueue: []*AccountInitialProbeTask{{
			ID:            7,
			AccountID:     99,
			ModelID:       "gpt-5.4",
			Status:        AccountInitialProbeStatusPending,
			TriggerSource: "admin_create",
		}},
	}
	accountRepo := &accountRepoStubForInitialProbe{
		mockAccountRepoForGemini: mockAccountRepoForGemini{
			accountsByID: map[int64]*Account{
				99: {ID: 99, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true},
			},
		},
	}
	runner := &initialProbeRunnerStub{
		results: []*ScheduledTestResult{
			{Status: "failed", ErrorMessage: "API returned 503: upstream unavailable"},
			{Status: "failed", ErrorMessage: "API returned 503: upstream unavailable"},
		},
	}
	svc := NewAccountInitialProbeService(repo, accountRepo, runner, nil)

	svc.runOnce()

	require.Len(t, runner.calls, 2)
	require.Len(t, repo.markFailed, 1)
	require.Equal(t, 7, int(repo.markFailed[0].taskID))
	require.Equal(t, 2, repo.markFailed[0].attemptCount)
	require.Contains(t, repo.markFailed[0].lastError, "503")
	require.Equal(t, []int64{99}, accountRepo.bulkUpdatedIDs)
	require.NotNil(t, accountRepo.bulkUpdatedState)
	require.Equal(t, StatusDisabled, *accountRepo.bulkUpdatedState)
}

func TestAccountInitialProbeServiceRunOnceSkipsManuallyDisabledAccount(t *testing.T) {
	repo := &initialProbeTaskRepoStub{
		claimQueue: []*AccountInitialProbeTask{{
			ID:        8,
			AccountID: 100,
			Status:    AccountInitialProbeStatusPending,
		}},
	}
	accountRepo := &accountRepoStubForInitialProbe{
		mockAccountRepoForGemini: mockAccountRepoForGemini{
			accountsByID: map[int64]*Account{
				100: {ID: 100, Platform: PlatformOpenAI, Status: StatusDisabled, Schedulable: true},
			},
		},
	}
	runner := &initialProbeRunnerStub{}
	svc := NewAccountInitialProbeService(repo, accountRepo, runner, nil)

	svc.runOnce()

	require.Empty(t, runner.calls)
	require.Len(t, repo.markSkipped, 1)
	require.Contains(t, repo.markSkipped[0].lastError, "account already disabled")
}

func TestAccountInitialProbeServiceRunOnceMarksSucceeded(t *testing.T) {
	repo := &initialProbeTaskRepoStub{
		claimQueue: []*AccountInitialProbeTask{{
			ID:        9,
			AccountID: 101,
			Status:    AccountInitialProbeStatusPending,
		}},
	}
	accountRepo := &accountRepoStubForInitialProbe{
		mockAccountRepoForGemini: mockAccountRepoForGemini{
			accountsByID: map[int64]*Account{
				101: {ID: 101, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true},
			},
		},
	}
	runner := &initialProbeRunnerStub{
		results: []*ScheduledTestResult{{Status: "success"}},
	}
	svc := NewAccountInitialProbeService(repo, accountRepo, runner, nil)

	svc.runOnce()

	require.Len(t, repo.markSucceeded, 1)
	require.Equal(t, 1, repo.markSucceeded[0].attemptCount)
	require.Nil(t, accountRepo.bulkUpdatedState)
}

func TestAccountInitialProbeServiceEnqueueReturnsRepositoryError(t *testing.T) {
	repo := &initialProbeTaskRepoStub{
		createErr: errors.New("boom"),
	}
	svc := NewAccountInitialProbeService(repo, nil, nil, nil)

	err := svc.EnqueueAccountInitialProbe(context.Background(), 303, PlatformOpenAI, "admin_create")
	require.Error(t, err)
}
