import http from 'node:http'
import { loadConfig } from './config.mjs'
import { DoorRuntime } from './door-runtime.mjs'

function sendJSON(res, statusCode, payload) {
  res.writeHead(statusCode, { 'content-type': 'application/json; charset=utf-8' })
  res.end(JSON.stringify(payload, null, 2))
}

const config = await loadConfig()
const runtime = new DoorRuntime(config)

await runtime.start()

const server = http.createServer((req, res) => {
  const url = new URL(req.url || '/', `http://${req.headers.host || '127.0.0.1'}`)

  if (req.method === 'GET' && url.pathname === '/health') {
    const doors = runtime.snapshot()
    const online = doors.filter((door) => door.state.online).length
    return sendJSON(res, 200, {
      ok: online === doors.length,
      config_path: config.configPath,
      total_doors: doors.length,
      online_doors: online
    })
  }

  if (req.method === 'GET' && url.pathname === '/doors') {
    return sendJSON(res, 200, runtime.snapshot())
  }

  if (req.method === 'GET' && url.pathname.startsWith('/doors/')) {
    const key = decodeURIComponent(url.pathname.slice('/doors/'.length))
    const door = runtime.snapshot().find((item) => item.key === key)
    if (!door) {
      return sendJSON(res, 404, { error: 'door not found', key })
    }
    return sendJSON(res, 200, door)
  }

  if (req.method === 'GET' && url.pathname === '/export/sub2api') {
    return sendJSON(res, 200, runtime.buildExportPayload())
  }

  return sendJSON(res, 404, { error: 'not found' })
})

server.listen(config.api.port, config.api.host, () => {
  console.log(
    `[door-gateway] api listening on http://${config.api.host}:${config.api.port} (config: ${config.configPath})`
  )
})

const shutdown = async () => {
  server.close()
  await runtime.stop()
  process.exit(0)
}

process.on('SIGINT', shutdown)
process.on('SIGTERM', shutdown)
