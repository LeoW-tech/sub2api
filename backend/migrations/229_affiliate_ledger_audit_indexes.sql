-- 补齐上游 v0.1.179 在同一返利审计迁移中新增的索引。
-- 本地 main 已使用 135_affiliate_ledger_audit_snapshots.sql 记录同一能力，
-- 因此不能修改已发布迁移；以独立迁移安全补齐缺失索引。
CREATE INDEX IF NOT EXISTS idx_user_affiliate_ledger_source_order_id
    ON user_affiliate_ledger(source_order_id)
    WHERE source_order_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_user_affiliate_ledger_rebate_lookup
    ON user_affiliate_ledger(action, source_order_id, user_id, source_user_id, created_at)
    WHERE action = 'accrue';
