import YAML from 'yaml'

function isLoopbackHost(host) {
  return host === '127.0.0.1' || host === '::1' || host === 'localhost'
}

export function buildWorkerConfig(door) {
  const proxyName = door.proxy_name || door.name
  const listenHost = door.listen_host || '127.0.0.1'
  const controllerHost = door.controller_host || '127.0.0.1'

  const document = {
    'mixed-port': door.listen_port,
    'socks-port': door.socks_port,
    'bind-address': listenHost,
    'allow-lan': !isLoopbackHost(listenHost),
    mode: 'rule',
    'log-level': 'info',
    'external-controller': `${controllerHost}:${door.controller_port}`,
    secret: door.secret,
    'unified-delay': true,
    proxies: [
      {
        ...door.upstream_proxy,
        name: proxyName
      }
    ],
    rules: [`MATCH, ${proxyName}`]
  }

  return YAML.stringify(document)
}
