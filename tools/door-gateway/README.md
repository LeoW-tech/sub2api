# Door Gateway

`door-gateway` 是一个给 Sub2API 准备的宿主机侧“小门控服务”。

它做三件事：

1. 从一条或多条 Clash/Mihomo 订阅里收集节点，为每扇门生成一个独立 Mihomo worker。
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
  "worker_base_dir": "/Users/meilinwang/Projects/sub2api/door-workers",
  "worker_bind_host": "127.0.0.1",
  "controller_bind_host": "127.0.0.1",
  "worker_port_start": 58080,
  "worker_socks_port_start": 59080,
  "controller_port_start": 60080,
  "sub2api_export_host": "host.docker.internal",
  "healthcheck_interval_ms": 30000,
  "export_protocol": "http",
  "sources": [
    {
      "name": "nomad",
      "url": "https://do02n.no-mad-sub.one/link/your-token?clash=3&extend=1"
    },
    {
      "name": "trojanflare",
      "url": "https://s1.trojanflare.one/clashx/your-token"
    }
  ]
}
```

字段说明：

- `sources`: 订阅源列表。每个 source 至少包含：
  - `name`: 内部唯一别名，会参与生成稳定 `door.key`
  - `url`: 远端 Clash/Mihomo 订阅地址
  - `path`: 本地 YAML 路径。适合测试或本机已落盘配置；`path` 与 `url` 二选一
  - `enabled`: 可选，默认 `true`
- `mihomo_binary`: 本机 Mihomo 内核路径
- `worker_base_dir`: `door-gateway` 自动生成 worker 配置和日志的目录
- `worker_bind_host`: Mihomo worker 监听地址，默认 `127.0.0.1`
- `controller_bind_host`: Mihomo controller 监听地址，默认 `127.0.0.1`
- `worker_port_start`: 门的 `mixed-port` 起始端口
- `worker_socks_port_start`: 门的 `socks-port` 起始端口
- `controller_port_start`: 门的管理控制端口起始值
- `export_protocol`: 导出给 Sub2API 时使用的协议，默认推荐 `http`
- `sub2api_export_host`: 导出给 Docker 里的 Sub2API 时使用的宿主机地址，默认建议 `host.docker.internal`

监听建议：

- 仅宿主机本地使用时，保持默认 `worker_bind_host=127.0.0.1`
- 如果 Sub2API 运行在 Docker 中，且 `sub2api_export_host=host.docker.internal`，必须同时设置 `worker_bind_host=0.0.0.0`
- 出于安全考虑，通常应继续保持 `controller_bind_host=127.0.0.1`

`door-gateway` 会按 `source.name + 节点指纹(type/server/port/name)` 自动生成稳定 `door.key`，所以：

- 同一订阅里节点内容不变时，`door.key` 保持稳定
- 两条订阅里就算出现同名节点，也不会互相覆盖
- 某个 source 被移除时，只会让这一组对应 doors 从 `/doors` 快照里消失

启动后，每扇门会拥有自己独立的 worker 目录、监听端口、控制端口和日志，不会去改你日常使用的 Clash Pro 通用门。

## 向后兼容

旧配置里的 `source_config_path + doors[]` 仍然可用：

- 适合你只想从单个本地 YAML 里手工挑几个节点出门
- 新的 `sources[]` 模式更适合“把多条订阅整体并入现有门池”

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
