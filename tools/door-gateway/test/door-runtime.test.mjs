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

test('updateHealthOnce probes loopback host when worker listens on 0.0.0.0', async () => {
  const runtime = new DoorRuntime(
    {
      sub2apiExportHost: 'host.docker.internal',
      healthcheckIntervalMs: 30000,
      doors: [
        {
          key: 'door-public',
          name: 'Public Door',
          protocol: 'http',
          listen_host: '0.0.0.0',
          probe_host: '127.0.0.1',
          listen_port: 59010,
          enabled: true
        }
      ]
    },
    {
      startupTimeoutMs: 10,
      probePort: async (host, port) => {
        assert.equal(host, '127.0.0.1')
        assert.equal(port, 59010)
        return { ok: true }
      }
    }
  )

  await runtime.updateHealthOnce()

  const state = runtime.states.get('door-public')
  assert.equal(state?.online, true)
  assert.equal(state?.last_error, null)
})
