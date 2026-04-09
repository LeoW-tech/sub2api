# Door Gateway

`door-gateway` 是一个给 Sub2API 准备的宿主机侧“小门控服务”。

它做三件事：

1. 从你现有的 Clash/Mihomo 配置里选出指定节点，为每扇门生成一个独立 Mihomo worker。
2. 提供统一的管理接口，方便查看 10-20 扇门是否在线。
3. 导出成 Sub2API `账号/代理导入` 能直接消费的 JSON。

## 为什么它不会影响你日常使用 Clash Pro

- `door-gateway` 不接管系统代理
- 不修改 Clash Pro 现有配置
- 只监听你显式配置的那些端口

所以：

- 平时浏览网页，继续走你原来的 Clash Pro 通用门
- Sub2API 专属账号，走 `door-gateway` 暴露出来的专属门

## 配置

复制 `doors.example.json` 为你自己的配置文件，例如：

```bash
cp tools/door-gateway/doors.example.json ~/door-gateway.json
```

示例配置：

```json
{
  "api": {
    "host": "127.0.0.1",
    "port": 19080
  },
  "mihomo_binary": "/Applications/Clash Verge.app/Contents/MacOS/verge-mihomo",
  "source_config_path": "/Users/meilinwang/.config/clash/vv148.no-mad-world.club.yaml",
  "worker_base_dir": "/Users/meilinwang/Projects/sub2api/door-workers",
  "worker_port_start": 58080,
  "worker_socks_port_start": 59080,
  "controller_port_start": 60080,
  "sub2api_export_host": "host.docker.internal",
  "healthcheck_interval_ms": 30000,
  "export_protocol": "http",
  "doors": [
    {
      "key": "door-hk-w10",
      "name": "🇭🇰 香港W10 | IEPL",
      "proxy_name": "🇭🇰 香港W10 | IEPL",
      "exit_ip": "203.0.113.10"
    }
  ]
}
```

字段说明：

- `key`: 门的稳定唯一标识，供 Sub2API 导入/绑定使用
- `name`: 你希望在 Sub2API 里看到的名字
- `proxy_name`: 这扇门要绑定的 Clash/Mihomo 节点名，建议直接沿用原配置里的名字
- `mihomo_binary`: 本机 Mihomo 内核路径
- `source_config_path`: 现有 Clash/Mihomo 配置文件路径
- `worker_base_dir`: `door-gateway` 自动生成 worker 配置和日志的目录
- `worker_port_start`: 门的 `mixed-port` 起始端口
- `worker_socks_port_start`: 门的 `socks-port` 起始端口
- `controller_port_start`: 门的管理控制端口起始值
- `export_protocol`: 导出给 Sub2API 时使用的协议，默认推荐 `http`
- `exit_ip`: 可选。你已有上游系统能提供时可直接带上
- `sub2api_export_host`: 导出给 Docker 里的 Sub2API 时使用的宿主机地址，默认建议 `host.docker.internal`

启动后，每扇门会拥有自己独立的 worker 目录、监听端口、控制端口和日志，不会去改你日常使用的 Clash Pro 通用门。

## 启动

```bash
DOOR_GATEWAY_CONFIG=~/door-gateway.json node tools/door-gateway/src/server.mjs
```

或：

```bash
cd tools/door-gateway
DOOR_GATEWAY_CONFIG=~/door-gateway.json npm start
```

## 管理接口

- `GET /health`
- `GET /doors`
- `GET /doors/:key`
- `GET /export/sub2api`

### 导出给 Sub2API

`GET /export/sub2api` 会返回一个 `sub2api-data` 结构，里面的 `proxies` 可以直接导入 Sub2API：

```bash
curl http://127.0.0.1:19080/export/sub2api
```

返回的每条代理会包含：

- `proxy_external_key`
- `name`
- `protocol`
- `host`
- `port`
- `exit_ip`（如果配置了）

## 测试

```bash
cd tools/door-gateway
npm test
```
