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

test('updateHealthOnce restarts a worker when its local listen port stops responding', async () => {
  const events = []
  let activeChild = null

  function createChild(pid) {
    const exitListeners = new Set()
    return {
      pid,
      exitCode: null,
      once(event, listener) {
        assert.equal(event, 'exit')
        exitListeners.add(listener)
      },
      kill(signal) {
        events.push(`kill:${pid}:${signal}`)
        this.exitCode = 0
        for (const listener of exitListeners) {
          listener(0, signal)
        }
      }
    }
  }

  const runtime = new DoorRuntime(
    {
      mihomoBinary: '/usr/bin/fake-mihomo',
      sub2apiExportHost: 'host.docker.internal',
      healthcheckIntervalMs: 30000,
      doors: [
        {
          key: 'door-restart',
          name: 'Restart Door',
          protocol: 'http',
          listen_host: '0.0.0.0',
          probe_host: '127.0.0.1',
          listen_port: 59010,
          controller_host: '127.0.0.1',
          controller_port: 60010,
          worker_dir: '/tmp/door-restart',
          secret: 'door-secret',
          enabled: true,
          upstream_proxy: { name: 'Restart Door', type: 'http', server: '127.0.0.1', port: 8080 }
        }
      ]
    },
    {
      startupTimeoutMs: 50,
      spawnImpl(_command, _args, _options) {
        activeChild = createChild(2002)
        events.push('spawn:2002')
        return activeChild
      },
      probePort: async (_host, _port) => {
        if (activeChild?.pid === 2002) {
          return { ok: true }
        }
        return { ok: false, error: 'connect ECONNREFUSED 127.0.0.1:59010' }
      }
    }
  )

  const staleChild = createChild(1001)
  runtime.workers.set('door-restart', staleChild)
  runtime.states.set('door-restart', {
    online: true,
    last_checked_at: null,
    last_error: null,
    pid: 1001
  })

  await runtime.updateHealthOnce()

  assert.deepEqual(events, ['kill:1001:SIGTERM', 'spawn:2002'])
  assert.equal(runtime.workers.get('door-restart')?.pid, 2002)

  const state = runtime.states.get('door-restart')
  assert.equal(state?.online, true)
  assert.equal(state?.last_error, null)
  assert.equal(state?.pid, 2002)
})
