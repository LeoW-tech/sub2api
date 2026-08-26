# AGENTS.md

请始终用中文与用户交流；每次完成代码、脚本、配置或文档变更后，及时提交本地 Git。

## 双机分工

- Linux `192.168.31.214` 只用 `/srv/sub2api/primary` 作为主工作区，合并、构建、全量检查和镜像制作都在这里完成。
- Tokyo `43.165.188.95`（`tokyo-vps`）只做 `git fetch`、`git merge --ff-only` 和经授权的部署；准备阶段不改 stable 服务、容器、镜像、配置或数据。
- Linux 旧目录 `/srv/sub2api/repo` 只作历史副本，未确认新工作区前不清理、不覆盖。
- `origin=https://github.com/LeoW-tech/sub2api.git`，`upstream=https://github.com/Wei-Shaw/sub2api.git`，稳定分支为 `main`。生产密钥、runtime、数据库和 Redis 数据不进入 Git 或镜像。

## 合并规则

- 任务指定 release/tag 时，用真实提交、`git merge-base --is-ancestor`、提交图和差异确认来源，不靠 HEAD 文本版号猜测。
- 当前 `main` 是带本地定制的集成分支，不能为了追上游而重置或丢弃定制功能。
- 同一能力优先保留上游实现，不同能力两边都保留；冲突按功能、调用链、配置、迁移、接口逐项处理，无法确认就停下来报告，禁止批量 `ours`/`theirs`。
- 已发布迁移不可改名或修改；新迁移用 `900001+` 区间，并按项目工具检查编号。生成代码先修复生成源，再按项目工具重新生成。
- 推送只使用可审计的 fast-forward，禁止 force push、destructive reset 和未经审查的整文件覆盖。

## Linux 检查

- 全量 Go 测试、golangci-lint、TypeScript、Vitest、迁移/脚本/生成代码检查只在 Linux 主工作区执行，Tokyo 不跑这些重任务。
- 重任务必须串行，不使用后台、`nohup`、`tmux`、`screen` 或重叠 SSH 会话。单个工具内可适度并发：
  `GOMAXPROCS=8 go test -p 8 -parallel 8 ./...`；
  `golangci-lint run --concurrency 8`；
  `pnpm typecheck`；
  `pnpm test:run -- --maxWorkers=8`。
- 工具链按项目要求使用 Go `1.27.0`、Node `24`、pnpm `9.15.9`、golangci-lint `2.13.0`，不得使用 Linux 历史副本的旧工具或依赖。
- 启动重任务前检查可用内存、swap、CPU/I/O 和 Docker 健康状态；每次记录时间、执行者、提交 SHA、命令、并发参数、工具版本、资源快照、日志路径和退出码。目标资源约 60%-80%；资源异常、swap 持续增长或无关容器持续重启时暂停，不停止无关服务、不宽泛杀进程、不重启整机。
- 任一全量检查未通过或未完成，不能进入交付。

## Docker 与镜像

- Linux 使用当前 Tokyo 项目的 Dockerfile 和 Compose 文件，核对架构、服务、健康检查、端口、网络、卷和 Postgres `18-alpine` / Redis `8-alpine` 依赖。应用代码变化后镜像摘要可以不同，但运行条件必须一致。
- 应用镜像使用提交 SHA 标签，并写入 `org.opencontainers.image.revision=<commit-sha>`；不要用含糊的 `stable` 标签作为交付产物。
- 不使用公共或付费镜像仓库。获得单独部署授权后，Linux 用 `docker save` 压缩包经 SSH 传到 Tokyo，校验 SHA256 后再加载。
- 准备阶段不加载镜像、不改生产 `.env`、不执行 Compose 或 systemd 重启，也不新增 cgroup、systemd 限额或代码级限制。

## Tokyo 拉取

Linux 检查和镜像构建成功后，Tokyo 只执行：

```bash
cd /srv/sub2api/repo
git fetch origin main
git merge --ff-only origin/main
git status --short
```

禁止在 Tokyo 执行 `docker compose build/up/restart`、`systemctl restart`、生产环境变量修改或测试任务。拉取代码不等于部署；stable 容器、镜像、数据和服务状态必须保持原样。

## 交付

- 远端 `main`、Linux 主工作区和 Tokyo 仓库最终指向同一提交，且提交图能证明包含指定上游基准。
- 全量检查、迁移升级验证、配置/脚本/生成代码静态检查和独立复查全部通过。
- 镜像 revision 标签、摘要、构建日志和检查日志可追溯；没有生产密钥、runtime 或数据库数据进入提交或镜像。
- 交付说明必须列出实际命令、顺序、结果和未执行项；未授权部署、重启和旧目录删除均不得暗中执行。
