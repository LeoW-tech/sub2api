# AGENTS.md

请始终用中文与用户交流。

## 双机职责与目录

- Linux `192.168.31.214` 是唯一的主工作区：`/srv/sub2api/primary`。合并、构建、全量检查和镜像制作都在这里完成。
- Tokyo `43.165.188.95`（`tokyo-vps`）只负责拉取已验证的代码和后续经授权的部署。当前运行目录为 `/srv/sub2api/repo`，stable 服务和生产 runtime 不得在准备阶段改变。
- Linux 旧目录 `/srv/sub2api/repo` 是历史副本，不是来源，不在其中开发或验证；迁移成功后再单独清理。
- `origin=https://github.com/LeoW-tech/sub2api.git`，`upstream=https://github.com/Wei-Shaw/sub2api.git`，稳定分支为 `main`。生产密钥、runtime、数据库和 Redis 数据永不进入 Git。

## 版本与上游合并

- 任务指定 release/tag 时，以其实际提交作为上游基准；用 `git merge-base --is-ancestor`、提交图和差异确认来源，不以 HEAD 标题或应用显示版号猜测。
- 当前 `main` 是带本地定制的集成分支，不得为了追上游而重置或丢弃定制功能。
- 同一能力以上游实现为基准，不同能力两边保留；冲突必须逐项按功能、调用链、配置、迁移、接口和界面处理。无法确认时停止并报告，禁止批量使用 `ours`/`theirs`。
- 已发布迁移不可改名或修改；本地新迁移使用 `900001+` 保留区间，并运行项目迁移编号检查。生成代码先修复生成源，再按项目工具重新生成。
- 每次代码、脚本、配置或文档变更完成后立即提交本地 Git；推送只使用可审计的 fast-forward，禁止 force push 和 destructive reset。

## Linux 全量验证

- 全量 Go 测试、golangci-lint、TypeScript、Vitest、迁移/脚本/生成代码检查只在 Linux 主工作区执行；Tokyo 禁止执行这些重任务。
- 重任务严格一个接一个，不使用后台、`nohup`、`tmux`、`screen` 或重叠 SSH 会话。单个工具内部允许适度并发：
  `GOMAXPROCS=8 go test -p 8 -parallel 8 ./...`；
  `golangci-lint run --concurrency 8`；
  `pnpm typecheck`；
  `pnpm test:run -- --maxWorkers=8`。
- 工具链按当前项目要求使用 Go `1.27.0`、Node `24`、pnpm `9.15.9`、golangci-lint `2.13.0`，不得使用 Linux 历史副本的旧工具。
- 启动重任务前记录时间、执行者、提交 SHA、工具版本和资源快照；目标资源使用约 60%-80%。可用内存不足、swap 持续增长、宿主机 I/O 异常或无关容器持续重启时暂停，不停止无关服务、不宽泛杀进程、不重启整机。
- 每项检查都要保存命令、并发参数、日志路径和退出码；任一全量检查未通过，不能进入交付。

## Docker 与镜像

- Linux 使用当前 Tokyo 项目的 Dockerfile 和 Compose 文件；核对架构、服务、健康检查、端口、网络、卷和 Postgres `18-alpine`/Redis `8-alpine` 依赖。应用代码变化后镜像摘要可以不同，但运行条件必须一致。
- 应用镜像使用不可变提交 SHA 标签，并写入 `org.opencontainers.image.revision=<commit-sha>`；不要使用含糊的 `stable` 标签作为构建产物。
- 生产部署获单独授权后，Linux 用 `docker save` 压缩包经 SSH 传到 Tokyo，校验 SHA256 后再由部署流程加载。当前准备阶段不加载镜像、不改生产 `.env`、不执行 Compose 或 systemd 重启。
- 不新增 cgroup、systemd 限额或代码级限制；安全边界由本文件规定的工作区、顺序和记录要求保证。

## Tokyo 拉取规则

在 Linux 全量检查和镜像构建成功后，Tokyo 只执行：

```bash
cd /srv/sub2api/repo
git fetch origin main
git merge --ff-only origin/main
git status --short
```

禁止在 Tokyo 执行 `docker compose build/up/restart`、`systemctl restart`、生产环境变量修改或测试任务。拉取代码不等于部署；stable 容器、镜像、数据和服务状态必须保持原样。

## 交付标准

- 远端 `main`、Linux 主工作区和 Tokyo 仓库指向同一提交，且提交图能证明包含指定上游基准。
- 全量检查、迁移升级验证、配置/脚本静态检查和独立审查全部通过，冲突标记已清除。
- 镜像 revision 标签、摘要、构建日志和检查日志可追溯；没有生产密钥、runtime 或数据库数据进入提交或镜像。
- 交付说明必须列出实际命令、顺序、结果和未执行项；未授权部署、重启和旧目录删除均不得暗中执行。
