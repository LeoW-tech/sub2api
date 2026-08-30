# Fast Responses bridge

This service keeps the public `/responses` and `/v1/responses` HTTP contract
while forwarding explicit `priority`/`fast` requests through the local
Sub2API Responses WebSocket ingress. API keys listed in
`/etc/sub2api/fast-bridge.env` are forced to `service_tier=priority`.

The bridge listens only on loopback (`127.0.0.1:18081`) and is exposed by the
Nginx exact locations for the two Responses paths. Credentials are never kept
in this directory or in Git.
