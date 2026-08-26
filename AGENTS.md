# AGENTS.md

请始终用中文与用户交流；每次完成代码、脚本、配置或文档变更后，及时提交本地 Git。

## 双机分工

- Linux `192.168.31.214` 只用 `/srv/sub2api/primary` 作为主工作区，合并、构建、全量检查都在这里做。
- Tokyo `43.165.188.95`（`tokyo-vps`）只做 `git fetch`、`git merge --ff-only` 和经授权的部署，准备阶段不改运行状态。
- Linux 旧目录 `/srv/sub2api/repo` 只作历史副本，未确认新工作区前不清理、不覆盖。
- 生产密钥、runtime、数据库和 Redis 数据不进入 Git 或镜像。

## 合并规则

- 任务指定 release/tag 时，用真实提交和提交图确认来源，不靠 HEAD 文本版号猜。
- 当前 `main` 是带本地定制的集成分支，不能为了追上游而重置或丢弃定制功能。
- 同一能力优先保留上游实现，不同能力两边都保留；冲突按功能、调用链、配置、迁移、接口逐项处理，无法确认就停下来报告，禁止批量 `ours`/`theirs`。
- 已发布迁移不可改名或修改；新迁移用 `900001+` 区间，并按项目工具检查编号。

## Linux 检查

- 全量 Go 测试、golangci-lint、TypeScript、Vitest、迁移/脚本/生成代码检查只在 Linux 主工作区执行，Tokyo 不跑这些重任务。
- 重任务必须串行；单个工具内可适度并发：`GOMAXPROCS=8 go test -p 8 -parallel 8 ./...`、`golangci-lint run --concurrency 8`、`pnpm test:run -- --maxWorkers=8`。
- 工具链按项目要求使用 Go `1.27.0`、Node `24`、pnpm `9.15.9`、golangci-lint `2.13.0`。
- 每次重任务都记录时间、执行者、提交 SHA、命令、并发参数、工具版本、资源快照、日志路径和退出码；资源异常、swap 持续增长或无关容器持续重启时暂停。
- 资源边界以“别把机器打满”为准，不新增 cgroup、systemd 限额或代码级限制。

## Docker 与镜像

- Linux 使用当前 Tokyo 项目的 Dockerfile 和 Compose 文件，核对架构、服务、健康检查、端口、网络、卷和 Postgres `18-alpine` / Redis `8-alpine` 依赖。
- 应用镜像使用提交 SHA 标签，并写入 `org.opencontainers.image.revision=<commit-sha>`；不要用含糊的 `stable` 标签作为交付产物。
- 准备阶段不加载镜像、不改生产 `.env`、不执行 Compose 或 systemd 重启。

## Tokyo 拉取

Linux 检查和镜像构建成功后，Tokyo 只执行：

```bash
cd /srv/sub2api/repo
git fetch origin main
git merge --ff-only origin/main
git status --short
```

禁止在 Tokyo 执行 `docker compose build/up/restart`、`systemctl restart`、生产环境变量修改或测试任务。

## 交付

- 远端 `main`、Linux 主工作区和 Tokyo 仓库最终指向同一提交。
- 全量检查、静态检查和独立复查全部通过后再交付。
- 交付说明必须列出实际命令、顺序、结果和未执行项。
