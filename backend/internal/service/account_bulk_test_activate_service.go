package service

import (
	"context"
	"errors"
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
	accountBulkTestActivateDefaultModel      = "gpt-5.4"
	accountBulkTestActivateConcurrency       = 80
	accountBulkTestActivateSchedule          = "0 */6 * * *"
	accountBulkTestActivatePageSize          = 200
	accountBulkTestActivateManualTimeout     = 45 * time.Minute
	accountBulkTestActivateScheduledTimeout  = 120 * time.Minute
	accountBulkTestActivateFinalizeTimeout   = 30 * time.Second
	accountBulkTestActivateNotifyTimeout     = 20 * time.Second
	accountBulkTestActivateOverlapSkipReason = "上一轮仍在运行，跳过本轮执行"
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
	RequestedTotal int     `json:"requested_total"`
	Total          int     `json:"total"`
	Processed      int     `json:"processed"`
	Remaining      int     `json:"remaining"`
	Success        int     `json:"success"`
	Failed         int     `json:"failed"`
	Skipped        int     `json:"skipped"`
	Activated      int     `json:"activated"`
	Deactivated    int     `json:"deactivated"`
	TimedOut       bool    `json:"timed_out"`
	RunSkipped     bool    `json:"run_skipped"`
	DurationMs     int64   `json:"duration_ms"`
	ErrorMessage   string  `json:"error_message,omitempty"`
	SuccessIDs     []int64 `json:"success_ids"`
	FailedIDs      []int64 `json:"failed_ids"`
	ActivatedIDs   []int64 `json:"activated_ids"`
	DeactivatedIDs []int64 `json:"deactivated_ids"`
}

type AccountBulkTestActivateService struct {
	accountRepo    AccountRepository
	accountTestSvc accountBulkTestRunner
	telegram       accountBulkTestActivateNotifier
	cfg            *config.Config

	cron      *cron.Cron
	startOnce sync.Once
	stopOnce  sync.Once

	scheduledRunMu      sync.Mutex
	scheduledRunRunning bool

	concurrencyOverride      int
	manualTimeoutOverride    time.Duration
	scheduledTimeoutOverride time.Duration
	finalizeTimeoutOverride  time.Duration
	notifyTimeoutOverride    time.Duration
}

func NewAccountBulkTestActivateService(
	accountRepo AccountRepository,
	accountTestSvc accountBulkTestRunner,
	telegram accountBulkTestActivateNotifier,
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

type accountBulkTestActivateNotifier interface {
	SendText(ctx context.Context, text string) error
}

type accountBulkTestActivateSoftDeleteInspector interface {
	ListSoftDeletedIDs(ctx context.Context, ids []int64) ([]int64, error)
}

type accountBulkTestActivateExecutionConfig struct {
	concurrency  int
	timeout      time.Duration
	detachCaller bool
}

type accountBulkTestActivateResolution struct {
	requestedIDs []int64
	accounts     []*Account
	skippedIDs   []int64
	missingIDs   []int64
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

	execCfg := s.executionConfig(trigger)
	execCtx, cancel := s.executionContext(ctx, execCfg)
	defer cancel()

	startedAt := time.Now()
	summary, err := s.execute(execCtx, accountIDs, trigger, execCfg)
	if summary != nil {
		summary.DurationMs = time.Since(startedAt).Milliseconds()
	}
	return summary, err
}

func (s *AccountBulkTestActivateService) execute(ctx context.Context, accountIDs []int64, trigger AccountBulkTestActivateTrigger, execCfg accountBulkTestActivateExecutionConfig) (*AccountBulkTestActivateSummary, error) {
	resolution, err := s.resolveAccounts(ctx, accountIDs)
	if err != nil {
		return nil, err
	}

	summary := &AccountBulkTestActivateSummary{
		Trigger:        string(trigger),
		ModelID:        accountBulkTestActivateDefaultModel,
		RequestedTotal: len(resolution.requestedIDs),
		Total:          len(resolution.requestedIDs),
		Skipped:        len(resolution.skippedIDs),
		SuccessIDs:     make([]int64, 0, len(resolution.accounts)),
		FailedIDs:      append([]int64(nil), resolution.missingIDs...),
		ActivatedIDs:   make([]int64, 0, len(resolution.accounts)),
		DeactivatedIDs: make([]int64, 0, len(resolution.accounts)),
	}

	logger.LegacyPrintf(
		"service.account_bulk_test_activate",
		"[AccountBulkTestActivate] started trigger=%s requested_total=%d live_total=%d skipped=%d concurrency=%d timeout=%s",
		summary.Trigger,
		summary.RequestedTotal,
		len(resolution.accounts),
		summary.Skipped,
		execCfg.concurrency,
		execCfg.timeout,
	)

	if len(resolution.accounts) == 0 {
		summary.Failed = len(summary.FailedIDs)
		summary.Remaining = accountBulkTestActivateRemaining(summary)
		s.logSummary("completed", summary)
		s.notifySummary(ctx, summary)
		return summary, nil
	}

	originalStatus := make(map[int64]string, len(resolution.accounts))
	ids := make([]int64, 0, len(resolution.accounts))
	for _, account := range resolution.accounts {
		if account == nil {
			continue
		}
		originalStatus[account.ID] = account.Status
		ids = append(ids, account.ID)
	}

	var (
		mu            sync.Mutex
		wg            sync.WaitGroup
		index         int
		processed     int
		testedSuccess []int64
		testedFailed  []int64
	)
	workerCount := execCfg.concurrency
	if workerCount > len(ids) {
		workerCount = len(ids)
	}
	if workerCount <= 0 {
		workerCount = 1
	}

	worker := func() {
		defer wg.Done()
		for {
			if err := ctx.Err(); err != nil {
				return
			}

			mu.Lock()
			if index >= len(ids) {
				mu.Unlock()
				return
			}
			accountID := ids[index]
			index++
			mu.Unlock()

			result, err := s.accountTestSvc.RunTestBackground(ctx, accountID, accountBulkTestActivateDefaultModel)
			success := err == nil && result != nil && strings.EqualFold(strings.TrimSpace(result.Status), "success")

			mu.Lock()
			processed++
			if success {
				summary.SuccessIDs = append(summary.SuccessIDs, accountID)
				testedSuccess = append(testedSuccess, accountID)
			} else {
				summary.FailedIDs = append(summary.FailedIDs, accountID)
				testedFailed = append(testedFailed, accountID)
			}
			mu.Unlock()
		}
	}

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go worker()
	}
	wg.Wait()

	summary.Processed = processed
	summary.Success = len(summary.SuccessIDs)
	summary.Failed = len(summary.FailedIDs)

	for _, accountID := range testedSuccess {
		if !strings.EqualFold(originalStatus[accountID], StatusActive) {
			summary.ActivatedIDs = append(summary.ActivatedIDs, accountID)
		}
	}
	for _, accountID := range testedFailed {
		if strings.EqualFold(originalStatus[accountID], StatusActive) {
			summary.DeactivatedIDs = append(summary.DeactivatedIDs, accountID)
		}
	}

	if ctxErr := ctx.Err(); ctxErr != nil {
		summary.TimedOut = errors.Is(ctxErr, context.DeadlineExceeded)
		accountBulkTestActivateAppendError(summary, ctxErr.Error())
	}

	finalizeCtx, cancel := accountBulkTestActivateDetachedContext(ctx, s.finalizeTimeout())
	defer cancel()

	if err := s.applyStatusUpdates(finalizeCtx, summary); err != nil {
		accountBulkTestActivateAppendError(summary, fmt.Sprintf("状态写回失败: %v", err))
	}

	summary.Activated = len(summary.ActivatedIDs)
	summary.Deactivated = len(summary.DeactivatedIDs)
	summary.Remaining = accountBulkTestActivateRemaining(summary)
	s.accountBulkTestActivateSortSummary(summary)
	s.logSummary("completed", summary)
	s.notifySummary(ctx, summary)
	return summary, nil
}

func (s *AccountBulkTestActivateService) runScheduled() {
	if !s.startScheduledRun() {
		summary := &AccountBulkTestActivateSummary{
			Trigger:      string(AccountBulkTestActivateTriggerSchedule),
			ModelID:      accountBulkTestActivateDefaultModel,
			RunSkipped:   true,
			ErrorMessage: accountBulkTestActivateOverlapSkipReason,
		}
		s.logSummary("skipped", summary)
		s.notifySummary(context.Background(), summary)
		return
	}
	defer s.finishScheduledRun()

	_, err := s.Execute(context.Background(), nil, AccountBulkTestActivateTriggerSchedule)
	if err != nil {
		failedSummary := &AccountBulkTestActivateSummary{
			Trigger:      string(AccountBulkTestActivateTriggerSchedule),
			ModelID:      accountBulkTestActivateDefaultModel,
			ErrorMessage: err.Error(),
			TimedOut:     errors.Is(err, context.DeadlineExceeded),
		}
		s.logSummary("failed", failedSummary)
		s.notifySummary(context.Background(), failedSummary)
		return
	}
}

func (s *AccountBulkTestActivateService) resolveAccounts(ctx context.Context, accountIDs []int64) (*accountBulkTestActivateResolution, error) {
	if len(accountIDs) > 0 {
		requestedIDs := dedupeInt64s(accountIDs)
		accounts, err := s.accountRepo.GetByIDs(ctx, requestedIDs)
		if err != nil {
			return nil, err
		}
		ordered, skippedIDs, missingIDs, err := s.partitionAccounts(ctx, requestedIDs, accounts)
		if err != nil {
			return nil, err
		}
		return &accountBulkTestActivateResolution{
			requestedIDs: requestedIDs,
			accounts:     ordered,
			skippedIDs:   skippedIDs,
			missingIDs:   missingIDs,
		}, nil
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

	requestedIDs := make([]int64, 0, len(out))
	for _, account := range out {
		if account == nil {
			continue
		}
		requestedIDs = append(requestedIDs, account.ID)
	}

	filtered, skippedIDs, missingIDs, err := s.partitionAccounts(ctx, requestedIDs, out)
	if err != nil {
		return nil, err
	}
	return &accountBulkTestActivateResolution{
		requestedIDs: requestedIDs,
		accounts:     filtered,
		skippedIDs:   skippedIDs,
		missingIDs:   missingIDs,
	}, nil
}

func (s *AccountBulkTestActivateService) notifySummary(ctx context.Context, summary *AccountBulkTestActivateSummary) {
	if s == nil || s.telegram == nil || summary == nil {
		return
	}
	notifyCtx, cancel := accountBulkTestActivateDetachedContext(ctx, s.notifyTimeout())
	defer cancel()

	message := s.formatSummaryMessage(summary)
	if err := s.telegram.SendText(notifyCtx, message); err != nil {
		logger.LegacyPrintf(
			"service.account_bulk_test_activate",
			"[AccountBulkTestActivate] telegram notify failed (category=%s): %v",
			accountBulkTestActivateNotifyErrorCategory(err),
			err,
		)
	}
}

func (s *AccountBulkTestActivateService) executionConfig(trigger AccountBulkTestActivateTrigger) accountBulkTestActivateExecutionConfig {
	concurrency := accountBulkTestActivateConcurrency
	if s != nil && s.concurrencyOverride > 0 {
		concurrency = s.concurrencyOverride
	}

	switch trigger {
	case AccountBulkTestActivateTriggerSchedule:
		timeout := accountBulkTestActivateScheduledTimeout
		if s != nil && s.scheduledTimeoutOverride > 0 {
			timeout = s.scheduledTimeoutOverride
		}
		return accountBulkTestActivateExecutionConfig{
			concurrency: concurrency,
			timeout:     timeout,
		}
	default:
		timeout := accountBulkTestActivateManualTimeout
		if s != nil && s.manualTimeoutOverride > 0 {
			timeout = s.manualTimeoutOverride
		}
		return accountBulkTestActivateExecutionConfig{
			concurrency:  concurrency,
			timeout:      timeout,
			detachCaller: true,
		}
	}
}

func (s *AccountBulkTestActivateService) executionContext(ctx context.Context, cfg accountBulkTestActivateExecutionConfig) (context.Context, context.CancelFunc) {
	if cfg.detachCaller {
		return accountBulkTestActivateDetachedContext(ctx, cfg.timeout)
	}
	if ctx == nil {
		return context.WithTimeout(context.Background(), cfg.timeout)
	}
	return context.WithTimeout(ctx, cfg.timeout)
}

func (s *AccountBulkTestActivateService) finalizeTimeout() time.Duration {
	if s != nil && s.finalizeTimeoutOverride > 0 {
		return s.finalizeTimeoutOverride
	}
	return accountBulkTestActivateFinalizeTimeout
}

func (s *AccountBulkTestActivateService) notifyTimeout() time.Duration {
	if s != nil && s.notifyTimeoutOverride > 0 {
		return s.notifyTimeoutOverride
	}
	return accountBulkTestActivateNotifyTimeout
}

func (s *AccountBulkTestActivateService) partitionAccounts(ctx context.Context, requestedIDs []int64, accounts []*Account) ([]*Account, []int64, []int64, error) {
	liveByID := make(map[int64]*Account, len(accounts))
	for _, account := range accounts {
		if account == nil || account.ID <= 0 {
			continue
		}
		liveByID[account.ID] = account
	}

	softDeleted := make(map[int64]struct{})
	if inspector, ok := s.accountRepo.(accountBulkTestActivateSoftDeleteInspector); ok && len(requestedIDs) > 0 {
		softDeletedIDs, err := inspector.ListSoftDeletedIDs(ctx, requestedIDs)
		if err != nil {
			return nil, nil, nil, err
		}
		for _, id := range softDeletedIDs {
			softDeleted[id] = struct{}{}
		}
	}

	ordered := make([]*Account, 0, len(requestedIDs))
	skippedIDs := make([]int64, 0)
	missingIDs := make([]int64, 0)
	for _, id := range requestedIDs {
		if _, ok := softDeleted[id]; ok {
			skippedIDs = append(skippedIDs, id)
			continue
		}
		if account, ok := liveByID[id]; ok && account != nil {
			ordered = append(ordered, account)
			continue
		}
		missingIDs = append(missingIDs, id)
	}

	return ordered, skippedIDs, missingIDs, nil
}

func (s *AccountBulkTestActivateService) applyStatusUpdates(ctx context.Context, summary *AccountBulkTestActivateSummary) error {
	if s == nil || s.accountRepo == nil || summary == nil {
		return nil
	}

	if len(summary.ActivatedIDs) > 0 {
		status := StatusActive
		if _, err := s.accountRepo.BulkUpdate(ctx, summary.ActivatedIDs, AccountBulkUpdate{Status: &status}); err != nil {
			return err
		}
	}
	if len(summary.DeactivatedIDs) > 0 {
		status := "inactive"
		if _, err := s.accountRepo.BulkUpdate(ctx, summary.DeactivatedIDs, AccountBulkUpdate{Status: &status}); err != nil {
			return err
		}
	}
	return nil
}

func (s *AccountBulkTestActivateService) accountBulkTestActivateSortSummary(summary *AccountBulkTestActivateSummary) {
	if summary == nil {
		return
	}
	sort.Slice(summary.SuccessIDs, func(i, j int) bool { return summary.SuccessIDs[i] < summary.SuccessIDs[j] })
	sort.Slice(summary.FailedIDs, func(i, j int) bool { return summary.FailedIDs[i] < summary.FailedIDs[j] })
	sort.Slice(summary.ActivatedIDs, func(i, j int) bool { return summary.ActivatedIDs[i] < summary.ActivatedIDs[j] })
	sort.Slice(summary.DeactivatedIDs, func(i, j int) bool { return summary.DeactivatedIDs[i] < summary.DeactivatedIDs[j] })
}

func (s *AccountBulkTestActivateService) formatSummaryMessage(summary *AccountBulkTestActivateSummary) string {
	if summary == nil {
		return ""
	}

	title := "批量测试激活完成"
	switch {
	case summary.RunSkipped:
		title = "批量测试激活已跳过"
	case summary.TimedOut:
		title = "批量测试激活部分完成（超时）"
	case summary.ErrorMessage != "":
		title = "批量测试激活完成（存在异常）"
	}

	message := fmt.Sprintf(
		"%s\n触发方式：%s\n请求总数：%d\n测试总数：%d\n已处理：%d\n剩余：%d\n成功：%d\n失败：%d\n跳过：%d\n新增启用：%d\n新增禁用：%d",
		title,
		summary.Trigger,
		summary.RequestedTotal,
		summary.Total,
		summary.Processed,
		summary.Remaining,
		summary.Success,
		summary.Failed,
		summary.Skipped,
		summary.Activated,
		summary.Deactivated,
	)
	if summary.DurationMs > 0 {
		message += fmt.Sprintf("\n耗时：%dms", summary.DurationMs)
	}
	if strings.TrimSpace(summary.ErrorMessage) != "" {
		message += "\n异常：" + strings.TrimSpace(summary.ErrorMessage)
	}
	return message
}

func (s *AccountBulkTestActivateService) logSummary(stage string, summary *AccountBulkTestActivateSummary) {
	if summary == nil {
		return
	}
	logger.LegacyPrintf(
		"service.account_bulk_test_activate",
		"[AccountBulkTestActivate] %s trigger=%s requested_total=%d total=%d processed=%d remaining=%d success=%d failed=%d skipped=%d activated=%d deactivated=%d timed_out=%t run_skipped=%t duration_ms=%d err=%q",
		stage,
		summary.Trigger,
		summary.RequestedTotal,
		summary.Total,
		summary.Processed,
		summary.Remaining,
		summary.Success,
		summary.Failed,
		summary.Skipped,
		summary.Activated,
		summary.Deactivated,
		summary.TimedOut,
		summary.RunSkipped,
		summary.DurationMs,
		summary.ErrorMessage,
	)
}

func (s *AccountBulkTestActivateService) startScheduledRun() bool {
	if s == nil {
		return false
	}
	s.scheduledRunMu.Lock()
	defer s.scheduledRunMu.Unlock()
	if s.scheduledRunRunning {
		return false
	}
	s.scheduledRunRunning = true
	return true
}

func (s *AccountBulkTestActivateService) finishScheduledRun() {
	if s == nil {
		return
	}
	s.scheduledRunMu.Lock()
	s.scheduledRunRunning = false
	s.scheduledRunMu.Unlock()
}

func accountBulkTestActivateDetachedContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		return context.WithTimeout(context.Background(), timeout)
	}
	return context.WithTimeout(context.WithoutCancel(ctx), timeout)
}

func accountBulkTestActivateAppendError(summary *AccountBulkTestActivateSummary, message string) {
	if summary == nil || strings.TrimSpace(message) == "" {
		return
	}
	if strings.TrimSpace(summary.ErrorMessage) == "" {
		summary.ErrorMessage = strings.TrimSpace(message)
		return
	}
	summary.ErrorMessage = summary.ErrorMessage + "; " + strings.TrimSpace(message)
}

func accountBulkTestActivateRemaining(summary *AccountBulkTestActivateSummary) int {
	if summary == nil {
		return 0
	}
	remaining := summary.Total - summary.Success - summary.Failed - summary.Skipped
	if remaining < 0 {
		return 0
	}
	return remaining
}

func accountBulkTestActivateNotifyErrorCategory(err error) string {
	switch {
	case err == nil:
		return "none"
	case errors.Is(err, context.Canceled):
		return "context_canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	case strings.Contains(err.Error(), "load telegram config"):
		return "settings_load_failed"
	case strings.Contains(err.Error(), "telegram api status"):
		return "telegram_api_status"
	case strings.Contains(err.Error(), "telegram api rejected"):
		return "telegram_api_rejected"
	default:
		return "send_failed"
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
