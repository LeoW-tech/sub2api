import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import http from 'node:http'
import test from 'node:test'
import assert from 'node:assert/strict'

import { loadConfigFromPath } from '../src/config.mjs'
import { buildWorkerConfig } from '../src/mihomo-config.mjs'

test('loadConfigFromPath resolves Clash proxy definitions into door workers', async () => {
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'door-gateway-config-'))
  const sourceConfigPath = path.join(tempDir, 'source.yaml')
  const gatewayConfigPath = path.join(tempDir, 'doors.json')

  fs.writeFileSync(
    sourceConfigPath,
    [
      'mixed-port: 7890',
      'proxies:',
      '  - name: 🇭🇰 香港W10 | IEPL',
      '    type: ss',
      '    server: hk10.example.com',
      '    port: 443',
      '    cipher: aes-256-gcm',
      '    password: secret-1',
      '  - name: 🇯🇵 日本W07 | IEPL',
      '    type: vmess',
      '    server: jp07.example.com',
      '    port: 8443',
      '    uuid: secret-2',
      '    alterId: 0',
      '    cipher: auto',
      ''
    ].join('\n'),
    'utf8'
  )

  fs.writeFileSync(
    gatewayConfigPath,
    JSON.stringify(
      {
        api: { host: '127.0.0.1', port: 19080 },
        mihomo_binary: '/Applications/Clash Verge.app/Contents/MacOS/verge-mihomo',
        source_config_path: sourceConfigPath,
        worker_base_dir: path.join(tempDir, 'workers'),
        worker_port_start: 58080,
        worker_socks_port_start: 58180,
        controller_port_start: 58280,
        sub2api_export_host: 'host.docker.internal',
        doors: [
          {
            key: 'door-hk-w10',
            name: '🇭🇰 香港W10 | IEPL',
            proxy_name: '🇭🇰 香港W10 | IEPL',
            exit_ip: '203.0.113.10'
          },
          {
            key: 'door-jp-w07',
            name: '🇯🇵 日本W07 | IEPL',
            proxy_name: '🇯🇵 日本W07 | IEPL'
          }
        ]
      },
      null,
      2
    ),
    'utf8'
  )

  const config = await loadConfigFromPath(gatewayConfigPath)

  assert.equal(config.doors.length, 2)
  assert.equal(config.doors[0].listen_host, '127.0.0.1')
  assert.equal(config.doors[0].probe_host, '127.0.0.1')
  assert.equal(config.doors[0].controller_host, '127.0.0.1')
  assert.equal(config.doors[0].listen_port, 58080)
  assert.equal(config.doors[0].socks_port, 58180)
  assert.equal(config.doors[0].controller_port, 58280)
  assert.equal(config.doors[0].upstream_proxy.server, 'hk10.example.com')
  assert.equal(config.doors[1].listen_port, 58081)
  assert.equal(config.doors[1].socks_port, 58181)
  assert.equal(config.doors[1].controller_port, 58281)
  assert.equal(config.doors[1].upstream_proxy.type, 'vmess')
})

test('loadConfigFromPath applies worker and controller bind hosts to generated doors', async () => {
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'door-gateway-bind-hosts-'))
  const sourceConfigPath = path.join(tempDir, 'source.yaml')
  const gatewayConfigPath = path.join(tempDir, 'doors.json')

  fs.writeFileSync(
    sourceConfigPath,
    [
      'proxies:',
      '  - name: Public Door',
      '    type: ss',
      '    server: public-door.example.com',
      '    port: 443',
      '    cipher: aes-256-gcm',
      '    password: secret-1',
      ''
    ].join('\n'),
    'utf8'
  )

  fs.writeFileSync(
    gatewayConfigPath,
    JSON.stringify(
      {
        api: { host: '127.0.0.1', port: 19080 },
        mihomo_binary: '/Applications/Clash Verge.app/Contents/MacOS/verge-mihomo',
        worker_base_dir: path.join(tempDir, 'workers'),
        worker_bind_host: '0.0.0.0',
        controller_bind_host: '127.0.0.1',
        sources: [{ name: 'public', path: './source.yaml' }]
      },
      null,
      2
    ),
    'utf8'
  )

  const config = await loadConfigFromPath(gatewayConfigPath)

  assert.equal(config.doors.length, 1)
  assert.equal(config.doors[0].listen_host, '0.0.0.0')
  assert.equal(config.doors[0].probe_host, '127.0.0.1')
  assert.equal(config.doors[0].controller_host, '127.0.0.1')
})

test('loadConfigFromPath aggregates doors from multiple source files with stable unique keys', async () => {
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'door-gateway-multi-source-'))
  const firstSourcePath = path.join(tempDir, 'first.yaml')
  const secondSourcePath = path.join(tempDir, 'second.yaml')
  const gatewayConfigPath = path.join(tempDir, 'doors.json')

  fs.writeFileSync(
    firstSourcePath,
    [
      'proxies:',
      '  - name: Shared Node',
      '    type: ss',
      '    server: shared-a.example.com',
      '    port: 443',
      '    cipher: aes-256-gcm',
      '    password: source-a',
      '  - name: Japan Fast',
      '    type: vmess',
      '    server: jp-fast.example.com',
      '    port: 8443',
      '    uuid: source-a-uuid',
      '    alterId: 0',
      '    cipher: auto',
      ''
    ].join('\n'),
    'utf8'
  )

  fs.writeFileSync(
    secondSourcePath,
    [
      'proxies:',
      '  - name: Shared Node',
      '    type: ss',
      '    server: shared-b.example.com',
      '    port: 8443',
      '    cipher: aes-256-gcm',
      '    password: source-b',
      '  - name: Singapore Edge',
      '    type: trojan',
      '    server: sg-edge.example.com',
      '    port: 443',
      '    password: source-b-pass',
      '    sni: sg-edge.example.com',
      ''
    ].join('\n'),
    'utf8'
  )

  fs.writeFileSync(
    gatewayConfigPath,
    JSON.stringify(
      {
        api: { host: '127.0.0.1', port: 19080 },
        mihomo_binary: '/Applications/Clash Verge.app/Contents/MacOS/verge-mihomo',
        worker_base_dir: path.join(tempDir, 'workers'),
        worker_port_start: 58080,
        worker_socks_port_start: 58180,
        controller_port_start: 58280,
        sources: [
          { name: 'nomad', path: './first.yaml' },
          { name: 'trojanflare', path: './second.yaml' }
        ]
      },
      null,
      2
    ),
    'utf8'
  )

  const firstLoad = await loadConfigFromPath(gatewayConfigPath)
  const secondLoad = await loadConfigFromPath(gatewayConfigPath)

  assert.equal(firstLoad.doors.length, 4)
  assert.deepEqual(
    firstLoad.doors.map((door) => door.key),
    secondLoad.doors.map((door) => door.key)
  )
  assert.equal(new Set(firstLoad.doors.map((door) => door.key)).size, 4)

  const sharedNodeDoors = firstLoad.doors.filter((door) => door.name === 'Shared Node')
  assert.equal(sharedNodeDoors.length, 2)
  assert.notEqual(sharedNodeDoors[0].key, sharedNodeDoors[1].key)
  assert.ok(sharedNodeDoors.some((door) => door.key.startsWith('nomad-')))
  assert.ok(sharedNodeDoors.some((door) => door.key.startsWith('trojanflare-')))
})

test('loadConfigFromPath fetches source configs from url and skips disabled sources', async () => {
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'door-gateway-remote-source-'))
  const disabledSourcePath = path.join(tempDir, 'disabled.yaml')
  const gatewayConfigPath = path.join(tempDir, 'doors.json')
  const sourcePayload = [
    'proxies:',
    '  - name: Remote Door',
    '    type: ss',
    '    server: remote.example.com',
    '    port: 443',
    '    cipher: aes-256-gcm',
    '    password: remote-secret',
    ''
  ].join('\n')

  fs.writeFileSync(
    disabledSourcePath,
    [
      'proxies:',
      '  - name: Disabled Door',
      '    type: ss',
      '    server: disabled.example.com',
      '    port: 443',
      '    cipher: aes-256-gcm',
      '    password: disabled-secret',
      ''
    ].join('\n'),
    'utf8'
  )

  const server = http.createServer((req, res) => {
    if (req.url === '/subscription.yaml') {
      res.writeHead(200, { 'content-type': 'text/yaml; charset=utf-8' })
      res.end(sourcePayload)
      return
    }
    res.writeHead(404)
    res.end('not found')
  })

  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve))
  const address = server.address()
  const port = typeof address === 'object' && address ? address.port : 0

  fs.writeFileSync(
    gatewayConfigPath,
    JSON.stringify(
      {
        api: { host: '127.0.0.1', port: 19080 },
        mihomo_binary: '/Applications/Clash Verge.app/Contents/MacOS/verge-mihomo',
        worker_base_dir: path.join(tempDir, 'workers'),
        sources: [
          { name: 'remote', url: `http://127.0.0.1:${port}/subscription.yaml` },
          { name: 'disabled', path: './disabled.yaml', enabled: false }
        ]
      },
      null,
      2
    ),
    'utf8'
  )

  try {
    const config = await loadConfigFromPath(gatewayConfigPath)
    assert.equal(config.doors.length, 1)
    assert.equal(config.doors[0].name, 'Remote Door')
    assert.match(config.doors[0].key, /^remote-/)
  } finally {
    await new Promise((resolve, reject) => {
      server.close((error) => (error ? reject(error) : resolve()))
    })
  }
})

test('loadConfigFromPath preserves legacy doors and appends generated source doors after them', async () => {
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'door-gateway-hybrid-source-'))
  const legacySourcePath = path.join(tempDir, 'legacy.yaml')
  const extraSourcePath = path.join(tempDir, 'extra.yaml')
  const gatewayConfigPath = path.join(tempDir, 'doors.json')

  fs.writeFileSync(
    legacySourcePath,
    [
      'proxies:',
      '  - name: Legacy JP',
      '    type: ss',
      '    server: legacy-jp.example.com',
      '    port: 443',
      '    cipher: aes-256-gcm',
      '    password: legacy-jp-secret',
      '  - name: Legacy HK',
      '    type: ss',
      '    server: legacy-hk.example.com',
      '    port: 443',
      '    cipher: aes-256-gcm',
      '    password: legacy-hk-secret',
      ''
    ].join('\n'),
    'utf8'
  )

  fs.writeFileSync(
    extraSourcePath,
    [
      'proxies:',
      '  - name: Extra SG',
      '    type: trojan',
      '    server: extra-sg.example.com',
      '    port: 443',
      '    password: extra-sg-secret',
      '    sni: extra-sg.example.com',
      ''
    ].join('\n'),
    'utf8'
  )

  fs.writeFileSync(
    gatewayConfigPath,
    JSON.stringify(
      {
        api: { host: '127.0.0.1', port: 19080 },
        mihomo_binary: '/Applications/Clash Verge.app/Contents/MacOS/verge-mihomo',
        source_config_path: './legacy.yaml',
        worker_base_dir: path.join(tempDir, 'workers'),
        worker_port_start: 58080,
        worker_socks_port_start: 58180,
        controller_port_start: 58280,
        doors: [
          {
            key: 'door-legacy-jp',
            name: 'Legacy JP',
            proxy_name: 'Legacy JP'
          },
          {
            key: 'door-legacy-hk',
            name: 'Legacy HK',
            proxy_name: 'Legacy HK'
          }
        ],
        sources: [
          {
            name: 'extra',
            path: './extra.yaml'
          }
        ]
      },
      null,
      2
    ),
    'utf8'
  )

  const config = await loadConfigFromPath(gatewayConfigPath)

  assert.deepEqual(
    config.doors.map((door) => door.key),
    ['door-legacy-jp', 'door-legacy-hk', config.doors[2].key]
  )
  assert.equal(config.doors[0].listen_port, 58080)
  assert.equal(config.doors[1].listen_port, 58081)
  assert.equal(config.doors[2].listen_port, 58082)
  assert.equal(config.doors[2].name, 'Extra SG')
  assert.match(config.doors[2].key, /^extra-/)
})

test('buildWorkerConfig emits a single-proxy Mihomo config for one door', () => {
  const configText = buildWorkerConfig({
    name: '🇭🇰 香港W10 | IEPL',
    proxy_name: '🇭🇰 香港W10 | IEPL',
    listen_host: '127.0.0.1',
    listen_port: 58080,
    socks_port: 58180,
    controller_port: 58280,
    secret: 'door-secret',
    upstream_proxy: {
      name: '🇭🇰 香港W10 | IEPL',
      type: 'ss',
      server: 'hk10.example.com',
      port: 443,
      cipher: 'aes-256-gcm',
      password: 'secret-1'
    }
  })

  assert.match(configText, /mixed-port: 58080/)
  assert.match(configText, /socks-port: 58180/)
  assert.match(configText, /bind-address: 127\.0\.0\.1/)
  assert.match(configText, /allow-lan: false/)
  assert.match(configText, /external-controller: 127\.0\.0\.1:58280/)
  assert.match(configText, /secret: door-secret/)
  assert.match(configText, /MATCH, 🇭🇰 香港W10 \| IEPL/)
  assert.match(configText, /server: hk10\.example\.com/)
})

test('buildWorkerConfig exposes worker port while keeping controller on loopback when bind host is public', () => {
  const configText = buildWorkerConfig({
    name: 'Public Door',
    proxy_name: 'Public Door',
    listen_host: '0.0.0.0',
    controller_host: '127.0.0.1',
    listen_port: 58080,
    socks_port: 58180,
    controller_port: 58280,
    secret: 'door-secret',
    upstream_proxy: {
      name: 'Public Door',
      type: 'ss',
      server: 'public-door.example.com',
      port: 443,
      cipher: 'aes-256-gcm',
      password: 'secret-1'
    }
  })

  assert.match(configText, /bind-address: 0\.0\.0\.0/)
  assert.match(configText, /allow-lan: true/)
  assert.match(configText, /external-controller: 127\.0\.0\.1:58280/)
})
