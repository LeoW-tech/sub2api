# 本地开发维护说明

## 目录约定

- 仓库根目录固定为 `/Users/meilinwang/Projects/sub2api`
- 运行时数据全部收敛到 `runtime/`
- `runtime/stable` 对应稳定环境，默认端口 `8080`
- `runtime/dev` 对应开发环境，端口 `127.0.0.1:8081`
- 默认运行时备份目录为 `runtime/backups/`，也可通过 `SUB2API_BACKUP_ROOT` 临时覆盖

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
./scripts/sub2api-local stable restart
./scripts/sub2api-local stable rebuild

./scripts/sub2api-local dev up --build
./scripts/sub2api-local dev down
./scripts/sub2api-local dev logs
./scripts/sub2api-local dev restart
./scripts/sub2api-local dev rebuild

./scripts/sub2api-local door restart
./scripts/sub2api-local door status

./scripts/sub2api-local sync upstream
./scripts/sub2api-local backup runtime
```

`./scripts/sub2api-local backup runtime` 默认会把备份写到 `runtime/backups/<时间戳>/`。

## 日常重启命令

最常用的是这几个：

```bash
# 重启稳定环境
./scripts/sub2api-local stable restart

# 重启开发环境
./scripts/sub2api-local dev restart

# 重启 door-gateway
./scripts/sub2api-local door restart

# 如果改了源码并需要重建稳定环境
./scripts/sub2api-local stable rebuild

# 如果改了源码并需要重建开发环境
./scripts/sub2api-local dev rebuild
```

## 工作流

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

## 运行时资产

- 稳定环境历史数据：`runtime/stable/data`
- 稳定环境数据库：`runtime/stable/postgres_data`
- 稳定环境 Redis：`runtime/stable/redis_data`
- `door-gateway` 配置：`runtime/stable/door-gateway.json`
- `door-gateway` worker 目录：`runtime/stable/door-workers`
- 运行时备份目录：`runtime/backups`

这些目录全部不进 git。
