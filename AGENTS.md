# AGENTS.md

请始终用中文与用户交流；每次完成代码、脚本、配置或文档变更后，及时提交本地 Git。

## 双机分工

- Linux `192.168.31.214` 只用 `/srv/sub2api/primary` 作为主工作区，合并、构建、全量检查和镜像制作都在这里完成。
- Tokyo `43.165.188.95`（`tokyo-vps`）只做 `git fetch`、`git merge --ff-only` 和经授权的部署；准备阶段不启动/重启 stable 服务，不改生产配置或数据。为完成镜像交付，允许把已校验的镜像归档传入 `incoming` 并在部署窗口加载；加载不等于启动服务。
- Linux 旧目录 `/srv/sub2api/repo` 只作历史副本，未确认新工作区前不清理、不覆盖。
- Linux 连接 Tokyo 必须使用 `/home/lim/.ssh/config` 中的 `tokyo-vps`（`ubuntu@43.165.188.95`）和 `~/.ssh/sub2api_tokyo_ed25519`；不得改用 `lim`，也不得在 prompt、日志或仓库暴露私钥。先用 `ssh -o BatchMode=yes -o ConnectTimeout=10 tokyo-vps 'hostname; id -un'` 验证连接。
- GitHub HTTPS 直连失败时，Linux 使用本机 egress `http://127.0.0.1:19181`，先检查 `http://127.0.0.1:19180/health`；该 egress 不替代 Tokyo 的 SSH 密钥。
- `origin=https://github.com/LeoW-tech/sub2api.git`，`upstream=https://github.com/Wei-Shaw/sub2api.git`，稳定分支为 `main`。生产密钥、runtime、数据库和 Redis 数据不进入 Git 或镜像。

## 合并规则

- 指定上游 release/tag 时，先核对真实提交，再将其合并进当前集成 `main`；不得仅凭 HEAD 文本版号判断。
- 任务指定 release/tag 时，用真实提交、`git merge-base --is-ancestor`、提交图和差异确认来源，不靠 HEAD 文本版号猜测。
- 指定 release/tag 优先于滚动的 `upstream/main`。记录 tag 对象、解引用后的提交及 fetch 时的分支头；tag 之后的提交须逐项审查并明确纳入范围，不得自动扩大到最新分支头。仅同步版本号的后续提交也须记录依据。
- 当前 `main` 是带本地定制的集成分支，不能为了追上游而重置或丢弃定制功能。
- 同一能力优先保留上游实现，不同能力两边都保留；冲突按功能、调用链、配置、迁移、接口逐项处理，无法确认就停下来报告，禁止批量 `ours`/`theirs`。
- 已发布迁移不可改名或修改；上游迁移保留原文件名、编号及内容，只有本地新增迁移使用 `900001+` 区间。迁移按完整文件名排序和登记，上游数字前缀相同不等于冲突；合并后检查编号、嵌入文件和已执行迁移的校验和。生成代码先修复生成源，再按项目工具重新生成。
- 需要核对上游时先 `git fetch upstream main --tags`（直连失败时使用 egress），以 `upstream/main` 或 `refs/remotes/upstream/main` 为准；不要使用不存在的 `refs/remotes/origin/upstream`。
- Linux 主工作区检查和构建成功后，必须先以可审计的 fast-forward 执行 `git push origin main`，再让 Tokyo 从 `origin/main` 快进；禁止 force push、destructive reset 和未经审查的整文件覆盖。

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

## 数据迁移与恢复

- Tokyo 的正式数据库、配置及运行镜像是生产基准；Linux 仅用于开发和隔离验证，不得用本地数据覆盖生产数据。部署前只读核对生产已执行迁移及待执行 SQL，记录相关表大小、预计影响行数和迁移超时的实际生效路径和值；区分初始安装超时与正常启动迁移超时。
- 涉及数据或约束变更时，部署前必须生成 PostgreSQL 一致性备份并校验 SHA256，记录生产旧镜像 ID/revision、迁移清单、备份位置和恢复命令。备份和配置保存在受限的非 Git、非构建上下文目录；配置副本或不完整的目录复制不能视为数据库备份。
- 在 Linux 隔离数据库中验证备份恢复、升级迁移、重复启动和旧应用对升级后数据库的兼容性；全程禁用生产出站访问及后台业务任务，禁止把生产凭据用于测试请求。记录删除型迁移预计与实际影响行数、迁移校验和及保留数据验证结果，验证不通过不得交付。
- 普通 SQL 迁移逐文件提交：失败只回滚当前文件，已提交文件不会自动撤销。切回旧镜像不等于恢复数据库；仅在已验证兼容时允许应用回退。数据库恢复须评估备份之后的新增数据并先停止写入，避免自动覆盖新产生的正式数据；不能安全恢复时停止部署并报告。

## Docker 与镜像

- Linux 使用当前 Tokyo 项目的 Dockerfile 和 Compose 文件，核对架构、服务、健康检查、端口、网络、卷和 Postgres `18-alpine` / Redis `8-alpine` 依赖。应用代码变化后镜像摘要可以不同，但运行条件必须一致。
- 应用镜像使用提交 SHA 标签，并写入 `org.opencontainers.image.revision=<commit-sha>`；不要用含糊的 `stable` 标签作为交付产物。
- 不使用公共或付费镜像仓库。Linux 先记录镜像 ID 和 revision，再执行 `docker save sub2api:<commit-sha> | gzip > sub2api-<commit-sha>.tar.gz`、`sha256sum sub2api-<commit-sha>.tar.gz > sub2api-<commit-sha>.tar.gz.sha256`，并用 `scp sub2api-<commit-sha>.tar.gz* tokyo-vps:/srv/sub2api/incoming/` 传输。
- Tokyo 在部署窗口进入 `incoming`，先执行 `sha256sum -c sub2api-<commit-sha>.tar.gz.sha256`，再执行 `docker load < sub2api-<commit-sha>.tar.gz`；用 `docker image inspect` 核对加载后的镜像 ID、`org.opencontainers.image.revision` 与 Linux 记录一致。部署或健康检查失败时按已验证的数据库兼容性和恢复方案决定是否切回旧镜像，旧镜像、生产数据和配置不得删除。
- 准备阶段不启动 Compose 或 systemd，不改生产 `.env`。最终启动必须复用已加载的 SHA 镜像，例如在仓库根目录执行 `SUB2API_IMAGE=sub2api:<commit-sha> ./scripts/sub2api-runtime-compose stable up -d --no-build`；该脚本负责注入 runtime 卷路径和 Linux override。禁止在 Tokyo 重新 `build`，以免绕过 Linux 的构建和检查结果。

## Tokyo 拉取

Linux 检查和镜像构建成功、且已将 `main` fast-forward 推送到 `origin` 后，Tokyo 先只执行：

```bash
cd /srv/sub2api/repo
git fetch origin main
git merge --ff-only origin/main
git status --short
```

拉取代码不等于部署；在最终部署窗口之前，禁止执行 Compose `up/restart`、`systemctl restart`、生产环境变量修改或测试任务，stable 容器、数据和服务状态必须保持原样。最终窗口只允许使用已校验的 SHA 镜像执行一次 `./scripts/sub2api-runtime-compose stable up -d --no-build`，随后按健康检查决定保留或切回旧镜像。

## 交付

- 远端 `main`、Linux 主工作区和 Tokyo 仓库最终指向同一提交，且提交图能证明包含指定上游基准。
- 全量检查、迁移升级验证、配置/脚本/生成代码静态检查和独立复查全部通过。
- 涉及迁移时，交付证据必须包括可恢复备份、升级前后迁移状态和实际影响行数；不得只以镜像回退或 `/health` 成功代替数据库及关键业务验证。
- 镜像 revision 标签、摘要、构建日志和检查日志可追溯；没有生产密钥、runtime 或数据库数据进入提交或镜像。
- 交付说明必须列出实际命令、顺序、结果和未执行项；未授权部署、重启和旧目录删除均不得暗中执行。
