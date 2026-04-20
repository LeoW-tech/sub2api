package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const (
	accountInitialProbeWorkerName        = "account_initial_probe_worker"
	accountInitialProbeInterval          = 10 * time.Second
	accountInitialProbeTaskTimeout       = 2 * time.Minute
	accountInitialProbeStaleRunningAfter = 10 * time.Minute
	accountInitialProbeMaxWorkers        = 4
	accountInitialProbeMaxAttempts       = 2
)

var accountInitialProbeStatusCodePattern = regexp.MustCompile(`API returned (\d{3})`)

type accountInitialProbeRunner interface {
	RunTestBackground(ctx context.Context, accountID int64, modelID string) (*ScheduledTestResult, error)
}

// AccountInitialProbeService enqueues and executes one-time probes for newly created accounts.
type AccountInitialProbeService struct {
	repo        AccountInitialProbeTaskRepository
	accountRepo AccountRepository
	runner      accountInitialProbeRunner
	timingWheel *TimingWheelService

	startOnce sync.Once
	stopOnce  sync.Once
	running   int32

	workerCtx    context.Context
	workerCancel context.CancelFunc
}

func NewAccountInitialProbeService(
	repo AccountInitialProbeTaskRepository,
	accountRepo AccountRepository,
	runner accountInitialProbeRunner,
	timingWheel *TimingWheelService,
) *AccountInitialProbeService {
	workerCtx, workerCancel := context.WithCancel(context.Background())
	return &AccountInitialProbeService{
		repo:         repo,
		accountRepo:  accountRepo,
		runner:       runner,
		timingWheel:  timingWheel,
		workerCtx:    workerCtx,
		workerCancel: workerCancel,
	}
}

func (s *AccountInitialProbeService) Start() {
	if s == nil || s.repo == nil || s.accountRepo == nil || s.runner == nil || s.timingWheel == nil {
		logger.LegacyPrintf("service.account_initial_probe", "[AccountInitialProbe] not started (missing deps)")
		return
	}
	s.startOnce.Do(func() {
		s.timingWheel.ScheduleRecurring(accountInitialProbeWorkerName, accountInitialProbeInterval, s.runOnce)
		logger.LegacyPrintf("service.account_initial_probe", "[AccountInitialProbe] started (interval=%s max_workers=%d)", accountInitialProbeInterval, accountInitialProbeMaxWorkers)
	})
}

func (s *AccountInitialProbeService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		if s.workerCancel != nil {
			s.workerCancel()
		}
		if s.timingWheel != nil {
			s.timingWheel.Cancel(accountInitialProbeWorkerName)
		}
		logger.LegacyPrintf("service.account_initial_probe", "[AccountInitialProbe] stopped")
	})
}

func (s *AccountInitialProbeService) EnqueueAccountInitialProbe(ctx context.Context, accountID int64, platform, triggerSource string) error {
	if s == nil || s.repo == nil {
		return nil
	}
	task := &AccountInitialProbeTask{
		AccountID:     accountID,
		Status:        AccountInitialProbeStatusPending,
		ModelID:       initialProbeModelForPlatform(platform),
		TriggerSource: strings.TrimSpace(triggerSource),
	}
	created, err := s.repo.CreateTaskIfAbsent(ctx, task)
	if err != nil {
		return fmt.Errorf("create initial probe task: %w", err)
	}
	if created {
		logger.LegacyPrintf("service.account_initial_probe", "[AccountInitialProbe] enqueued: account=%d platform=%s model=%s source=%s", accountID, platform, task.ModelID, task.TriggerSource)
		go s.runOnce()
	}
	return nil
}

func initialProbeModelForPlatform(platform string) string {
	if platform == PlatformOpenAI {
		return "gpt-5.4"
	}
	return ""
}

func (s *AccountInitialProbeService) runOnce() {
	if s == nil || s.repo == nil || s.accountRepo == nil || s.runner == nil {
		return
	}
	if !atomic.CompareAndSwapInt32(&s.running, 0, 1) {
		return
	}
	defer atomic.StoreInt32(&s.running, 0)

	parent := context.Background()
	if s.workerCtx != nil {
		parent = s.workerCtx
	}
	ctx, cancel := context.WithTimeout(parent, accountInitialProbeTaskTimeout)
	defer cancel()

	var tasks []*AccountInitialProbeTask
	for len(tasks) < accountInitialProbeMaxWorkers {
		task, err := s.repo.ClaimNextPendingTask(ctx, int64(accountInitialProbeStaleRunningAfter.Seconds()))
		if err != nil {
			logger.LegacyPrintf("service.account_initial_probe", "[AccountInitialProbe] claim pending task failed: %v", err)
			return
		}
		if task == nil {
			break
		}
		tasks = append(tasks, task)
	}
	if len(tasks) == 0 {
		return
	}

	var wg sync.WaitGroup
	for _, task := range tasks {
		wg.Add(1)
		go func(task *AccountInitialProbeTask) {
			defer wg.Done()
			s.runTask(ctx, task)
		}(task)
	}
	wg.Wait()
}

func (s *AccountInitialProbeService) runTask(ctx context.Context, task *AccountInitialProbeTask) {
	account, err := s.accountRepo.GetByID(ctx, task.AccountID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(strings.ToLower(err.Error()), "not found") {
			_ = s.repo.MarkTaskSkipped(ctx, task.ID, 0, "account not found")
			return
		}
		_ = s.repo.MarkTaskFailed(ctx, task.ID, 0, err.Error())
		return
	}
	if account == nil {
		_ = s.repo.MarkTaskSkipped(ctx, task.ID, 0, "account not found")
		return
	}
	if account.Status == StatusDisabled {
		_ = s.repo.MarkTaskSkipped(ctx, task.ID, 0, "account already disabled")
		return
	}
	if account.Status == StatusError {
		_ = s.repo.MarkTaskSkipped(ctx, task.ID, 0, "account already errored")
		return
	}

	attempts := 0
	var lastError string
	for attempts < accountInitialProbeMaxAttempts {
		attempts++
		result, runErr := s.runner.RunTestBackground(ctx, task.AccountID, task.ModelID)
		if runErr == nil && result != nil && strings.EqualFold(result.Status, "success") {
			_ = s.repo.MarkTaskSucceeded(ctx, task.ID, attempts)
			return
		}
		lastError = resultErrorMessage(result, runErr)
		if !isRetryableInitialProbeError(lastError) || attempts >= accountInitialProbeMaxAttempts {
			break
		}
	}

	latest, latestErr := s.accountRepo.GetByID(ctx, task.AccountID)
	if latestErr == nil && latest != nil && (latest.Status == StatusDisabled || latest.Status == StatusError) {
		_ = s.repo.MarkTaskSkipped(ctx, task.ID, attempts, "account status changed during probe")
		return
	}

	status := StatusDisabled
	if _, err := s.accountRepo.BulkUpdate(ctx, []int64{task.AccountID}, AccountBulkUpdate{
		Status: &status,
	}); err != nil {
		lastError = strings.TrimSpace(lastError + "; disable failed: " + err.Error())
	}
	_ = s.repo.MarkTaskFailed(ctx, task.ID, attempts, lastError)
}

func resultErrorMessage(result *ScheduledTestResult, runErr error) string {
	if runErr != nil {
		return strings.TrimSpace(runErr.Error())
	}
	if result == nil {
		return "initial probe failed"
	}
	if msg := strings.TrimSpace(result.ErrorMessage); msg != "" {
		return msg
	}
	if !strings.EqualFold(result.Status, "success") {
		return "initial probe failed"
	}
	return ""
}

func isRetryableInitialProbeError(msg string) bool {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return false
	}
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "timeout") ||
		strings.Contains(lower, "temporarily unavailable") ||
		strings.Contains(lower, "connection reset") ||
		strings.Contains(lower, "connection refused") ||
		strings.Contains(lower, "eof") ||
		strings.Contains(lower, "request failed") ||
		strings.Contains(lower, "stream read error") {
		return true
	}
	matches := accountInitialProbeStatusCodePattern.FindStringSubmatch(msg)
	if len(matches) != 2 {
		return false
	}
	code, err := strconv.Atoi(matches[1])
	if err != nil {
		return false
	}
	return code >= http.StatusInternalServerError
}
