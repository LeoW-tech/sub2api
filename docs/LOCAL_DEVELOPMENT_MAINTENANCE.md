# 本地开发维护说明

这份文档是对根目录 [`常用命令.md`](../常用命令.md) 的展开说明，重点解释双机运行、守护方式和同步流程，不重复列出所有命令变体。

## 主机角色与路径矩阵

- Linux：主运行面
  - 当前仓库根目录：`/srv/sub2api/repo`
  - 当前 runtime 根目录：`/srv/sub2api/runtime`
  - 主要职责：稳定环境运行、systemd 托管、主同步提交
- Mac：辅助运行面
  - 仓库根目录：`/Users/meilinwang/Projects/sub2api`
  - 主要职责：从 GitHub 拉取更新、开发验证、必要时本机运行、launchd 自动恢复

## 目录约定

- 脚本会优先探测仓库内 `runtime/`
- 如果仓库内没有有效 runtime，则自动退回仓库同级 `../runtime/`
- 当前 Linux 实机现状：
  - 仓库根目录：`/srv/sub2api/repo`
  - runtime 根目录：`/srv/sub2api/runtime`
- 运行时数据全部收敛到 runtime 根目录
- `runtime/stable` 对应稳定环境，默认端口 `8080`
- `runtime/dev` 对应开发环境，端口 `127.0.0.1:8081`
- 默认运行时备份目录为 `<runtime-root>/backups/`，也可通过 `SUB2API_BACKUP_ROOT` 临时覆盖

默认访问约定：

- stable 同时支持本机 `http://127.0.0.1:8080/` 和局域网 `http://<本机局域网IP>:8080/`
- dev 默认仅本机访问 `http://127.0.0.1:8081/`

## Git 约定

- `origin` 指向你的 fork：`https://github.com/LeoW-tech/sub2api.git`
- `upstream` 指向原仓库：`https://github.com/Wei-Shaw/sub2api.git`
- `upstream-main` 只镜像 `upstream/main`
- `main` 是本地稳定集成分支
- 日常开发从 `main` 切 `feature/*`
- 跟上游同步时使用 `sync/upstream-YYYYMMDD`

## 统一入口

推荐统一使用 `./scripts/sub2api-local`：

```bash
./scripts/sub2api-local stable up
./scripts/sub2api-local stable down
./scripts/sub2api-local stable logs
./scripts/sub2api-local stable status
./scripts/sub2api-local stable restart
./scripts/sub2api-local stable rebuild

./scripts/sub2api-local dev up --build
./scripts/sub2api-local dev down
./scripts/sub2api-local dev logs
./scripts/sub2api-local dev status
./scripts/sub2api-local dev restart
./scripts/sub2api-local dev rebuild

./scripts/sub2api-local door restart
./scripts/sub2api-local door status

./scripts/sub2api-local systemd install
./scripts/sub2api-local systemd status
./scripts/sub2api-local systemd restart

./scripts/sub2api-local autostart install
./scripts/sub2api-local autostart uninstall
./scripts/sub2api-local autostart status
./scripts/sub2api-local autostart restart

./scripts/sub2api-local sync upstream
./scripts/sub2api-local backup runtime
```

`./scripts/sub2api-local backup runtime` 默认会把备份写到当前 runtime 根目录下的 `backups/<时间戳>/`。在 Mac 上如果检测到 `LaunchAgents` 文件，也会一并备份 `com.sub2api.autostart.plist` 与 `com.sub2api.door-gateway.plist`。

## 日常重启命令

最常用的是这几个：

```bash
# 重启稳定环境
./scripts/sub2api-local stable restart

# 查看稳定环境状态（容器、health、host.docker.internal、systemd）
./scripts/sub2api-local stable status

# 重启开发环境
./scripts/sub2api-local dev restart

# 重启 door-gateway
./scripts/sub2api-local door restart

# 重新触发登录后自动恢复链路
./scripts/sub2api-local autostart restart

# 如果改了源码并需要重建稳定环境
./scripts/sub2api-local stable rebuild

# 如果改了源码并需要重建开发环境
./scripts/sub2api-local dev rebuild
```

## 工作流

### Linux 主运行面：systemd 托管

如果你希望 Linux 在开机后自动恢复 `stable + door-gateway`，并让 `door-gateway` 在异常退出后自动拉起，使用：

```bash
sudo ./scripts/sub2api-local systemd install
```

安装动作会：

- 渲染并安装 `/etc/systemd/system/sub2api-stable.service`
- 渲染并安装 `/etc/systemd/system/sub2api-door-gateway.service`
- 让 stable 栈统一通过仓库内 `scripts/sub2api-runtime-compose` 启动
- 把当前 runtime 根目录显式写入 systemd 环境，避免仓库内外 runtime 路径漂移
- 在 Linux 自动追加 `deploy/local/docker-compose.runtime.linux.yml`
- 为 `sub2api` 容器注入 `host.docker.internal:host-gateway`
- 先恢复 stable 栈，再按顺序拉起 `door-gateway`

查看 Linux 守护状态：

```bash
./scripts/sub2api-local systemd status
./scripts/sub2api-local stable status
```

重启 Linux 守护链路：

```bash
sudo ./scripts/sub2api-local systemd restart
```

`stable status` 会同时输出：

- 当前平台使用的 compose 文件
- `sub2api/postgres/redis` 容器状态
- `sub2api` 与 `door-gateway` 的 health 结果
- 容器内 `host.docker.internal` 的解析结果
- Linux 上的 `sub2api-stable.service` / `sub2api-door-gateway.service` 状态

预期恢复链路：

- 宿主机重启后，systemd 先执行 `sub2api-stable.service`，再启动 `sub2api-door-gateway.service`
- Docker 服务恢复后，可执行 `sudo ./scripts/sub2api-local systemd restart` 重新串起 stable 和 `door-gateway`
- `door-gateway` 进程异常退出后，systemd 会按 `Restart=always` 自动拉起
- `door-gateway` 内部如果只出现单个 worker 监听口失效，健康轮询会自动重启该 worker
- 容器本身异常退出后，由 compose 中的 `restart: unless-stopped` 自动恢复

### Mac 辅助运行面：autostart / launchd 自动恢复

如果你希望 macOS 在登录后自动恢复 `stable + door-gateway`，使用：

```bash
./scripts/sub2api-local autostart install
```

安装动作会：

- 校验 `colima`、`docker`、`node` 是否可用
- 生成 `~/Library/LaunchAgents/com.sub2api.autostart.plist`
- 生成 `~/Library/LaunchAgents/com.sub2api.door-gateway.plist`
- 启动主协调器，自动恢复 `stable` 与 `door-gateway`
- 校验 `http://127.0.0.1:8080/health` 与 `http://127.0.0.1:19080/health`

当前默认 macOS 容器运行时为 Colima；`autostart` 会在登录后先恢复 Colima，再恢复 stable 栈和 `door-gateway`。

查看当前状态：

```bash
./scripts/sub2api-local autostart status
```

如果更换了 Node 路径、Homebrew 路径或希望重新生成 plist，可执行：

```bash
./scripts/sub2api-local autostart install
./scripts/sub2api-local autostart restart
```

如果只想取消“登录后自动恢复”，但不主动停止当前运行中的服务，可执行：

```bash
./scripts/sub2api-local autostart uninstall
```

### 开发自己的功能

```bash
git checkout main
git pull --ff-only origin main
git checkout -b feature/<topic>
./scripts/sub2api-local dev rebuild
./scripts/sub2api-local dev up
```

功能验证通过后：

```bash
git add .
git commit -m "feat: <topic>"
git push -u origin feature/<topic>
```

### 双机同步到 GitHub

日常同步建议遵循这个方向：

1. Linux 主运行面完成改动并提交。
2. 需要共享给另一台机器时，由 Linux 推送到 `origin`。
3. Mac 辅助运行面从 `origin` 拉取更新，再做验证或本机重启。

Linux 提交并按需推送：

```bash
git add .
git commit -m "<type>: <topic>"
git push origin <branch>
```

Mac 拉取同步：

```bash
git fetch origin --prune
git checkout main
git pull --ff-only origin main
```

Mac 拉取后常见验证：

```bash
./scripts/sub2api-local stable status
./scripts/sub2api-local autostart status
./scripts/sub2api-local door status
```

### 同步原仓库更新

```bash
git checkout main
./scripts/sub2api-local sync upstream
```

如果自动 merge 成功，会停在新的 `sync/upstream-YYYYMMDD` 分支上。完成验证后再把该分支合回 `main`，然后：

```bash
git checkout main
./scripts/sub2api-local stable rebuild
./scripts/sub2api-local stable up
```

如果这次更新还要同步到 Mac，就继续执行上面的 GitHub 中转流程。

## 运行时资产

- 稳定环境历史数据：`runtime/stable/data`
- 稳定环境数据库：`runtime/stable/postgres_data`
- 稳定环境 Redis：`runtime/stable/redis_data`
- `door-gateway` 配置：`runtime/stable/door-gateway.json`
- Linux 专用 compose override：`deploy/local/docker-compose.runtime.linux.yml`
- `door-gateway` worker 目录：`runtime/stable/door-workers`
- 运行时备份目录：`runtime/backups`
- Linux systemd 模板：`deploy/local/systemd/*.service.template`
- Mac 当前用户 `LaunchAgent`：`~/Library/LaunchAgents/com.sub2api.autostart.plist`、`~/Library/LaunchAgents/com.sub2api.door-gateway.plist`

这些目录全部不进 git。
