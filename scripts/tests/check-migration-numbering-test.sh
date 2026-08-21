#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd -- "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CHECKER="$REPO_ROOT/scripts/check-migration-numbering"
TMP_ROOT="$(mktemp -d)"
trap 'rm -rf "$TMP_ROOT"' EXIT

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

expect_failure() {
  local expected="$1"
  shift
  local output

  if output="$("$@" 2>&1)"; then
    fail "命令本应失败: $*"
  fi
  [[ "$output" == *"$expected"* ]] || fail "失败信息缺少 '$expected': $output"
}

fixture="$TMP_ROOT/migrations"
mkdir -p "$fixture"
: > "$fixture/225_upstream_a.sql"
: > "$fixture/225_upstream_b.sql"
printf '900001\n' > "$fixture/NEXT_LOCAL_MIGRATION"

"$CHECKER" "$fixture"

: > "$fixture/900001_local_feature.sql"
printf '900002\n' > "$fixture/NEXT_LOCAL_MIGRATION"
"$CHECKER" "$fixture"

: > "$fixture/900001_local_duplicate.sql"
expect_failure "本地迁移编号重复" "$CHECKER" "$fixture"
rm "$fixture/900001_local_duplicate.sql"

printf '900003\n' > "$fixture/NEXT_LOCAL_MIGRATION"
expect_failure "NEXT_LOCAL_MIGRATION 必须等于" "$CHECKER" "$fixture"

printf '900002\n' > "$fixture/NEXT_LOCAL_MIGRATION"
: > "$fixture/900003_local_skipped.sql"
expect_failure "本地迁移编号必须连续" "$CHECKER" "$fixture"
rm "$fixture/900003_local_skipped.sql"

: > "$fixture/230_local_wrong_range.sql"
expect_failure "本地迁移必须使用保留编号区间" "$CHECKER" "$fixture"

echo "migration numbering tests passed"
