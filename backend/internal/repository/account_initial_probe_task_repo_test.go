package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAccountInitialProbeTaskRepositoryCreateTaskIfAbsentCreatesTask(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &accountInitialProbeTaskRepository{db: db}

	now := time.Date(2026, 4, 21, 10, 0, 0, 0, time.UTC)
	task := &service.AccountInitialProbeTask{
		AccountID:     88,
		Status:        service.AccountInitialProbeStatusPending,
		ModelID:       "gpt-5.4",
		TriggerSource: "admin_create",
	}

	mock.ExpectQuery("INSERT INTO account_initial_probe_tasks").
		WithArgs(task.AccountID, task.Status, task.ModelID, task.TriggerSource).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "account_id", "status", "model_id", "trigger_source", "attempt_count", "last_error", "started_at", "finished_at", "created_at", "updated_at",
		}).AddRow(int64(1), task.AccountID, task.Status, task.ModelID, task.TriggerSource, 0, nil, nil, nil, now, now))

	created, err := repo.CreateTaskIfAbsent(context.Background(), task)
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, int64(1), task.ID)
	require.Equal(t, now, task.CreatedAt)
	require.Equal(t, now, task.UpdatedAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAccountInitialProbeTaskRepositoryClaimNextPendingTaskNone(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &accountInitialProbeTaskRepository{db: db}

	mock.ExpectQuery("UPDATE account_initial_probe_tasks").
		WithArgs(service.AccountInitialProbeStatusPending, service.AccountInitialProbeStatusRunning, int64(600), service.AccountInitialProbeStatusRunning).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "account_id", "status", "model_id", "trigger_source", "attempt_count", "last_error", "started_at", "finished_at", "created_at", "updated_at",
		}))

	task, err := repo.ClaimNextPendingTask(context.Background(), 600)
	require.NoError(t, err)
	require.Nil(t, task)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAccountInitialProbeTaskRepositoryMarkTaskFailed(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &accountInitialProbeTaskRepository{db: db}

	mock.ExpectExec("UPDATE account_initial_probe_tasks").
		WithArgs(service.AccountInitialProbeStatusFailed, 2, "API returned 503", int64(5)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.MarkTaskFailed(context.Background(), 5, 2, "API returned 503")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAccountInitialProbeTaskRepositoryCreateTaskIfAbsentLoadsExistingOnConflict(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &accountInitialProbeTaskRepository{db: db}

	now := time.Date(2026, 4, 21, 11, 0, 0, 0, time.UTC)
	task := &service.AccountInitialProbeTask{
		AccountID:     99,
		Status:        service.AccountInitialProbeStatusPending,
		ModelID:       "gpt-5.4",
		TriggerSource: "admin_create",
	}

	mock.ExpectQuery("INSERT INTO account_initial_probe_tasks").
		WithArgs(task.AccountID, task.Status, task.ModelID, task.TriggerSource).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("SELECT id, account_id, status, model_id, trigger_source, attempt_count, last_error, started_at, finished_at, created_at, updated_at FROM account_initial_probe_tasks WHERE account_id = \\$1").
		WithArgs(task.AccountID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "account_id", "status", "model_id", "trigger_source", "attempt_count", "last_error", "started_at", "finished_at", "created_at", "updated_at",
		}).AddRow(int64(3), task.AccountID, task.Status, task.ModelID, task.TriggerSource, 0, nil, nil, nil, now, now))

	created, err := repo.CreateTaskIfAbsent(context.Background(), task)
	require.NoError(t, err)
	require.False(t, created)
	require.Equal(t, int64(3), task.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}
