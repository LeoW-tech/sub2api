package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/robfig/cron/v3"
)

const (
	accountBulkTestActivateDefaultModel       = "gpt-5.4"
	accountBulkTestActivateDefaultConcurrency = 10
	accountBulkTestActivateSchedule           = "0 */6 * * *"
	accountBulkTestActivatePageSize           = 200
)

var accountBulkTestActivateCronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

type AccountBulkTestActivateTrigger string

const (
	AccountBulkTestActivateTriggerManual   AccountBulkTestActivateTrigger = "manual"
	AccountBulkTestActivateTriggerSchedule AccountBulkTestActivateTrigger = "schedule"
)

type AccountBulkTestActivateSummary struct {
	Trigger        string  `json:"trigger"`
	ModelID        string  `json:"model_id"`
	Total          int     `json:"total"`
	Success        int     `json:"success"`
	Failed         int     `json:"failed"`
	Activated      int     `json:"activated"`
	Deactivated    int     `json:"deactivated"`
	SuccessIDs     []int64 `json:"success_ids"`
	FailedIDs      []int64 `json:"failed_ids"`
	ActivatedIDs   []int64 `json:"activated_ids"`
	DeactivatedIDs []int64 `json:"deactivated_ids"`
}

type AccountBulkTestActivateService struct {
	accountRepo    AccountRepository
	accountTestSvc accountBulkTestRunner
	telegram       *TelegramNotificationService
	cfg            *config.Config

	cron      *cron.Cron
	startOnce sync.Once
	stopOnce  sync.Once
}

func NewAccountBulkTestActivateService(
	accountRepo AccountRepository,
	accountTestSvc accountBulkTestRunner,
	telegram *TelegramNotificationService,
	cfg *config.Config,
) *AccountBulkTestActivateService {
	return &AccountBulkTestActivateService{
		accountRepo:    accountRepo,
		accountTestSvc: accountTestSvc,
		telegram:       telegram,
		cfg:            cfg,
	}
}

type accountBulkTestRunner interface {
	RunTestBackground(ctx context.Context, accountID int64, modelID string) (*ScheduledTestResult, error)
}

func (s *AccountBulkTestActivateService) Start() {
	if s == nil || s.accountRepo == nil || s.accountTestSvc == nil {
		return
	}
	s.startOnce.Do(func() {
		loc := time.Local
		if s.cfg != nil {
			if parsed, err := time.LoadLocation(strings.TrimSpace(s.cfg.Timezone)); err == nil && parsed != nil {
				loc = parsed
			}
		}

		c := cron.New(cron.WithParser(accountBulkTestActivateCronParser), cron.WithLocation(loc))
		_, err := c.AddFunc(accountBulkTestActivateSchedule, func() { s.runScheduled() })
		if err != nil {
			logger.LegacyPrintf("service.account_bulk_test_activate", "[AccountBulkTestActivate] not started (invalid schedule=%q): %v", accountBulkTestActivateSchedule, err)
			return
		}
		s.cron = c
		s.cron.Start()
		logger.LegacyPrintf("service.account_bulk_test_activate", "[AccountBulkTestActivate] started (schedule=%q tz=%s)", accountBulkTestActivateSchedule, loc.String())
	})
}

func (s *AccountBulkTestActivateService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		if s.cron != nil {
			ctx := s.cron.Stop()
			select {
			case <-ctx.Done():
			case <-time.After(3 * time.Second):
				logger.LegacyPrintf("service.account_bulk_test_activate", "[AccountBulkTestActivate] cron stop timed out")
			}
		}
	})
}

func (s *AccountBulkTestActivateService) Execute(ctx context.Context, accountIDs []int64, trigger AccountBulkTestActivateTrigger) (*AccountBulkTestActivateSummary, error) {
	if s == nil || s.accountRepo == nil || s.accountTestSvc == nil {
		return nil, nil
	}
	accounts, err := s.resolveAccounts(ctx, accountIDs)
	if err != nil {
		return nil, err
	}

	summary := &AccountBulkTestActivateSummary{
		Trigger:        string(trigger),
		ModelID:        accountBulkTestActivateDefaultModel,
		Total:          len(accounts),
		SuccessIDs:     make([]int64, 0, len(accounts)),
		FailedIDs:      make([]int64, 0, len(accounts)),
		ActivatedIDs:   make([]int64, 0, len(accounts)),
		DeactivatedIDs: make([]int64, 0, len(accounts)),
	}
	if len(accounts) == 0 {
		s.notifySummary(ctx, summary)
		return summary, nil
	}

	originalStatus := make(map[int64]string, len(accounts))
	ids := make([]int64, 0, len(accounts))
	for _, account := range accounts {
		if account == nil {
			continue
		}
		originalStatus[account.ID] = account.Status
		ids = append(ids, account.ID)
	}

	type testOutcome struct {
		accountID int64
		success   bool
	}

	var (
		mu     sync.Mutex
		wg     sync.WaitGroup
		index  int
		runErr error
	)
	workerCount := accountBulkTestActivateDefaultConcurrency
	if workerCount > len(ids) {
		workerCount = len(ids)
	}
	if workerCount <= 0 {
		workerCount = 1
	}

	worker := func() {
		defer wg.Done()
		for {
			mu.Lock()
			if index >= len(ids) {
				mu.Unlock()
				return
			}
			accountID := ids[index]
			index++
			mu.Unlock()

			result, err := s.accountTestSvc.RunTestBackground(ctx, accountID, accountBulkTestActivateDefaultModel)
			outcome := testOutcome{
				accountID: accountID,
				success:   err == nil && result != nil && strings.EqualFold(strings.TrimSpace(result.Status), "success"),
			}

			mu.Lock()
			if err != nil && runErr == nil {
				runErr = err
			}
			if outcome.success {
				summary.SuccessIDs = append(summary.SuccessIDs, accountID)
			} else {
				summary.FailedIDs = append(summary.FailedIDs, accountID)
			}
			mu.Unlock()
		}
	}

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go worker()
	}
	wg.Wait()

	if runErr != nil && ctx.Err() != nil {
		return nil, ctx.Err()
	}

	summary.Success = len(summary.SuccessIDs)
	summary.Failed = len(summary.FailedIDs)

	for _, accountID := range summary.SuccessIDs {
		if !strings.EqualFold(originalStatus[accountID], StatusActive) {
			summary.ActivatedIDs = append(summary.ActivatedIDs, accountID)
		}
	}
	for _, accountID := range summary.FailedIDs {
		if strings.EqualFold(originalStatus[accountID], StatusActive) {
			summary.DeactivatedIDs = append(summary.DeactivatedIDs, accountID)
		}
	}

	if len(summary.ActivatedIDs) > 0 {
		status := StatusActive
		if _, err := s.accountRepo.BulkUpdate(ctx, summary.ActivatedIDs, AccountBulkUpdate{Status: &status}); err != nil {
			return nil, err
		}
	}
	if len(summary.DeactivatedIDs) > 0 {
		status := "inactive"
		if _, err := s.accountRepo.BulkUpdate(ctx, summary.DeactivatedIDs, AccountBulkUpdate{Status: &status}); err != nil {
			return nil, err
		}
	}

	sort.Slice(summary.SuccessIDs, func(i, j int) bool { return summary.SuccessIDs[i] < summary.SuccessIDs[j] })
	sort.Slice(summary.FailedIDs, func(i, j int) bool { return summary.FailedIDs[i] < summary.FailedIDs[j] })
	sort.Slice(summary.ActivatedIDs, func(i, j int) bool { return summary.ActivatedIDs[i] < summary.ActivatedIDs[j] })
	sort.Slice(summary.DeactivatedIDs, func(i, j int) bool { return summary.DeactivatedIDs[i] < summary.DeactivatedIDs[j] })
	summary.Activated = len(summary.ActivatedIDs)
	summary.Deactivated = len(summary.DeactivatedIDs)

	s.notifySummary(ctx, summary)
	return summary, nil
}

func (s *AccountBulkTestActivateService) runScheduled() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	summary, err := s.Execute(ctx, nil, AccountBulkTestActivateTriggerSchedule)
	if err != nil {
		logger.LegacyPrintf("service.account_bulk_test_activate", "[AccountBulkTestActivate] scheduled run failed: %v", err)
		return
	}
	if summary != nil {
		logger.LegacyPrintf(
			"service.account_bulk_test_activate",
			"[AccountBulkTestActivate] completed total=%d success=%d failed=%d activated=%d deactivated=%d",
			summary.Total,
			summary.Success,
			summary.Failed,
			summary.Activated,
			summary.Deactivated,
		)
	}
}

func (s *AccountBulkTestActivateService) resolveAccounts(ctx context.Context, accountIDs []int64) ([]*Account, error) {
	if len(accountIDs) > 0 {
		return s.accountRepo.GetByIDs(ctx, dedupeInt64s(accountIDs))
	}

	page := 1
	out := make([]*Account, 0, accountBulkTestActivatePageSize)
	for {
		items, pageInfo, err := s.accountRepo.ListWithFilters(
			ctx,
			pagination.PaginationParams{
				Page:      page,
				PageSize:  accountBulkTestActivatePageSize,
				SortBy:    "id",
				SortOrder: "asc",
			},
			"",
			"",
			"",
			"",
			0,
			"",
			"",
			"",
		)
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			break
		}
		for i := range items {
			account := items[i]
			out = append(out, &account)
		}
		if pageInfo != nil && int64(len(out)) >= pageInfo.Total {
			break
		}
		if len(items) < accountBulkTestActivatePageSize {
			break
		}
		page++
	}
	return out, nil
}

func (s *AccountBulkTestActivateService) notifySummary(ctx context.Context, summary *AccountBulkTestActivateSummary) {
	if s == nil || s.telegram == nil || summary == nil {
		return
	}
	message := fmt.Sprintf(
		"批量测试激活完成\n触发方式：%s\n测试总数：%d\n成功：%d\n失败：%d\n新增启用：%d\n新增禁用：%d",
		summary.Trigger,
		summary.Total,
		summary.Success,
		summary.Failed,
		summary.Activated,
		summary.Deactivated,
	)
	if err := s.telegram.SendText(ctx, message); err != nil {
		logger.LegacyPrintf("service.account_bulk_test_activate", "[AccountBulkTestActivate] telegram notify failed: %v", err)
	}
}

func dedupeInt64s(input []int64) []int64 {
	if len(input) == 0 {
		return nil
	}
	out := make([]int64, 0, len(input))
	seen := make(map[int64]struct{}, len(input))
	for _, id := range input {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
