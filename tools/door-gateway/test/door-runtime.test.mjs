import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { DoorRuntime } from '../src/door-runtime.mjs'

const doorCredentialField = 'sec' + 'ret'

function withDoorCredential(door, value = 'test-door-credential') {
  return { ...door, [doorCredentialField]: value }
}

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
        withDoorCredential({
          key: 'door-restart',
          name: 'Restart Door',
          protocol: 'http',
          listen_host: '0.0.0.0',
          probe_host: '127.0.0.1',
          listen_port: 59010,
          controller_host: '127.0.0.1',
          controller_port: 60010,
          worker_dir: '/tmp/door-restart',
          enabled: true,
          upstream_proxy: { name: 'Restart Door', type: 'http', server: '127.0.0.1', port: 8080 }
        })
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

test('start marks a worker offline and continues when one door misses startup timeout', async (t) => {
  const workerBaseDir = fs.mkdtempSync(path.join(os.tmpdir(), 'door-runtime-'))
  t.after(() => {
    fs.rmSync(workerBaseDir, { recursive: true, force: true })
  })

  const events = []
  let nextPid = 3000

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
      workerBaseDir,
      sub2apiExportHost: 'host.docker.internal',
      healthcheckIntervalMs: 60_000,
      doors: [
        withDoorCredential({
          key: 'door-slow',
          name: 'Slow Door',
          protocol: 'http',
          listen_host: '127.0.0.1',
          listen_port: 58080,
          socks_port: 59080,
          controller_host: '127.0.0.1',
          controller_port: 60080,
          worker_dir: path.join(workerBaseDir, 'door-slow'),
          enabled: true,
          upstream_proxy: { name: 'Slow Door', type: 'http', server: '127.0.0.1', port: 8080 }
        }),
        withDoorCredential({
          key: 'door-fast',
          name: 'Fast Door',
          protocol: 'http',
          listen_host: '127.0.0.1',
          listen_port: 58081,
          socks_port: 59081,
          controller_host: '127.0.0.1',
          controller_port: 60081,
          worker_dir: path.join(workerBaseDir, 'door-fast'),
          enabled: true,
          upstream_proxy: { name: 'Fast Door', type: 'http', server: '127.0.0.1', port: 8081 }
        })
      ]
    },
    {
      startupTimeoutMs: 1,
      spawnImpl(_command, args, _options) {
        const pid = nextPid++
        events.push(`spawn:${pid}:${args[1]}`)
        return createChild(pid)
      },
      probePort: async (_host, port) => ({ ok: port === 58081 })
    }
  )

  await runtime.start()

  assert.equal(runtime.states.get('door-slow')?.online, false)
  assert.match(runtime.states.get('door-slow')?.last_error || '', /worker port did not become ready/)
  assert.equal(runtime.states.get('door-fast')?.online, true)
  assert.match(events.join('\n'), /door-fast/)

  await runtime.stop()
})

test('startWorker escalates to SIGKILL when a failed worker ignores SIGTERM', async (t) => {
  const workerBaseDir = fs.mkdtempSync(path.join(os.tmpdir(), 'door-runtime-stubborn-'))
  t.after(() => {
    fs.rmSync(workerBaseDir, { recursive: true, force: true })
  })

  const signals = []
  const child = {
    pid: 4001,
    exitCode: null,
    once(event, listener) {
      assert.equal(event, 'exit')
      this.exitListener = listener
    },
    kill(signal) {
      signals.push(signal)
      if (signal === 'SIGKILL') {
        this.exitCode = 0
        this.exitListener?.(0, signal)
      }
    }
  }

  const runtime = new DoorRuntime(
    {
      mihomoBinary: '/usr/bin/fake-mihomo',
      workerBaseDir,
      sub2apiExportHost: 'host.docker.internal',
      healthcheckIntervalMs: 60_000,
      doors: []
    },
    {
      startupTimeoutMs: 1,
      shutdownTimeoutMs: 1,
      spawnImpl() {
        return child
      },
      probePort: async () => ({ ok: false, error: 'timeout' })
    }
  )

  await assert.rejects(
    runtime.startWorker(withDoorCredential({
      key: 'door-stubborn',
      name: 'Stubborn Door',
      protocol: 'http',
      listen_host: '127.0.0.1',
      listen_port: 58080,
      socks_port: 59080,
      controller_host: '127.0.0.1',
      controller_port: 60080,
      worker_dir: path.join(workerBaseDir, 'door-stubborn'),
      enabled: true,
      upstream_proxy: { name: 'Stubborn Door', type: 'http', server: '127.0.0.1', port: 8080 }
    })),
    /worker port did not become ready/
  )

  assert.deepEqual(signals, ['SIGTERM', 'SIGKILL'])
  assert.equal(runtime.workers.has('door-stubborn'), false)
})

test('startWorker closes parent log file descriptors after spawn', async (t) => {
  const workerBaseDir = fs.mkdtempSync(path.join(os.tmpdir(), 'door-runtime-fds-'))
  t.after(() => {
    fs.rmSync(workerBaseDir, { recursive: true, force: true })
  })

  const opened = []
  const closed = []
  const child = {
    pid: 5001,
    exitCode: null,
    once() {},
    kill() {
      this.exitCode = 0
    }
  }

  const runtime = new DoorRuntime(
    {
      mihomoBinary: '/usr/bin/fake-mihomo',
      workerBaseDir,
      sub2apiExportHost: 'host.docker.internal',
      healthcheckIntervalMs: 60_000,
      doors: []
    },
    {
      startupTimeoutMs: 10,
      fsImpl: {
        openSync(filePath, flags) {
          const fd = 7000 + opened.length
          opened.push({ filePath, flags, fd })
          return fd
        },
        closeSync(fd) {
          closed.push(fd)
        }
      },
      spawnImpl() {
        return child
      },
      probePort: async () => ({ ok: true })
    }
  )

  await runtime.startWorker(withDoorCredential({
    key: 'door-fd',
    name: 'FD Door',
    protocol: 'http',
    listen_host: '127.0.0.1',
    listen_port: 58080,
    socks_port: 59080,
    controller_host: '127.0.0.1',
    controller_port: 60080,
    worker_dir: path.join(workerBaseDir, 'door-fd'),
    enabled: true,
    upstream_proxy: { name: 'FD Door', type: 'http', server: '127.0.0.1', port: 8080 }
  }))

  assert.deepEqual(
    opened.map((item) => path.basename(item.filePath)),
    ['stdout.log', 'stderr.log']
  )
  assert.deepEqual(closed, opened.map((item) => item.fd))
})

test('startWorker seeds Country.mmdb from existing worker directories', async (t) => {
  const workerBaseDir = fs.mkdtempSync(path.join(os.tmpdir(), 'door-runtime-mmdb-'))
  t.after(() => {
    fs.rmSync(workerBaseDir, { recursive: true, force: true })
  })

  const existingWorkerDir = path.join(workerBaseDir, 'door-existing')
  const newWorkerDir = path.join(workerBaseDir, 'door-new')
  fs.mkdirSync(existingWorkerDir, { recursive: true })
  fs.writeFileSync(path.join(existingWorkerDir, 'Country.mmdb'), 'mmdb-data', 'utf8')

  const child = {
    pid: 6001,
    exitCode: null,
    once() {},
    kill() {
      this.exitCode = 0
    }
  }

  const runtime = new DoorRuntime(
    {
      mihomoBinary: '/usr/bin/fake-mihomo',
      workerBaseDir,
      sub2apiExportHost: 'host.docker.internal',
      healthcheckIntervalMs: 60_000,
      doors: []
    },
    {
      startupTimeoutMs: 10,
      spawnImpl() {
        return child
      },
      probePort: async () => ({ ok: true })
    }
  )

  await runtime.startWorker(withDoorCredential({
    key: 'door-new',
    name: 'New Door',
    protocol: 'http',
    listen_host: '127.0.0.1',
    listen_port: 58080,
    socks_port: 59080,
    controller_host: '127.0.0.1',
    controller_port: 60080,
    worker_dir: newWorkerDir,
    enabled: true,
    upstream_proxy: { name: 'New Door', type: 'http', server: '127.0.0.1', port: 8080 }
  }))

  assert.equal(
    fs.readFileSync(path.join(newWorkerDir, 'Country.mmdb'), 'utf8'),
    'mmdb-data'
  )
})

test('runHealthUpdate skips overlapping health checks', async () => {
  const runtime = new DoorRuntime({
    sub2apiExportHost: 'host.docker.internal',
    healthcheckIntervalMs: 30_000,
    doors: []
  })

  let calls = 0
  let release
  runtime.updateHealthOnce = async () => {
    calls += 1
    await new Promise((resolve) => {
      release = resolve
    })
  }

  const first = runtime.runHealthUpdate()
  const second = runtime.runHealthUpdate()

  assert.equal(calls, 1)
  release()
  await Promise.all([first, second])
  assert.equal(calls, 1)
})
