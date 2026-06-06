import csv
import importlib.util
import json
from pathlib import Path


SCRIPT_PATH = Path(__file__).resolve().parents[1] / "scripts" / "reconcile-egress-proxies.py"
spec = importlib.util.spec_from_file_location("reconcile_egress_proxies", SCRIPT_PATH)
reconcile = importlib.util.module_from_spec(spec)
spec.loader.exec_module(reconcile)


def test_extract_proxy_external_keys_from_nested_export():
    payload = {
        "items": [
            {"proxy_external_key": " egress-control:sub2api:a "},
            {"proxyExternalKey": "egress-control:sub2api:b"},
            {"export_external_key": "egress-control:sub2api:export-a"},
            {"exportExternalKey": "egress-control:sub2api:export-b"},
            {"proxy_external_key": ""},
            {"nested": {"proxy_external_key": ["egress-control:sub2api:c", " "]}},
            {"proxy_external_keys": ["egress-control:sub2api:d"]},
            {"export_external_keys": ["egress-control:sub2api:export-c"]},
        ]
    }

    assert reconcile.extract_proxy_external_keys(payload) == {
        "egress-control:sub2api:a",
        "egress-control:sub2api:b",
        "egress-control:sub2api:c",
        "egress-control:sub2api:d",
        "egress-control:sub2api:export-a",
        "egress-control:sub2api:export-b",
        "egress-control:sub2api:export-c",
    }


def test_find_stale_candidates_uses_only_safe_conditions():
    rows = [
        {"id": 1, "name": "safe", "external_key": "missing", "deleted_at": None, "account_count": 0, "status": "active"},
        {"id": 2, "name": "present", "external_key": "present", "deleted_at": None, "account_count": 0, "status": "active"},
        {"id": 3, "name": "has-account", "external_key": "missing-2", "deleted_at": None, "account_count": 1, "status": "active"},
        {"id": 4, "name": "deleted", "external_key": "missing-3", "deleted_at": "2026-01-01T00:00:00Z", "account_count": 0, "status": "active"},
        {"id": 5, "name": "blank-key", "external_key": " ", "deleted_at": None, "account_count": 0, "status": "active"},
    ]

    candidates = reconcile.find_stale_candidates(rows, {"present"})

    assert [item["id"] for item in candidates] == [1]
    assert candidates[0]["reason"] == "missing_from_egress_export"


def test_write_reports_outputs_json_and_csv(tmp_path):
    candidates = [
        {
            "id": 1,
            "name": "safe",
            "external_key": "missing",
            "status": "active",
            "deleted_at": None,
            "account_count": 0,
            "protocol": "http",
            "host": "127.0.0.1",
            "port": 8080,
            "reason": "missing_from_egress_export",
        }
    ]

    paths = reconcile.write_reports(tmp_path, candidates)

    json_rows = json.loads(paths["json"].read_text())
    assert json_rows[0]["external_key"] == "missing"
    assert json_rows == candidates

    with paths["csv"].open(newline="") as fh:
        csv_rows = list(csv.DictReader(fh))
    assert csv_rows[0]["external_key"] == "missing"
    assert csv_rows[0]["account_count"] == "0"


def test_run_defaults_to_dry_run_without_applying(tmp_path, monkeypatch):
    export_path = tmp_path / "egress.json"
    export_path.write_text(json.dumps({"proxies": [{"proxy_external_key": "present"}]}), encoding="utf-8")

    class FakeConnection:
        closed = False

        def close(self):
            self.closed = True

    fake_conn = FakeConnection()

    monkeypatch.setattr(reconcile, "connect_postgres", lambda postgres_url: fake_conn)
    monkeypatch.setattr(
        reconcile,
        "fetch_sub2api_proxies",
        lambda conn: [
            {"id": 1, "name": "stale", "external_key": "missing", "deleted_at": None, "account_count": 0, "status": "active"},
            {"id": 2, "name": "present", "external_key": "present", "deleted_at": None, "account_count": 0, "status": "active"},
        ],
    )

    def fail_apply(conn, candidates):
        raise AssertionError("dry-run must not call apply_inactive")

    monkeypatch.setattr(reconcile, "apply_inactive", fail_apply)

    exit_code = reconcile.main([
        "--export-json",
        str(export_path),
        "--postgres-url",
        "postgres://example/db",
        "--report-dir",
        str(tmp_path / "reports"),
    ])

    assert exit_code == 0
    rows = json.loads((tmp_path / "reports" / "stale-proxies.json").read_text())
    assert [row["external_key"] for row in rows] == ["missing"]
    assert fake_conn.closed is True


class RecordingCursor:
    def __init__(self):
        self.executed = []
        self.updated_ids = []
        self._last_returned = []

    def execute(self, sql, params=None):
        self.executed.append((sql, params))
        assert not sql.lstrip().upper().startswith("DELETE")
        if sql.lstrip().upper().startswith("UPDATE"):
            proxy_id = params[0]
            external_key = params[1]
            self.updated_ids.append(proxy_id)
            self._last_returned = [(proxy_id,)]
            assert external_key.startswith("missing-")

    def fetchall(self):
        return self._last_returned

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc, tb):
        return False


class RecordingConnection:
    def __init__(self):
        self.cursor_obj = RecordingCursor()
        self.commits = 0

    def cursor(self):
        return self.cursor_obj

    def commit(self):
        self.commits += 1


def test_apply_inactive_updates_only_candidate_ids():
    conn = RecordingConnection()
    candidates = [
        {"id": 11, "external_key": "missing-a"},
        {"id": 12, "external_key": "missing-b"},
    ]

    updated = reconcile.apply_inactive(conn, candidates)

    assert updated == [11, 12]
    assert conn.cursor_obj.updated_ids == [11, 12]
    assert conn.commits == 1
    for sql, params in conn.cursor_obj.executed:
        assert "SET status = 'inactive'" in sql
        assert "BTRIM(p.external_key) = %s" in sql
        assert "deleted_at IS NULL" in sql
        assert "NOT EXISTS" in sql
        assert params[0] in (11, 12)
