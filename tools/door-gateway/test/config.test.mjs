import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'
import assert from 'node:assert/strict'

import { loadConfigFromPath } from '../src/config.mjs'
import { buildWorkerConfig } from '../src/mihomo-config.mjs'

test('loadConfigFromPath resolves Clash proxy definitions into door workers', () => {
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

  const config = loadConfigFromPath(gatewayConfigPath)

  assert.equal(config.doors.length, 2)
  assert.equal(config.doors[0].listen_port, 58080)
  assert.equal(config.doors[0].socks_port, 58180)
  assert.equal(config.doors[0].controller_port, 58280)
  assert.equal(config.doors[0].upstream_proxy.server, 'hk10.example.com')
  assert.equal(config.doors[1].listen_port, 58081)
  assert.equal(config.doors[1].socks_port, 58181)
  assert.equal(config.doors[1].controller_port, 58281)
  assert.equal(config.doors[1].upstream_proxy.type, 'vmess')
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
  assert.match(configText, /external-controller: 127\.0\.0\.1:58280/)
  assert.match(configText, /secret: door-secret/)
  assert.match(configText, /MATCH, 🇭🇰 香港W10 \| IEPL/)
  assert.match(configText, /server: hk10\.example\.com/)
})
