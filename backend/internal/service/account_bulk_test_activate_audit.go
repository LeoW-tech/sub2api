package service

import (
	"context"
	"time"
)

type AccountBulkTestActivateAuditRepository interface {
	CreateRun(ctx context.Context, run *AccountBulkTestActivateRun) (*AccountBulkTestActivateRun, error)
	SaveResults(ctx context.Context, results []*AccountBulkTestActivateResult) error
	FinishRun(ctx context.Context, run *AccountBulkTestActivateRun) error
}

type AccountBulkTestActivateRun struct {
	ID             int64
	Trigger        string
	ModelID        string
	RequestedTotal int
	Total          int
	Processed      int
	Remaining      int
	Success        int
	Failed         int
	Skipped        int
	Activated      int
	Deactivated    int
	TimedOut       bool
	RunSkipped     bool
	DurationMs     int64
	ErrorMessage   string
	StartedAt      time.Time
	FinishedAt     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type AccountBulkTestActivateResult struct {
	ID              int64     `json:"id,omitempty"`
	RunID           int64     `json:"run_id,omitempty"`
	AccountID       int64     `json:"account_id"`
	OriginalStatus  string    `json:"original_status"`
	Status          string    `json:"status"`
	Action          string    `json:"action"`
	Attempts        int       `json:"attempts"`
	FailureCategory string    `json:"failure_category,omitempty"`
	ErrorMessage    string    `json:"error_message,omitempty"`
	LatencyMs       int64     `json:"latency_ms"`
	StartedAt       time.Time `json:"started_at"`
	FinishedAt      time.Time `json:"finished_at"`
	CreatedAt       time.Time `json:"created_at,omitempty"`
}
