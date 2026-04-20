import fs from 'node:fs'
import path from 'node:path'
import net from 'node:net'
import { spawn } from 'node:child_process'

import { buildWorkerConfig } from './mihomo-config.mjs'

function nowISO() {
  return new Date().toISOString()
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

function probePort(host, port) {
  return new Promise((resolve) => {
    const socket = net.createConnection({ host, port }, () => {
      socket.destroy()
      resolve({ ok: true })
    })
    socket.setTimeout(3000)
    socket.on('timeout', () => {
      socket.destroy()
      resolve({ ok: false, error: 'timeout' })
    })
    socket.on('error', (error) => {
      resolve({ ok: false, error: error.message })
    })
  })
}

async function waitForPort(host, port, timeoutMs, probeFn = probePort) {
  const startedAt = Date.now()
  while (Date.now() - startedAt < timeoutMs) {
    const result = await probeFn(host, port)
    if (result.ok) {
      return
    }
    await sleep(200)
  }
  throw new Error(`worker port did not become ready in time: ${host}:${port}`)
}

function defaultSpawnImpl(command, args, options) {
  return spawn(command, args, options)
}

export class DoorRuntime {
  constructor(config, dependencies = {}) {
    this.config = config
    this.states = new Map()
    this.healthTimer = null
    this.workers = new Map()
    this.restartLocks = new Set()
    this.isStopping = false
    this.spawnImpl = dependencies.spawnImpl || defaultSpawnImpl
    this.probePort = dependencies.probePort || probePort
    this.startupTimeoutMs = dependencies.startupTimeoutMs || 15000
  }

  buildExportPayload() {
    return {
      type: 'sub2api-data',
      version: 1,
      exported_at: nowISO(),
      proxies: this.config.doors
        .filter((door) => door.enabled !== false)
        .map((door) => ({
          proxy_key: `${door.protocol}|${this.config.sub2apiExportHost}|${door.listen_port}||`,
          proxy_external_key: door.key,
          name: door.name,
          protocol: door.protocol,
          host: this.config.sub2apiExportHost,
          port: door.listen_port,
          status: 'active',
          exit_ip: door.exit_ip || undefined
        })),
      accounts: []
    }
  }

  snapshot() {
    return this.config.doors.map((door) => ({
      ...door,
      state: this.states.get(door.key) || {
        online: false,
        last_checked_at: null,
        last_error: 'not checked',
        pid: null
      }
    }))
  }

  async updateHealthOnce() {
    for (const door of this.config.doors) {
      if (door.enabled === false) {
        this.states.set(door.key, {
          online: false,
          last_checked_at: nowISO(),
          last_error: 'disabled',
          pid: null
        })
        continue
      }

      const child = this.workers.get(door.key)
      if ((!child || child.exitCode !== null) && !this.isStopping) {
        await this.ensureWorkerRunning(door)
      }

      const result = await this.probePort(door.probe_host || door.listen_host, door.listen_port)
      this.states.set(door.key, {
        online: result.ok,
        last_checked_at: nowISO(),
        last_error: result.ok ? null : result.error,
        pid: this.workers.get(door.key)?.pid || null
      })
    }
  }

  async start() {
    fs.mkdirSync(this.config.workerBaseDir, { recursive: true })

    for (const door of this.config.doors) {
      if (door.enabled === false) continue

      await this.startWorker(door)
    }

    await this.updateHealthOnce()
    this.healthTimer = setInterval(() => {
      this.updateHealthOnce().catch((error) => {
        console.error('[door-gateway] health update failed', error)
      })
    }, this.config.healthcheckIntervalMs)
  }

  async startWorker(door) {
    fs.mkdirSync(door.worker_dir, { recursive: true })

    const configPath = path.join(door.worker_dir, 'config.yaml')
    fs.writeFileSync(configPath, buildWorkerConfig(door), 'utf8')

    const stdoutPath = path.join(door.worker_dir, 'stdout.log')
    const stderrPath = path.join(door.worker_dir, 'stderr.log')
    const stdout = fs.openSync(stdoutPath, 'a')
    const stderr = fs.openSync(stderrPath, 'a')

    const child = this.spawnImpl(
      this.config.mihomoBinary,
      [
        '-d',
        door.worker_dir,
        '-f',
        configPath,
        '-ext-ctl',
        `${door.controller_host || '127.0.0.1'}:${door.controller_port}`,
        '-secret',
        door.secret
      ],
      {
        detached: false,
        stdio: ['ignore', stdout, stderr]
      }
    )

    child.once('exit', (code, signal) => {
      const previous = this.states.get(door.key)
      this.states.set(door.key, {
        online: false,
        last_checked_at: nowISO(),
        last_error: previous?.last_error || `worker exited (${signal || code || 'unknown'})`,
        pid: null
      })
      this.workers.delete(door.key)
    })

    this.workers.set(door.key, child)

    try {
      await waitForPort(
        door.probe_host || door.listen_host,
        door.listen_port,
        this.startupTimeoutMs,
        this.probePort
      )
      this.states.set(door.key, {
        online: true,
        last_checked_at: nowISO(),
        last_error: null,
        pid: child.pid || null
      })
    } catch (error) {
      child.kill('SIGTERM')
      this.workers.delete(door.key)
      throw error
    }
  }

  async stop() {
    this.isStopping = true
    if (this.healthTimer) {
      clearInterval(this.healthTimer)
      this.healthTimer = null
    }

    await Promise.all(
      Array.from(this.workers.values()).map(
        (child) =>
          new Promise((resolve) => {
            if (child.exitCode !== null) {
              resolve()
              return
            }

            child.once('exit', () => resolve())
            child.kill('SIGTERM')
            setTimeout(() => {
              if (child.exitCode === null) {
                child.kill('SIGKILL')
              }
            }, 3000).unref()
          })
      )
    )

    this.workers.clear()
  }

  async ensureWorkerRunning(door) {
    if (this.restartLocks.has(door.key)) {
      return
    }

    this.restartLocks.add(door.key)
    try {
      await this.startWorker(door)
    } catch (error) {
      this.states.set(door.key, {
        online: false,
        last_checked_at: nowISO(),
        last_error: error.message,
        pid: null
      })
    } finally {
      this.restartLocks.delete(door.key)
    }
  }
}
