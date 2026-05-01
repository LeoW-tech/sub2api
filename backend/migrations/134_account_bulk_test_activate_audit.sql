CREATE TABLE IF NOT EXISTS account_bulk_test_activate_runs (
    id BIGSERIAL PRIMARY KEY,
    trigger VARCHAR(32) NOT NULL,
    model_id VARCHAR(100) NOT NULL,
    requested_total INTEGER NOT NULL DEFAULT 0,
    total INTEGER NOT NULL DEFAULT 0,
    processed INTEGER NOT NULL DEFAULT 0,
    remaining INTEGER NOT NULL DEFAULT 0,
    success INTEGER NOT NULL DEFAULT 0,
    failed INTEGER NOT NULL DEFAULT 0,
    skipped INTEGER NOT NULL DEFAULT 0,
    activated INTEGER NOT NULL DEFAULT 0,
    deactivated INTEGER NOT NULL DEFAULT 0,
    timed_out BOOLEAN NOT NULL DEFAULT FALSE,
    run_skipped BOOLEAN NOT NULL DEFAULT FALSE,
    duration_ms BIGINT NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS account_bulk_test_activate_results (
    id BIGSERIAL PRIMARY KEY,
    run_id BIGINT NOT NULL REFERENCES account_bulk_test_activate_runs(id) ON DELETE CASCADE,
    account_id BIGINT NOT NULL,
    original_status VARCHAR(32) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL,
    action VARCHAR(32) NOT NULL DEFAULT 'noop',
    attempts INTEGER NOT NULL DEFAULT 1,
    failure_category VARCHAR(64) NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    latency_ms BIGINT NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_account_bulk_test_activate_runs_created_at
    ON account_bulk_test_activate_runs(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_account_bulk_test_activate_results_account_created
    ON account_bulk_test_activate_results(account_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_account_bulk_test_activate_results_run_id
    ON account_bulk_test_activate_results(run_id);

CREATE INDEX IF NOT EXISTS idx_account_bulk_test_activate_results_status
    ON account_bulk_test_activate_results(status, failure_category);
