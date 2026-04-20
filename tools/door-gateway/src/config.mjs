import fs from 'node:fs'
import path from 'node:path'
import crypto from 'node:crypto'
import { fileURLToPath } from 'node:url'
import YAML from 'yaml'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const DEFAULT_CONFIG_PATH = path.resolve(__dirname, '../doors.example.json')

export async function loadConfig() {
  return loadConfigFromPath(
    process.env.DOOR_GATEWAY_CONFIG
      ? path.resolve(process.env.DOOR_GATEWAY_CONFIG)
      : DEFAULT_CONFIG_PATH
  )
}

export async function loadConfigFromPath(configPath) {
  const raw = fs.readFileSync(configPath, 'utf8')
  const parsed = JSON.parse(raw)
  const configDir = path.dirname(configPath)

  if (!parsed.api?.host || !parsed.api?.port) {
    throw new Error('config.api.host and config.api.port are required')
  }
  if (!parsed.mihomo_binary) {
    throw new Error('config.mihomo_binary is required')
  }

  const workerBaseDir = path.resolve(
    configDir,
    parsed.worker_base_dir || '../door-workers'
  )
  const workerPortStart = parsed.worker_port_start || 58080
  const workerSocksPortStart = parsed.worker_socks_port_start || workerPortStart + 1000
  const controllerPortStart = parsed.controller_port_start || workerPortStart + 2000
  const exportProtocol = parsed.export_protocol || 'http'
  const workerBindHost = normalizeBindHost(parsed.worker_bind_host, '127.0.0.1')
  const controllerBindHost = normalizeBindHost(parsed.controller_bind_host, '127.0.0.1')
  const doors = await resolveDoors(parsed, {
    controllerBindHost,
    configDir,
    exportProtocol,
    workerBindHost,
    workerBaseDir,
    workerPortStart,
    workerSocksPortStart,
    controllerPortStart
  })

  if (!Array.isArray(doors) || doors.length === 0) {
    throw new Error('resolved door list is empty')
  }

  return {
    configPath,
    api: parsed.api,
    sub2apiExportHost: parsed.sub2api_export_host || 'host.docker.internal',
    healthcheckIntervalMs: parsed.healthcheck_interval_ms || 30000,
    mihomoBinary: path.resolve(configDir, parsed.mihomo_binary),
    workerBaseDir,
    doors
  }
}

async function resolveDoors(parsed, options) {
  const resolvedDoors = []

  if (Array.isArray(parsed.doors) && parsed.doors.length > 0) {
    if (!parsed.source_config_path) {
      throw new Error('config.source_config_path is required')
    }

    const sourceConfigPath = path.resolve(options.configDir, parsed.source_config_path)
    const sourceConfig = parseSourceConfigText(fs.readFileSync(sourceConfigPath, 'utf8'), sourceConfigPath)
    const proxyMap = buildProxyMap(sourceConfig)

    resolvedDoors.push(
      ...parsed.doors.map((door, index) => normalizeDoor(door, index, {
        ...options,
        proxyMap
      }))
    )
  }

  if (Array.isArray(parsed.sources) && parsed.sources.length > 0) {
    resolvedDoors.push(
      ...(await loadDoorsFromSources(parsed.sources, {
        ...options,
        indexOffset: resolvedDoors.length
      }))
    )
  }

  if (resolvedDoors.length === 0) {
    throw new Error('config.doors or config.sources must provide at least one door')
  }

  const keySet = new Set()
  for (const door of resolvedDoors) {
    if (keySet.has(door.key)) {
      throw new Error(`duplicate door key: ${door.key}`)
    }
    keySet.add(door.key)
  }

  return resolvedDoors
}

async function loadDoorsFromSources(sources, options) {
  const generatedDoors = []

  for (const source of sources) {
    const normalizedSource = normalizeSourceDefinition(source)
    if (!normalizedSource.enabled) {
      continue
    }

    const sourceText = await readSourceText(normalizedSource, options.configDir)
    const sourceConfig = parseSourceConfigText(sourceText, normalizedSource.name)
    const proxyList = extractProxyList(sourceConfig)
    const seenFingerprints = new Set()

    for (const proxy of proxyList) {
      const proxyName = safeProxyName(proxy?.name)
      if (!proxyName) {
        continue
      }

      const fingerprint = buildProxyFingerprint(proxy)
      if (seenFingerprints.has(fingerprint)) {
        continue
      }
      seenFingerprints.add(fingerprint)

      generatedDoors.push({
        key: `${normalizedSource.name}-${fingerprint}`,
        name: proxyName,
        proxy_name: proxyName,
        source_name: normalizedSource.name,
        upstream_proxy: structuredClone(proxy)
      })
    }
  }

  const indexOffset = Number(options.indexOffset || 0)
  return generatedDoors.map((door, index) => normalizeDoor(door, indexOffset + index, options))
}

function normalizeDoor(door, index, options) {
  if (!door.key || !door.name) {
    throw new Error('each door must include key and name')
  }
  const proxyName = (door.proxy_name || door.name || '').trim()
  if (!proxyName) {
    throw new Error(`door ${door.key} must include proxy_name or name`)
  }
  const upstreamProxy = door.upstream_proxy
    ? structuredClone(door.upstream_proxy)
    : options.proxyMap?.get(proxyName)
  if (!upstreamProxy) {
    throw new Error(`proxy_name not found in source config: ${proxyName}`)
  }

  const listenHost = normalizeBindHost(door.listen_host, options.workerBindHost || '127.0.0.1')
  const controllerHost = normalizeBindHost(
    door.controller_host,
    options.controllerBindHost || '127.0.0.1'
  )
  const probeHost = normalizeProbeHost(door.probe_host, listenHost)

  return {
    enabled: true,
    protocol: options.exportProtocol,
    ...door,
    listen_host: listenHost,
    probe_host: probeHost,
    controller_host: controllerHost,
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
  const rawList = extractProxyList(sourceConfig)
  if (!Array.isArray(rawList) || rawList.length === 0) {
    throw new Error('source config does not contain proxies')
  }

  return new Map(
    rawList
      .filter((item) => item && typeof item === 'object' && typeof item.name === 'string')
      .map((item) => [item.name.trim(), item])
  )
}

function extractProxyList(sourceConfig) {
  return sourceConfig?.proxies || sourceConfig?.Proxy || []
}

function normalizeSourceDefinition(source) {
  if (!source || typeof source !== 'object') {
    throw new Error('each source must be an object')
  }

  const normalizedName = normalizeSourceName(source.name)
  const hasPath = Boolean(String(source.path || '').trim())
  const hasUrl = Boolean(String(source.url || '').trim())

  if (hasPath === hasUrl) {
    throw new Error(`source ${normalizedName} must include exactly one of path or url`)
  }

  return {
    name: normalizedName,
    enabled: source.enabled !== false,
    path: hasPath ? String(source.path).trim() : null,
    url: hasUrl ? String(source.url).trim() : null
  }
}

function normalizeSourceName(value) {
  const normalized = String(value || '')
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')

  if (!normalized) {
    throw new Error('source.name is required')
  }

  return normalized
}

async function readSourceText(source, configDir) {
  if (source.path) {
    return fs.readFileSync(path.resolve(configDir, source.path), 'utf8')
  }

  const response = await fetch(source.url)
  if (!response.ok) {
    throw new Error(`failed to fetch source ${source.name}: HTTP ${response.status}`)
  }

  return response.text()
}

function parseSourceConfigText(text, label) {
  try {
    return YAML.parse(text)
  } catch (error) {
    throw new Error(`failed to parse source config ${label}: ${error.message}`)
  }
}

function buildProxyFingerprint(proxy) {
  const normalized = [
    String(proxy?.type || '').trim().toLowerCase(),
    String(proxy?.server || '').trim().toLowerCase(),
    String(proxy?.port || '').trim(),
    safeProxyName(proxy?.name)
  ].join('|')

  return crypto.createHash('sha1').update(normalized).digest('hex').slice(0, 12)
}

function normalizeBindHost(value, fallback) {
  const normalized = String(value || '')
    .trim()
    .toLowerCase()
  if (normalized) {
    return normalized
  }
  return fallback
}

function normalizeProbeHost(value, listenHost) {
  const normalized = String(value || '')
    .trim()
    .toLowerCase()
  if (normalized) {
    return normalized
  }
  if (listenHost === '0.0.0.0') {
    return '127.0.0.1'
  }
  if (listenHost === '::') {
    return '::1'
  }
  return listenHost
}

function safeProxyName(value) {
  return String(value || '').trim()
}
