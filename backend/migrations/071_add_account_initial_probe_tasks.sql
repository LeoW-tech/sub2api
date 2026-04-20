-- 071: Add one-time initial probe tasks for newly created accounts

CREATE TABLE IF NOT EXISTS account_initial_probe_tasks (
    id             BIGSERIAL PRIMARY KEY,
    account_id     BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    status         VARCHAR(20) NOT NULL,
    model_id       TEXT NOT NULL DEFAULT '',
    trigger_source VARCHAR(64) NOT NULL DEFAULT '',
    attempt_count  INTEGER NOT NULL DEFAULT 0,
    last_error     TEXT NOT NULL DEFAULT '',
    started_at     TIMESTAMPTZ NULL,
    finished_at    TIMESTAMPTZ NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_account_initial_probe_tasks_account UNIQUE (account_id)
);

CREATE INDEX IF NOT EXISTS idx_account_initial_probe_tasks_status_created
    ON account_initial_probe_tasks(status, created_at, id);
