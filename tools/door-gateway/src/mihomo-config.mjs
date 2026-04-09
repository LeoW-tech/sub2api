import YAML from 'yaml'

export function buildWorkerConfig(door) {
  const proxyName = door.proxy_name || door.name

  const document = {
    'mixed-port': door.listen_port,
    'socks-port': door.socks_port,
    'allow-lan': false,
    mode: 'rule',
    'log-level': 'info',
    'external-controller': `${door.listen_host || '127.0.0.1'}:${door.controller_port}`,
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
