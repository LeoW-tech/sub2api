#!/usr/bin/env python3
"""HTTP-to-Responses-WebSocket bridge for the local Sub2API gateway.

The bridge keeps the public OpenAI Responses HTTP contract while using the
gateway's WebSocket ingress for requests that explicitly request Fast or are
authenticated with the configured admin key.
"""

from __future__ import annotations

import asyncio
import json
import logging
import os
import secrets
from typing import Any

from aiohttp import ClientSession, ClientTimeout, WSMsgType, web


LOG = logging.getLogger("sub2api-fast-bridge")
UPSTREAM_HTTP_URL = os.environ.get(
    "UPSTREAM_HTTP_URL", "http://127.0.0.1:8080/v1/responses"
)
UPSTREAM_WS_URL = os.environ.get(
    "UPSTREAM_WS_URL", "ws://127.0.0.1:8080/v1/responses"
)
LISTEN_HOST = os.environ.get("LISTEN_HOST", "127.0.0.1")
LISTEN_PORT = int(os.environ.get("LISTEN_PORT", "18081"))
MAX_BODY_BYTES = int(os.environ.get("MAX_BODY_BYTES", str(64 * 1024 * 1024)))


def configured_api_keys() -> set[str]:
    return {
        value.strip()
        for value in os.environ.get("FAST_BRIDGE_API_KEYS", "").split(",")
        if value.strip()
    }


def configured_user_agent_patterns() -> tuple[str, ...]:
    return tuple(
        value.strip().lower()
        for value in os.environ.get(
            "FAST_BRIDGE_USER_AGENT_PATTERNS", "Codex Desktop,Codex CLI"
        ).split(",")
        if value.strip()
    )


def bearer_token(request: web.Request) -> str:
    value = request.headers.get("Authorization", "")
    scheme, _, token = value.partition(" ")
    if scheme.lower() != "bearer":
        return ""
    return token.strip()


def token_matches(token: str, candidates: set[str]) -> bool:
    return any(secrets.compare_digest(token, candidate) for candidate in candidates)


def should_bridge(token: str, body: dict[str, Any], admin_keys: set[str]) -> bool:
    if token_matches(token, admin_keys):
        return True
    tier = str(body.get("service_tier", "")).strip().lower()
    return tier in {"priority", "fast"}


def websocket_payload(body: dict[str, Any], force: bool) -> dict[str, Any]:
    payload = dict(body)
    payload["type"] = "response.create"
    payload["stream"] = True
    if force or str(payload.get("service_tier", "")).strip().lower() == "fast":
        payload["service_tier"] = "priority"
    return payload


def forwarded_headers(request: web.Request) -> dict[str, str]:
    allowed = {
        "authorization",
        "user-agent",
        "originator",
        "x-client-request-id",
        "chatgpt-account-id",
        "openai-beta",
        "x-codex-client-version",
    }
    result: dict[str, str] = {}
    for name, value in request.headers.items():
        lower = name.lower()
        if lower in allowed or lower.startswith("x-codex-"):
            result[name] = value
    return result


def json_error(message: str, status: int = 502) -> web.Response:
    return web.json_response(
        {"error": {"type": "upstream_error", "message": message}}, status=status
    )


def event_response(payload: dict[str, Any]) -> dict[str, Any] | None:
    if payload.get("type") in {"response.completed", "response.done"}:
        response = payload.get("response")
        return response if isinstance(response, dict) else None
    return None


def event_bytes(payload: dict[str, Any]) -> bytes:
    event_type = str(payload.get("type", "message"))
    encoded = json.dumps(payload, separators=(",", ":"), ensure_ascii=False)
    return f"event: {event_type}\ndata: {encoded}\n\n".encode("utf-8")


class Bridge:
    def __init__(self) -> None:
        self.session: ClientSession | None = None

    async def start(self, _app: web.Application) -> None:
        self.session = ClientSession(
            timeout=ClientTimeout(total=None, sock_connect=30, sock_read=None)
        )

    async def stop(self, _app: web.Application) -> None:
        if self.session is not None:
            await self.session.close()
            self.session = None

    async def upstream_ws(self, request: web.Request):
        if self.session is None:
            raise RuntimeError("bridge session is not initialized")
        return await self.session.ws_connect(
            UPSTREAM_WS_URL,
            headers=forwarded_headers(request),
            autoping=True,
            heartbeat=25,
            max_msg_size=MAX_BODY_BYTES,
        )

    async def upstream_http(self, request: web.Request, body: bytes) -> web.StreamResponse:
        if self.session is None:
            return json_error("bridge session is not initialized", 503)
        try:
            async with self.session.post(
                UPSTREAM_HTTP_URL,
                data=body,
                headers={**forwarded_headers(request), "Content-Type": "application/json"},
            ) as upstream:
                response_headers = {
                    key: value
                    for key, value in upstream.headers.items()
                    if key.lower()
                    in {"content-type", "cache-control", "x-request-id", "retry-after"}
                }
                response = web.StreamResponse(
                    status=upstream.status, headers=response_headers
                )
                await response.prepare(request)
                async for chunk in upstream.content.iter_chunked(64 * 1024):
                    await response.write(chunk)
                await response.write_eof()
                return response
        except asyncio.CancelledError:
            raise
        except Exception as exc:  # pragma: no cover - network failure path
            LOG.warning("HTTP fallback failed: %s", exc)
            return json_error("upstream HTTP request failed")

    async def responses(self, request: web.Request) -> web.StreamResponse:
        if request.headers.get("Upgrade", "").lower() == "websocket":
            return await self.websocket_proxy(request)

        try:
            body_bytes = await request.read()
            if len(body_bytes) > MAX_BODY_BYTES:
                return json_error("request body is too large", 413)
            body = json.loads(body_bytes)
            if not isinstance(body, dict):
                return web.json_response(
                    {"error": {"type": "invalid_request_error", "message": "body must be an object"}},
                    status=400,
                )
        except json.JSONDecodeError:
            return web.json_response(
                {"error": {"type": "invalid_request_error", "message": "invalid JSON body"}},
                status=400,
            )

        token = bearer_token(request)
        admin_keys = configured_api_keys()
        admin = token_matches(token, admin_keys)
        user_agent = request.headers.get("User-Agent", "")
        client_fast = any(
            pattern in user_agent.lower() for pattern in configured_user_agent_patterns()
        )
        bridge_request = should_bridge(token, body, admin_keys) or client_fast
        LOG.info(
            "route method=%s bridge=%s admin_key=%s client_fast=%s tier=%s ua=%s",
            request.method,
            bridge_request,
            admin,
            client_fast,
            str(body.get("service_tier", ""))[:24],
            user_agent[:96],
        )
        if not bridge_request:
            return await self.upstream_http(request, body_bytes)

        try:
            upstream = await self.upstream_ws(request)
            await upstream.send_str(
                json.dumps(
                    websocket_payload(body, force=admin or client_fast),
                    separators=(",", ":"),
                )
            )
        except Exception as exc:
            LOG.warning("WebSocket bridge connect failed: %s", exc)
            return json_error("upstream WebSocket connection failed")

        stream = bool(body.get("stream", False))
        final_response: dict[str, Any] | None = None
        if stream:
            response = web.StreamResponse(
                status=200,
                headers={
                    "Content-Type": "text/event-stream; charset=utf-8",
                    "Cache-Control": "no-cache",
                    "Connection": "keep-alive",
                    "X-Accel-Buffering": "no",
                },
            )
            await response.prepare(request)
        client_disconnected = False
        try:
            async for message in upstream:
                if message.type == WSMsgType.TEXT:
                    try:
                        payload = json.loads(message.data)
                    except json.JSONDecodeError:
                        continue
                    if not isinstance(payload, dict):
                        continue
                    final_response = event_response(payload) or final_response
                    if stream:
                        await response.write(event_bytes(payload))
                    if payload.get("type") in {"response.completed", "response.done", "response.failed"}:
                        break
                elif message.type == WSMsgType.BINARY:
                    try:
                        payload = json.loads(message.data.decode("utf-8"))
                    except (UnicodeDecodeError, json.JSONDecodeError):
                        continue
                    if isinstance(payload, dict) and stream:
                        await response.write(event_bytes(payload))
                elif message.type in {WSMsgType.CLOSE, WSMsgType.CLOSED, WSMsgType.ERROR}:
                    break
        except ConnectionResetError:
            client_disconnected = True
        except asyncio.CancelledError:
            raise
        except Exception as exc:  # pragma: no cover - network failure path
            LOG.warning("WebSocket bridge relay failed: %s", exc)
            if not stream:
                return json_error("upstream WebSocket relay failed")
        finally:
            await upstream.close()

        if stream:
            if not client_disconnected:
                await response.write_eof()
            return response
        if final_response is None:
            return json_error("upstream WebSocket returned no completed response")
        return web.json_response(final_response)

    async def websocket_proxy(self, request: web.Request) -> web.StreamResponse:
        downstream = web.WebSocketResponse(autoping=True, heartbeat=25, max_msg_size=MAX_BODY_BYTES)
        await downstream.prepare(request)
        try:
            upstream = await self.upstream_ws(request)
        except Exception as exc:
            LOG.warning("WebSocket passthrough connect failed: %s", exc)
            await downstream.close(code=1011, message=b"upstream websocket unavailable")
            return downstream

        async def client_to_upstream() -> None:
            async for message in downstream:
                if message.type == WSMsgType.TEXT:
                    await upstream.send_str(message.data)
                elif message.type == WSMsgType.BINARY:
                    await upstream.send_bytes(message.data)
                elif message.type == WSMsgType.CLOSE:
                    break

        async def upstream_to_client() -> None:
            async for message in upstream:
                if message.type == WSMsgType.TEXT:
                    await downstream.send_str(message.data)
                elif message.type == WSMsgType.BINARY:
                    await downstream.send_bytes(message.data)
                elif message.type in {WSMsgType.CLOSE, WSMsgType.CLOSED, WSMsgType.ERROR}:
                    break

        tasks = {
            asyncio.create_task(client_to_upstream()),
            asyncio.create_task(upstream_to_client()),
        }
        try:
            await asyncio.wait(tasks, return_when=asyncio.FIRST_COMPLETED)
        finally:
            for task in tasks:
                task.cancel()
            await upstream.close()
            await downstream.close()
        return downstream


def create_app() -> web.Application:
    bridge = Bridge()
    app = web.Application(client_max_size=MAX_BODY_BYTES)
    app.cleanup_ctx.append(lambda app: _bridge_context(app, bridge))
    app.router.add_get("/healthz", lambda _request: web.json_response({"status": "ok"}))
    app.router.add_route("*", "/v1/responses", bridge.responses)
    app.router.add_route("*", "/responses", bridge.responses)
    return app


async def _bridge_context(app: web.Application, bridge: Bridge):
    await bridge.start(app)
    try:
        yield
    finally:
        await bridge.stop(app)


if __name__ == "__main__":
    logging.basicConfig(
        level=os.environ.get("LOG_LEVEL", "INFO"),
        format="%(asctime)s %(levelname)s %(name)s %(message)s",
    )
    web.run_app(create_app(), host=LISTEN_HOST, port=LISTEN_PORT, access_log=None)
