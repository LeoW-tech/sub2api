import test from 'node:test'
import assert from 'node:assert/strict'
import { DoorRuntime } from '../src/door-runtime.mjs'

test('buildExportPayload emits sub2api-compatible proxies with external keys', () => {
  const runtime = new DoorRuntime({
    sub2apiExportHost: 'host.docker.internal',
    healthcheckIntervalMs: 30000,
    doors: [
      {
        key: 'door-hk-w10',
        name: '🇭🇰 香港 W10 | IEPL',
        protocol: 'http',
        listen_host: '127.0.0.1',
        listen_port: 59010,
        target_host: '127.0.0.1',
        target_port: 7890,
        enabled: true,
        exit_ip: '203.0.113.10'
      }
    ]
  })

  const payload = runtime.buildExportPayload()
  assert.equal(payload.type, 'sub2api-data')
  assert.equal(payload.proxies.length, 1)
  assert.deepEqual(payload.proxies[0], {
    proxy_key: 'http|host.docker.internal|59010||',
    proxy_external_key: 'door-hk-w10',
    name: '🇭🇰 香港 W10 | IEPL',
    protocol: 'http',
    host: 'host.docker.internal',
    port: 59010,
    status: 'active',
    exit_ip: '203.0.113.10'
  })
})
