import json

from server import should_bridge, websocket_payload


def test_admin_key_bridges_and_forces_priority():
    assert should_bridge("sk-admin", {}, {"sk-admin"})
    payload = websocket_payload({"model": "gpt-5.6-sol", "input": "hello"}, force=True)
    assert payload["type"] == "response.create"
    assert payload["stream"] is True
    assert payload["service_tier"] == "priority"


def test_explicit_fast_alias_bridges_without_force():
    body = {"model": "gpt-5.6-sol", "service_tier": "fast"}
    assert should_bridge("sk-other", body, set())
    assert websocket_payload(body, force=False)["service_tier"] == "priority"


def test_default_request_stays_http_for_non_admin():
    assert not should_bridge("sk-other", {"model": "gpt-5.6-sol"}, set())
