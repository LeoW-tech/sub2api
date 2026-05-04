-- Build affiliate ledger audit lookup indexes online so writes can continue during startup migrations.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_affiliate_ledger_source_order_id
    ON user_affiliate_ledger(source_order_id)
    WHERE source_order_id IS NOT NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_affiliate_ledger_rebate_lookup
    ON user_affiliate_ledger(action, source_order_id, user_id, source_user_id, created_at)
    WHERE action = 'accrue';
