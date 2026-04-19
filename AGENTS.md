# AGENTS.md

请始终用中文与用户交流。

每次完成代码、脚本、配置或文档改动后，请及时提交本地 git。除非用户明确要求，否则不要自动推送远端。

## 项目现状

这是一个已经完成 Linux 正式迁移的 Sub2API 项目。

- Linux 正式仓库根目录：`/srv/sub2api/repo`
- Linux 正式运行时根目录：`/srv/sub2api/runtime`
- Linux 正式 stable 运行时：`/srv/sub2api/runtime/stable`
- 对外正式域名：`https://api.cloudalpha.top/`
- 当前正式运行方式：
  - `sub2api` 通过 Docker Compose 运行
  - PostgreSQL / Redis 使用项目自己的容器
  - `door-gateway` 作为 Linux 宿主机 `systemd` 服务运行
  - `cloudflared` 作为 Linux 宿主机 `systemd` 服务运行

## 分类导航

当前仓库内的信息按下面方式分流：

- 项目事实与协作约束：保留在本文件
- 常用命令、运维入口、重启/重建方式：移至根目录 [`常用命令.md`](./常用命令.md)
- 本地运维细节说明：见 [`docs/LOCAL_DEVELOPMENT_MAINTENANCE.md`](./docs/LOCAL_DEVELOPMENT_MAINTENANCE.md)

## 目录约定

必须遵守下面的目录边界：

- 源码、脚本、文档、部署模板都在仓库内
- 所有正式运行时数据都在 `/srv/sub2api/runtime/`
- 严禁把运行时数据重新放回仓库根目录

当前正式运行时结构：

```text
/srv/sub2api/
  repo/
  runtime/
    stable/
      .env
      data/
      postgres_data/
      redis_data/
      door-gateway.json
      door-source-recovered.yaml
      door-workers/
  backups/
```

说明：

- `runtime/stable` 是 Linux 正式环境，默认服务端口 `8080`
- 当前正式环境已开放局域网访问，`BIND_HOST=0.0.0.0`
- `door-gateway` 配置在 `runtime/stable/door-gateway.json`
- `door-gateway` worker 数据在 `runtime/stable/door-workers/`
- `runtime/` 整体不进 git

前端访问地址：

- 稳定环境前端（本机）：`http://127.0.0.1:8080/`
- 稳定环境前端（局域网）：`http://<本机局域网IP>:8080/`
- 稳定环境前端（公网 / Cloudflare Tunnel）：`https://api.cloudalpha.top/`
- `door-gateway` 健康检查：`http://127.0.0.1:19080/health`
- Docker 内的 Sub2API 访问 Linux 宿主机门控时，走 `http://host.docker.internal:19080/`

## Git 与分支

默认分支和用途如下：

- `main`
  用于稳定集成，只部署用户确认可保留的功能
- `upstream-main`
  只镜像 `upstream/main`，禁止直接开发
- `feature/*`
  日常功能开发分支，从 `main` 切出
- `sync/upstream-YYYYMMDD`
  同步上游时的临时分支，从 `main` 切出并合入 `upstream-main`

工作规则：

- 不要在 `upstream-main` 上开发
- 尽量不要直接在 `main` 上做功能开发
- 新功能优先从 `main` 切 `feature/*`
- 跟上游同步时，使用统一脚本，不要手动设计新的同步流程

## 重要约束

- 不要把 `runtime/` 下的文件加入 git
- 不要删除或覆盖用户的运行时数据，除非用户明确要求
- 修改 stable 相关内容时，优先保证正式环境可恢复
- 修改 Linux 正式环境的 `door-gateway` 时，要同时考虑：
  - `systemd`
  - 配置路径
  - worker 目录
  - 监听地址
  - 导出给 Docker 的主机名
  - 日志路径
- 不要擅自改动固定密钥、生产数据或 Cloudflare Tunnel 名称
- 不要默认复用宿主机 PostgreSQL / Redis

## 正式运行面

当前 Linux 正式环境的守护方式：

- `sub2api-stable.service`：负责正式 Docker Compose 运行面
- `sub2api-door-gateway.service`：负责宿主机门控
- `cloudflared.service`：负责公网 Tunnel

当前 Linux 正式环境的运行约束：

- `sub2api` 对外端口为 `8080`
- 当前允许本机和局域网直接访问 `8080`
- PostgreSQL / Redis 不映射宿主机 `5432/6379`
- 公网入口继续走 Cloudflare Tunnel，不走端口转发

## 完成前检查

在声称完成之前，至少确认：

1. `git status` 是否干净或是否只剩预期改动
2. 如涉及 stable 运行面，`http://127.0.0.1:8080/health` 是否正常
3. 如涉及公网入口，`https://api.cloudalpha.top/health` 是否正常
4. 如涉及 `door-gateway`，`http://127.0.0.1:19080/health` 是否正常
5. 变更是否已经提交本地 git
