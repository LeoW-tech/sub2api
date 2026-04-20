package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type accountInitialProbeTaskRepository struct {
	db *sql.DB
}

func NewAccountInitialProbeTaskRepository(db *sql.DB) service.AccountInitialProbeTaskRepository {
	return &accountInitialProbeTaskRepository{db: db}
}

func (r *accountInitialProbeTaskRepository) CreateTaskIfAbsent(ctx context.Context, task *service.AccountInitialProbeTask) (bool, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO account_initial_probe_tasks (account_id, status, model_id, trigger_source, attempt_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 0, NOW(), NOW())
		ON CONFLICT (account_id) DO NOTHING
		RETURNING id, account_id, status, model_id, trigger_source, attempt_count, last_error, started_at, finished_at, created_at, updated_at
	`, task.AccountID, task.Status, task.ModelID, task.TriggerSource)
	if err := scanAccountInitialProbeTask(row, task); err == nil {
		return true, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}

	row = r.db.QueryRowContext(ctx, `
		SELECT id, account_id, status, model_id, trigger_source, attempt_count, last_error, started_at, finished_at, created_at, updated_at
		FROM account_initial_probe_tasks
		WHERE account_id = $1
	`, task.AccountID)
	if err := scanAccountInitialProbeTask(row, task); err != nil {
		return false, err
	}
	return false, nil
}

func (r *accountInitialProbeTaskRepository) ClaimNextPendingTask(ctx context.Context, staleRunningAfterSeconds int64) (*service.AccountInitialProbeTask, error) {
	if staleRunningAfterSeconds <= 0 {
		staleRunningAfterSeconds = 600
	}
	task := &service.AccountInitialProbeTask{}
	err := scanSingleRow(ctx, r.db, `
		WITH next AS (
			SELECT id
			FROM account_initial_probe_tasks
			WHERE status = $1
				OR (
					status = $2
					AND started_at IS NOT NULL
					AND started_at < NOW() - ($3 * interval '1 second')
				)
			ORDER BY created_at ASC, id ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE account_initial_probe_tasks AS tasks
		SET status = $4,
			started_at = NOW(),
			finished_at = NULL,
			updated_at = NOW()
		FROM next
		WHERE tasks.id = next.id
		RETURNING tasks.id, tasks.account_id, tasks.status, tasks.model_id, tasks.trigger_source, tasks.attempt_count, tasks.last_error, tasks.started_at, tasks.finished_at, tasks.created_at, tasks.updated_at
	`, []any{
		service.AccountInitialProbeStatusPending,
		service.AccountInitialProbeStatusRunning,
		staleRunningAfterSeconds,
		service.AccountInitialProbeStatusRunning,
	},
		&task.ID,
		&task.AccountID,
		&task.Status,
		&task.ModelID,
		&task.TriggerSource,
		&task.AttemptCount,
		&task.LastError,
		&task.StartedAt,
		&task.FinishedAt,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (r *accountInitialProbeTaskRepository) MarkTaskSucceeded(ctx context.Context, taskID int64, attemptCount int) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE account_initial_probe_tasks
		SET status = $1,
			attempt_count = $2,
			last_error = '',
			finished_at = NOW(),
			updated_at = NOW()
		WHERE id = $3
	`, service.AccountInitialProbeStatusSucceeded, attemptCount, taskID)
	return err
}

func (r *accountInitialProbeTaskRepository) MarkTaskFailed(ctx context.Context, taskID int64, attemptCount int, lastError string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE account_initial_probe_tasks
		SET status = $1,
			attempt_count = $2,
			last_error = $3,
			finished_at = NOW(),
			updated_at = NOW()
		WHERE id = $4
	`, service.AccountInitialProbeStatusFailed, attemptCount, lastError, taskID)
	return err
}

func (r *accountInitialProbeTaskRepository) MarkTaskSkipped(ctx context.Context, taskID int64, attemptCount int, lastError string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE account_initial_probe_tasks
		SET status = $1,
			attempt_count = $2,
			last_error = $3,
			finished_at = NOW(),
			updated_at = NOW()
		WHERE id = $4
	`, service.AccountInitialProbeStatusSkipped, attemptCount, lastError, taskID)
	return err
}

func scanAccountInitialProbeTask(row scannable, task *service.AccountInitialProbeTask) error {
	var lastError sql.NullString
	if err := row.Scan(
		&task.ID,
		&task.AccountID,
		&task.Status,
		&task.ModelID,
		&task.TriggerSource,
		&task.AttemptCount,
		&lastError,
		&task.StartedAt,
		&task.FinishedAt,
		&task.CreatedAt,
		&task.UpdatedAt,
	); err != nil {
		return err
	}
	if lastError.Valid {
		task.LastError = lastError.String
	} else {
		task.LastError = ""
	}
	return nil
}
