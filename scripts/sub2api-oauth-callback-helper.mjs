#!/usr/bin/env node

import http from 'node:http'

const DEFAULT_HOST = '127.0.0.1'
const DEFAULT_PORT = 1455

function parseArgs(argv) {
  const options = {
    target: process.env.SUB2API_OAUTH_CALLBACK_TARGET || '',
    host: process.env.SUB2API_OAUTH_CALLBACK_HOST || DEFAULT_HOST,
    port: Number(process.env.SUB2API_OAUTH_CALLBACK_PORT || DEFAULT_PORT),
  }

  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i]
    if (arg === '--target') {
      options.target = argv[i + 1] || ''
      i += 1
    } else if (arg.startsWith('--target=')) {
      options.target = arg.slice('--target='.length)
    } else if (arg === '--host') {
      options.host = argv[i + 1] || ''
      i += 1
    } else if (arg.startsWith('--host=')) {
      options.host = arg.slice('--host='.length)
    } else if (arg === '--port') {
      options.port = Number(argv[i + 1])
      i += 1
    } else if (arg.startsWith('--port=')) {
      options.port = Number(arg.slice('--port='.length))
    } else if (arg === '--help' || arg === '-h') {
      printHelp()
      process.exit(0)
    } else {
      throw new Error(`Unknown argument: ${arg}`)
    }
  }

  return options
}

function printHelp() {
  console.log(`Usage:
  node scripts/sub2api-oauth-callback-helper.mjs --target http://<Linux-LAN-IP>:8080

Options:
  --target  Linux admin UI origin, for example http://192.168.1.20:8080
  --host    Local listen host, defaults to 127.0.0.1
  --port    Local listen port, defaults to 1455

Environment:
  SUB2API_OAUTH_CALLBACK_TARGET
  SUB2API_OAUTH_CALLBACK_HOST
  SUB2API_OAUTH_CALLBACK_PORT`)
}

function parseTarget(rawTarget) {
  if (!rawTarget) {
    throw new Error('--target is required')
  }
  const target = new URL(rawTarget)
  if (target.protocol !== 'http:' && target.protocol !== 'https:') {
    throw new Error('--target must start with http:// or https://')
  }
  if (target.username || target.password) {
    throw new Error('--target must not contain credentials')
  }
  return target
}

function shortState(state) {
  if (!state) return ''
  if (state.length <= 10) return state
  return `${state.slice(0, 4)}...${state.slice(-6)}`
}

function buildRedirectURL(target, sourceURL) {
  const redirectURL = new URL('/auth/callback', target.origin)
  sourceURL.searchParams.forEach((value, key) => {
    redirectURL.searchParams.append(key, value)
  })
  redirectURL.searchParams.set('admin_oauth_provider', 'openai')
  return redirectURL
}

function writeText(res, statusCode, text) {
  res.writeHead(statusCode, {
    'Content-Type': 'text/plain; charset=utf-8',
    'Cache-Control': 'no-store',
  })
  res.end(text)
}

let options
let target
try {
  options = parseArgs(process.argv.slice(2))
  target = parseTarget(options.target)
  if (!Number.isInteger(options.port) || options.port <= 0 || options.port > 65535) {
    throw new Error('--port must be an integer between 1 and 65535')
  }
  if (!options.host) {
    throw new Error('--host is required')
  }
} catch (error) {
  console.error(error.message)
  console.error('')
  printHelp()
  process.exit(1)
}

const server = http.createServer((req, res) => {
  const sourceURL = new URL(req.url || '/', `http://${options.host}:${options.port}`)
  if (sourceURL.pathname !== '/auth/callback') {
    writeText(res, 404, 'Not found')
    return
  }

  const code = sourceURL.searchParams.get('code') || ''
  const state = sourceURL.searchParams.get('state') || ''
  if (!code || !state) {
    writeText(res, 400, 'Missing code or state')
    return
  }

  const redirectURL = buildRedirectURL(target, sourceURL)
  console.log(`Received OpenAI OAuth callback; redirecting to ${redirectURL.origin}/auth/callback with state ${shortState(state)}`)
  res.writeHead(302, {
    Location: redirectURL.toString(),
    'Cache-Control': 'no-store',
  })
  res.end()
})

server.on('error', (error) => {
  console.error(`OAuth callback helper failed: ${error.message}`)
  process.exit(1)
})

server.listen(options.port, options.host, () => {
  console.log(`Sub2API OpenAI OAuth callback helper listening on http://${options.host}:${options.port}`)
  console.log(`Forwarding callbacks to ${target.origin}/auth/callback`)
})

for (const signal of ['SIGINT', 'SIGTERM']) {
  process.on(signal, () => {
    server.close(() => {
      process.exit(0)
    })
  })
}
