-- 091_add_proxy_external_key_and_exit_ip.sql
-- 为代理增加外部门标识和可选出口 IP 元数据，支持“门控服务”同步

ALTER TABLE proxies
    ADD COLUMN IF NOT EXISTS external_key VARCHAR(191),
    ADD COLUMN IF NOT EXISTS exit_ip VARCHAR(64),
    ADD COLUMN IF NOT EXISTS exit_ip_checked_at TIMESTAMPTZ;

CREATE UNIQUE INDEX IF NOT EXISTS idx_proxies_external_key
    ON proxies(external_key)
    WHERE external_key IS NOT NULL AND deleted_at IS NULL;
