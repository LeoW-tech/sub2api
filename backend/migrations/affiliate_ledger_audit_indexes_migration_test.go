package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAffiliateLedgerAuditIndexesMigrationIsIdempotent(t *testing.T) {
	content, err := FS.ReadFile("229_affiliate_ledger_audit_indexes.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "CREATE INDEX IF NOT EXISTS idx_user_affiliate_ledger_source_order_id")
	require.Contains(t, sql, "CREATE INDEX IF NOT EXISTS idx_user_affiliate_ledger_rebate_lookup")
	require.Contains(t, sql, "WHERE source_order_id IS NOT NULL")
	require.Contains(t, sql, "WHERE action = 'accrue'")
}
