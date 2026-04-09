import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import YAML from 'yaml'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const DEFAULT_CONFIG_PATH = path.resolve(__dirname, '../doors.example.json')

export function loadConfig() {
  return loadConfigFromPath(
    process.env.DOOR_GATEWAY_CONFIG
      ? path.resolve(process.env.DOOR_GATEWAY_CONFIG)
      : DEFAULT_CONFIG_PATH
  )
}

export function loadConfigFromPath(configPath) {
  const raw = fs.readFileSync(configPath, 'utf8')
  const parsed = JSON.parse(raw)

  if (!parsed.api?.host || !parsed.api?.port) {
    throw new Error('config.api.host and config.api.port are required')
  }
  if (!Array.isArray(parsed.doors) || parsed.doors.length === 0) {
    throw new Error('config.doors must be a non-empty array')
  }
  if (!parsed.mihomo_binary) {
    throw new Error('config.mihomo_binary is required')
  }
  if (!parsed.source_config_path) {
    throw new Error('config.source_config_path is required')
  }

  const sourceConfigPath = path.resolve(path.dirname(configPath), parsed.source_config_path)
  const sourceConfig = YAML.parse(fs.readFileSync(sourceConfigPath, 'utf8'))
  const proxyMap = buildProxyMap(sourceConfig)

  const workerBaseDir = path.resolve(
    path.dirname(configPath),
    parsed.worker_base_dir || '../door-workers'
  )
  const workerPortStart = parsed.worker_port_start || 58080
  const workerSocksPortStart = parsed.worker_socks_port_start || workerPortStart + 1000
  const controllerPortStart = parsed.controller_port_start || workerPortStart + 2000
  const exportProtocol = parsed.export_protocol || 'http'

  return {
    configPath,
    api: parsed.api,
    sub2apiExportHost: parsed.sub2api_export_host || 'host.docker.internal',
    healthcheckIntervalMs: parsed.healthcheck_interval_ms || 30000,
    mihomoBinary: path.resolve(path.dirname(configPath), parsed.mihomo_binary),
    workerBaseDir,
    doors: parsed.doors.map((door, index) => normalizeDoor(door, index, {
      proxyMap,
      exportProtocol,
      workerBaseDir,
      workerPortStart,
      workerSocksPortStart,
      controllerPortStart
    }))
  }
}

function normalizeDoor(door, index, options) {
  if (!door.key || !door.name) {
    throw new Error('each door must include key and name')
  }
  const proxyName = (door.proxy_name || door.name || '').trim()
  if (!proxyName) {
    throw new Error(`door ${door.key} must include proxy_name or name`)
  }
  const upstreamProxy = options.proxyMap.get(proxyName)
  if (!upstreamProxy) {
    throw new Error(`proxy_name not found in source config: ${proxyName}`)
  }

  return {
    enabled: true,
    protocol: options.exportProtocol,
    listen_host: '127.0.0.1',
    ...door,
    proxy_name: proxyName,
    listen_port: door.listen_port || options.workerPortStart + index,
    socks_port: door.socks_port || options.workerSocksPortStart + index,
    controller_port: door.controller_port || options.controllerPortStart + index,
    secret: door.secret || `door-${door.key}`,
    worker_dir: path.join(options.workerBaseDir, door.key),
    upstream_proxy: structuredClone(upstreamProxy)
  }
}

function buildProxyMap(sourceConfig) {
  const rawList = sourceConfig?.proxies || sourceConfig?.Proxy || []
  if (!Array.isArray(rawList) || rawList.length === 0) {
    throw new Error('source config does not contain proxies')
  }

  return new Map(
    rawList
      .filter((item) => item && typeof item === 'object' && typeof item.name === 'string')
      .map((item) => [item.name.trim(), item])
  )
}
