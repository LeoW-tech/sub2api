ALTER TABLE proxies
    ADD COLUMN IF NOT EXISTS network_status VARCHAR(20);

ALTER TABLE proxies
    ADD COLUMN IF NOT EXISTS network_checked_at TIMESTAMPTZ;

ALTER TABLE proxies
    ADD COLUMN IF NOT EXISTS network_error_message TEXT;

ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS network_auto_paused BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_proxies_network_status ON proxies(network_status);
CREATE INDEX IF NOT EXISTS idx_accounts_network_auto_paused ON accounts(network_auto_paused) WHERE deleted_at IS NULL;

COMMENT ON COLUMN proxies.network_status IS '代理网络状态：online/offline，NULL 表示尚未检测';
COMMENT ON COLUMN proxies.network_checked_at IS '代理网络状态最近检测时间';
COMMENT ON COLUMN proxies.network_error_message IS '最近一次代理网络检测失败原因';
COMMENT ON COLUMN accounts.network_auto_paused IS '是否因代理网络异常被系统自动暂停调度';
