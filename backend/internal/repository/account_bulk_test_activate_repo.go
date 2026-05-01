package repository

import (
	"context"
	"database/sql"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type accountBulkTestActivateAuditRepository struct {
	db *sql.DB
}

func NewAccountBulkTestActivateAuditRepository(db *sql.DB) service.AccountBulkTestActivateAuditRepository {
	return &accountBulkTestActivateAuditRepository{db: db}
}

func (r *accountBulkTestActivateAuditRepository) CreateRun(ctx context.Context, run *service.AccountBulkTestActivateRun) (*service.AccountBulkTestActivateRun, error) {
	if run == nil {
		return nil, service.ErrAccountNilInput
	}
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO account_bulk_test_activate_runs (
			trigger, model_id, requested_total, total, processed, remaining,
			success, failed, skipped, activated, deactivated, timed_out,
			run_skipped, duration_ms, error_message, started_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, NOW(), NOW())
		RETURNING id, trigger, model_id, requested_total, total, processed, remaining,
			success, failed, skipped, activated, deactivated, timed_out, run_skipped,
			duration_ms, error_message, started_at, finished_at, created_at, updated_at
	`,
		run.Trigger, run.ModelID, run.RequestedTotal, run.Total, run.Processed, run.Remaining,
		run.Success, run.Failed, run.Skipped, run.Activated, run.Deactivated, run.TimedOut,
		run.RunSkipped, run.DurationMs, run.ErrorMessage, run.StartedAt,
	)
	return scanAccountBulkTestActivateRun(row)
}

func (r *accountBulkTestActivateAuditRepository) SaveResults(ctx context.Context, results []*service.AccountBulkTestActivateResult) error {
	if len(results) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO account_bulk_test_activate_results (
			run_id, account_id, original_status, status, action, attempts,
			failure_category, error_message, latency_ms, started_at, finished_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
	`)
	if err != nil {
		return err
	}
	defer func() { _ = stmt.Close() }()

	for _, result := range results {
		if result == nil {
			continue
		}
		if _, err = stmt.ExecContext(ctx,
			result.RunID,
			result.AccountID,
			result.OriginalStatus,
			result.Status,
			result.Action,
			result.Attempts,
			result.FailureCategory,
			result.ErrorMessage,
			result.LatencyMs,
			result.StartedAt,
			result.FinishedAt,
		); err != nil {
			return err
		}
	}
	err = tx.Commit()
	return err
}

func (r *accountBulkTestActivateAuditRepository) FinishRun(ctx context.Context, run *service.AccountBulkTestActivateRun) error {
	if run == nil || run.ID <= 0 {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE account_bulk_test_activate_runs
		SET processed = $2,
			remaining = $3,
			success = $4,
			failed = $5,
			skipped = $6,
			activated = $7,
			deactivated = $8,
			timed_out = $9,
			run_skipped = $10,
			duration_ms = $11,
			error_message = $12,
			finished_at = $13,
			updated_at = NOW()
		WHERE id = $1
	`,
		run.ID,
		run.Processed,
		run.Remaining,
		run.Success,
		run.Failed,
		run.Skipped,
		run.Activated,
		run.Deactivated,
		run.TimedOut,
		run.RunSkipped,
		run.DurationMs,
		run.ErrorMessage,
		run.FinishedAt,
	)
	return err
}

func scanAccountBulkTestActivateRun(row scannable) (*service.AccountBulkTestActivateRun, error) {
	out := &service.AccountBulkTestActivateRun{}
	if err := row.Scan(
		&out.ID,
		&out.Trigger,
		&out.ModelID,
		&out.RequestedTotal,
		&out.Total,
		&out.Processed,
		&out.Remaining,
		&out.Success,
		&out.Failed,
		&out.Skipped,
		&out.Activated,
		&out.Deactivated,
		&out.TimedOut,
		&out.RunSkipped,
		&out.DurationMs,
		&out.ErrorMessage,
		&out.StartedAt,
		&out.FinishedAt,
		&out.CreatedAt,
		&out.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return out, nil
}
