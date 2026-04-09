# AGENTS.md

请始终用中文与用户交流。

每次完成代码、脚本、配置或文档改动后，请及时提交本地 git。除非用户明确要求，否则不要自动推送远端。

## 项目现状

这是一个已经完成本地单仓库整合的 Sub2API 项目。

- 唯一仓库根目录：`/Users/meilinwang/Projects/sub2api`
- 你的 fork：`origin = https://github.com/LeoW-tech/sub2api.git`
- 原始仓库：`upstream = https://github.com/Wei-Shaw/sub2api.git`
- 稳定集成分支：`main`
- 上游镜像分支：`upstream-main`

本仓库不是“纯上游镜像”，而是：

1. 跟踪原始仓库最新 `upstream/main`
2. 在本地 `main` 上叠加用户自己的定制功能
3. 使用双环境运行，避免开发环境影响稳定环境

## 目录约定

必须遵守下面的目录边界：

- 源码、脚本、文档、部署模板都在仓库内
- 所有运行时数据都在 `runtime/`
- 严禁把运行时数据重新放回仓库根目录

当前运行时结构：

```text
runtime/
  stable/
    .env
    data/
    postgres_data/
    redis_data/
    door-gateway.json
    door-workers/
  dev/
    .env
    data/
    postgres_data/
    redis_data/
  backups/
    <timestamp>/
      runtime/
      com.sub2api.door-gateway.plist
```

说明：

- `runtime/stable` 是稳定环境，默认服务端口 `8080`
- `runtime/dev` 是开发环境，服务端口 `127.0.0.1:8081`
- `runtime/backups` 是默认运行时备份目录
- `door-gateway` 配置在 `runtime/stable/door-gateway.json`
- `door-gateway` worker 数据在 `runtime/stable/door-workers/`
- `runtime/` 整体不进 git

前端访问地址：

- 稳定环境前端（本机）：`http://127.0.0.1:8080/`
- 稳定环境前端（局域网）：`http://<本机局域网IP>:8080/`
- 开发环境前端：`http://127.0.0.1:8081/`
- `door-gateway` 健康检查：`http://127.0.0.1:19080/health`

## 分支与维护模式

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

## 统一运维入口

优先使用统一脚本：

- `./scripts/sub2api-local`

常用命令：

```bash
# stable: 启动稳定环境
./scripts/sub2api-local stable up
# stable: 停止稳定环境
./scripts/sub2api-local stable down
# stable: 查看稳定环境日志
./scripts/sub2api-local stable logs
# stable: 重启稳定环境
./scripts/sub2api-local stable restart
# stable: 重建并启动稳定环境
./scripts/sub2api-local stable rebuild

# dev: 启动开发环境并在需要时构建
./scripts/sub2api-local dev up --build
# dev: 停止开发环境
./scripts/sub2api-local dev down
# dev: 查看开发环境日志
./scripts/sub2api-local dev logs
# dev: 重启开发环境
./scripts/sub2api-local dev restart
# dev: 重建并启动开发环境
./scripts/sub2api-local dev rebuild

# door: 重启 door-gateway 宿主机服务
./scripts/sub2api-local door restart
# door: 查看 door-gateway 当前 launchctl 状态
./scripts/sub2api-local door status

# maintenance: 同步原始仓库最新 main 到本地同步分支
./scripts/sub2api-local sync upstream
# maintenance: 备份 runtime 运行时数据
./scripts/sub2api-local backup runtime
```

`./scripts/sub2api-local backup runtime` 默认会把备份写入 `runtime/backups/<timestamp>/`；如有需要，可通过 `SUB2API_BACKUP_ROOT` 临时覆盖。

也可以使用 Makefile 别名：

```bash
# stable: 启动稳定环境
make stable-up
# stable: 停止稳定环境
make stable-down
# stable: 查看稳定环境日志
make stable-logs
# stable: 重启稳定环境
make stable-restart
# stable: 重建并启动稳定环境
make stable-rebuild

# dev: 启动开发环境并构建
make dev-up
# dev: 停止开发环境
make dev-down
# dev: 查看开发环境日志
make dev-logs
# dev: 重启开发环境
make dev-restart
# dev: 重建并启动开发环境
make dev-rebuild

# door: 重启 door-gateway
make door-restart
# door: 查看 door-gateway 状态
make door-status

# maintenance: 同步上游
make sync-upstream
# maintenance: 备份 runtime
make backup-runtime
```

## 日常开发流程

开发用户自己的功能时：

```bash
git checkout main
git pull --ff-only origin main
git checkout -b feature/<topic>
./scripts/sub2api-local dev rebuild
./scripts/sub2api-local dev up
```

完成后：

```bash
git add .
git commit -m "feat: <topic>"
git push -u origin feature/<topic>
```

如果用户没有要求推送，只提交本地即可。

## 同步原始仓库流程

同步上游时不要自己重新发明流程，直接使用：

```bash
./scripts/sub2api-local sync upstream
```

它会做这些事：

1. `fetch upstream` 和 `fetch origin`
2. 更新 `upstream-main`
3. 推送 `origin/upstream-main`
4. 从 `main` 切出新的 `sync/upstream-YYYYMMDD`
5. 把 `upstream-main` 合入这个同步分支

同步后应继续：

```bash
./scripts/sub2api-local dev rebuild
./scripts/sub2api-local dev up
```

验证无误后，再把同步分支合回 `main`。

## 重要约束

- 不要把 `runtime/` 下的文件加入 git
- 不要删除或覆盖用户的运行时数据，除非用户明确要求
- 修改稳定环境相关内容时，优先保证 `stable` 可恢复
- 修改 `door-gateway` 时，要同时考虑 LaunchAgent 路径、配置路径和日志路径
- 如果调整脚本接口，必须同步更新 `docs/LOCAL_DEVELOPMENT_MAINTENANCE.md`

## 完成前检查

在声称完成之前，至少确认：

1. `git status` 是否干净或是否只剩预期改动
2. 如涉及 stable/dev 运行面，相关服务是否真的可访问
3. 如涉及 `door-gateway`，`http://127.0.0.1:19080/health` 是否正常
4. 变更是否已经提交本地 git
