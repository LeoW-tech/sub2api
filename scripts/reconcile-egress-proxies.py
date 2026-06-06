#!/usr/bin/env python3
"""Report sub2api proxies that are no longer present in an egress export.

Default mode is dry-run: write reports only. Use --apply-inactive to mark safe
stale candidates inactive; this script never deletes proxies.
"""

from __future__ import annotations

import argparse
import csv
import json
import os
import sys
from pathlib import Path
from typing import Any, Iterable

REPORT_FIELDS = [
    "id",
    "name",
    "external_key",
    "status",
    "deleted_at",
    "account_count",
    "protocol",
    "host",
    "port",
    "reason",
]

PROXY_QUERY = """
SELECT
  p.id,
  p.name,
  p.external_key,
  p.status,
  p.deleted_at,
  p.protocol,
  p.host,
  p.port,
  COALESCE(a.account_count, 0) AS account_count
FROM proxies p
LEFT JOIN (
  SELECT proxy_id, COUNT(*) AS account_count
  FROM accounts
  WHERE proxy_id IS NOT NULL AND deleted_at IS NULL
  GROUP BY proxy_id
) a ON a.proxy_id = p.id
ORDER BY p.id ASC
"""

APPLY_INACTIVE_SQL = """
UPDATE proxies p
SET status = 'inactive', updated_at = NOW()
WHERE p.id = %s
  AND BTRIM(p.external_key) = %s
  AND p.deleted_at IS NULL
  AND NULLIF(BTRIM(p.external_key), '') IS NOT NULL
  AND NOT EXISTS (
    SELECT 1
    FROM accounts a
    WHERE a.proxy_id = p.id AND a.deleted_at IS NULL
  )
RETURNING p.id
"""


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Generate stale sub2api proxy candidates from an egress export JSON.",
    )
    parser.add_argument(
        "--export-json",
        default=os.environ.get("EGRESS_EXPORT_JSON"),
        help="Path to egress export JSON. Env fallback: EGRESS_EXPORT_JSON.",
    )
    parser.add_argument(
        "--postgres-url",
        default=postgres_url_from_env(),
        help=(
            "Postgres connection URL/DSN. Env fallback: "
            "SUB2API_POSTGRES_URL, DATABASE_URL, POSTGRES_DSN, or standard PG* vars."
        ),
    )
    parser.add_argument(
        "--report-dir",
        default="reports/egress-proxy-reconcile",
        help="Directory for stale-proxies.json and stale-proxies.csv.",
    )
    parser.add_argument(
        "--apply-inactive",
        action="store_true",
        help="Mark safe stale candidates inactive. Omit for dry-run report only.",
    )
    return parser.parse_args(argv)


def postgres_url_from_env() -> str | None:
    for name in ("SUB2API_POSTGRES_URL", "DATABASE_URL", "POSTGRES_DSN"):
        value = os.environ.get(name)
        if value:
            return value
    if has_standard_pg_env():
        return ""
    return None


def has_standard_pg_env() -> bool:
    return any(os.environ.get(name) for name in ("PGHOST", "PGPORT", "PGDATABASE", "PGUSER", "PGPASSWORD"))


def load_json(path: str | os.PathLike[str]) -> Any:
    with Path(path).open(encoding="utf-8") as fh:
        return json.load(fh)


def extract_proxy_external_keys(payload: Any) -> set[str]:
    keys: set[str] = set()

    def walk(value: Any) -> None:
        if isinstance(value, dict):
            for key, child in value.items():
                if key in {
                    "proxy_external_key",
                    "proxyExternalKey",
                    "proxy_external_keys",
                    "proxyExternalKeys",
                    "export_external_key",
                    "exportExternalKey",
                    "export_external_keys",
                    "exportExternalKeys",
                }:
                    add_key_value(child)
                else:
                    walk(child)
        elif isinstance(value, list):
            for item in value:
                walk(item)

    def add_key_value(value: Any) -> None:
        if isinstance(value, str):
            normalized = value.strip()
            if normalized:
                keys.add(normalized)
        elif isinstance(value, list):
            for item in value:
                add_key_value(item)
        elif isinstance(value, dict):
            walk(value)

    walk(payload)
    return keys


def connect_postgres(postgres_url: str | None):
    dsn = postgres_url or ""
    try:
        import psycopg  # type: ignore

        return psycopg.connect(dsn)
    except ImportError:
        try:
            import psycopg2  # type: ignore

            return psycopg2.connect(dsn)
        except ImportError as exc:
            raise RuntimeError("需要安装 psycopg 或 psycopg2 才能连接 Postgres") from exc


def fetch_sub2api_proxies(conn) -> list[dict[str, Any]]:
    with conn.cursor() as cur:
        cur.execute(PROXY_QUERY)
        columns = [column_name(desc) for desc in cur.description]
        return [dict(zip(columns, row)) for row in cur.fetchall()]


def column_name(description_item: Any) -> str:
    name = getattr(description_item, "name", None)
    if name is not None:
        return str(name)
    return str(description_item[0])


def find_stale_candidates(
    proxy_rows: Iterable[dict[str, Any]],
    current_egress_keys: set[str],
) -> list[dict[str, Any]]:
    candidates: list[dict[str, Any]] = []
    for row in proxy_rows:
        external_key = str(row.get("external_key") or "").strip()
        if not external_key:
            continue
        if row.get("deleted_at") is not None:
            continue
        if external_key in current_egress_keys:
            continue
        account_count = int(row.get("account_count") or 0)
        if account_count != 0:
            continue

        item = dict(row)
        item["external_key"] = external_key
        item["account_count"] = account_count
        item["reason"] = "missing_from_egress_export"
        candidates.append(item)
    return candidates


def report_row(item: dict[str, Any]) -> dict[str, Any]:
    return {field: normalize_report_value(item.get(field)) for field in REPORT_FIELDS}


def normalize_report_value(value: Any) -> Any:
    if value is None:
        return None
    if hasattr(value, "isoformat"):
        return value.isoformat()
    return value


def write_reports(report_dir: str | os.PathLike[str], candidates: list[dict[str, Any]]) -> dict[str, Path]:
    out_dir = Path(report_dir)
    out_dir.mkdir(parents=True, exist_ok=True)

    json_path = out_dir / "stale-proxies.json"
    csv_path = out_dir / "stale-proxies.csv"
    rows = [report_row(item) for item in candidates]

    json_path.write_text(json.dumps(rows, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    with csv_path.open("w", encoding="utf-8", newline="") as fh:
        writer = csv.DictWriter(fh, fieldnames=REPORT_FIELDS)
        writer.writeheader()
        writer.writerows(rows)

    return {"json": json_path, "csv": csv_path}


def apply_inactive(conn, candidates: list[dict[str, Any]]) -> list[int]:
    updated_ids: list[int] = []
    with conn.cursor() as cur:
        for item in candidates:
            cur.execute(APPLY_INACTIVE_SQL, (item["id"], item["external_key"]))
            updated_ids.extend(int(row[0]) for row in cur.fetchall())
    conn.commit()
    return updated_ids


def run(options: argparse.Namespace) -> int:
    if not options.export_json:
        raise SystemExit("缺少 --export-json 或 EGRESS_EXPORT_JSON")
    if options.postgres_url is None:
        raise SystemExit("缺少 --postgres-url 或 SUB2API_POSTGRES_URL/DATABASE_URL/POSTGRES_DSN/PG* 环境变量")

    export_payload = load_json(options.export_json)
    current_egress_keys = extract_proxy_external_keys(export_payload)

    conn = connect_postgres(options.postgres_url)
    try:
        proxy_rows = fetch_sub2api_proxies(conn)
        candidates = find_stale_candidates(proxy_rows, current_egress_keys)
        paths = write_reports(options.report_dir, candidates)
        updated_ids: list[int] = []
        if options.apply_inactive:
            updated_ids = apply_inactive(conn, candidates)
    finally:
        close = getattr(conn, "close", None)
        if close is not None:
            close()

    summary = {
        "mode": "apply-inactive" if options.apply_inactive else "dry-run",
        "egress_proxy_external_key_count": len(current_egress_keys),
        "sub2api_proxy_count": len(proxy_rows),
        "stale_candidate_count": len(candidates),
        "inactive_updated_count": len(updated_ids),
        "inactive_updated_ids": updated_ids,
        "reports": {name: str(path) for name, path in paths.items()},
    }
    print(json.dumps(summary, ensure_ascii=False, indent=2))
    return 0


def main(argv: list[str] | None = None) -> int:
    return run(parse_args(sys.argv[1:] if argv is None else argv))


if __name__ == "__main__":
    raise SystemExit(main())


if __name__ == "__main__":
    raise SystemExit(main())
