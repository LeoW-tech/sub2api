package service

import (
	"context"
	"time"
)

const (
	AccountInitialProbeStatusPending   = "pending"
	AccountInitialProbeStatusRunning   = "running"
	AccountInitialProbeStatusSucceeded = "succeeded"
	AccountInitialProbeStatusFailed    = "failed"
	AccountInitialProbeStatusSkipped   = "skipped"
)

const (
	AccountInitialProbeTriggerAdminCreate = "admin_create"
	AccountInitialProbeTriggerCRSSync     = "crs_sync"
)

// AccountInitialProbeTask represents a one-time initial probe for a newly created account.
type AccountInitialProbeTask struct {
	ID            int64
	AccountID     int64
	Status        string
	ModelID       string
	TriggerSource string
	AttemptCount  int
	LastError     string
	StartedAt     *time.Time
	FinishedAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// AccountInitialProbeTaskRepository persists one-time initial probe tasks.
type AccountInitialProbeTaskRepository interface {
	CreateTaskIfAbsent(ctx context.Context, task *AccountInitialProbeTask) (bool, error)
	ClaimNextPendingTask(ctx context.Context, staleRunningAfterSeconds int64) (*AccountInitialProbeTask, error)
	MarkTaskSucceeded(ctx context.Context, taskID int64, attemptCount int) error
	MarkTaskFailed(ctx context.Context, taskID int64, attemptCount int, lastError string) error
	MarkTaskSkipped(ctx context.Context, taskID int64, attemptCount int, lastError string) error
}

// AccountInitialProbeEnqueuer allows new-account creation flows to enqueue a first-run probe.
type AccountInitialProbeEnqueuer interface {
	EnqueueAccountInitialProbe(ctx context.Context, accountID int64, platform, triggerSource string) error
}
