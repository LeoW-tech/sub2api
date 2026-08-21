# AGENTS.md

请始终用中文与用户交流。

每次完成代码、脚本、配置或文档改动后，请及时提交本地 git。除非用户明确要求，否则不要自动推送远端。

## 项目现状

这是一个已经完成本地单仓库整合的 Sub2API 项目。

- Linux 主运行面：
  - 当前仓库根目录：`/srv/sub2api/repo`
  - 当前 runtime 根目录：`/srv/sub2api/runtime`
- Mac 辅助运行面：
  - 仓库根目录：`/Users/meilinwang/Projects/sub2api`
  - 用途：从 GitHub 拉取更新、开发验证、必要时作为备用运行面
- 你的 fork：`origin = https://github.com/LeoW-tech/sub2api.git`
- 原始仓库：`upstream = https://github.com/Wei-Shaw/sub2api.git`
- 稳定集成分支：`main`
- 上游镜像分支：`upstream-main`

本仓库不是“纯上游镜像”，而是：

1. 按任务指定的 `upstream/main`、tag 或 release 同步原始仓库
2. 同时维护用户自己的定制功能
3. 使用双环境运行，避免开发环境影响稳定环境

## 分类导航

当前仓库内的信息按下面方式分流：

- 项目事实与协作约束：保留在本文件
- 常用命令、运维入口、开发流程、同步流程：见 [`常用命令.md`](常用命令.md)
- 本地运维细节说明：见 [`docs/LOCAL_DEVELOPMENT_MAINTENANCE.md`](docs/LOCAL_DEVELOPMENT_MAINTENANCE.md)

## VPS 开发验证基础设施

东京 VPS 已持久化配置独立的开发验证工具链，不与 stable runtime 共享运行时数据：

- 环境入口：`/home/ubuntu/.config/sub2api/dev-env.sh`
- Go 1.26.6：`/home/ubuntu/.local/toolchains/go1.26.6`，入口在 `/home/ubuntu/.local/bin`
- Go 模块缓存：`/home/ubuntu/.cache/sub2api-go-mod`
- Go 构建缓存：`/home/ubuntu/.cache/sub2api-go-build`
- pnpm store：`/home/ubuntu/.cache/sub2api-pnpm-store`

在 VPS 上执行静态验证前先加载该环境入口；不得把这些缓存、`node_modules` 或运行时数据加入 git。依赖权限应保持为 `ubuntu` 用户可读写，避免 Docker root 缓存阻断后续验证。

## 目录约定

必须遵守下面的目录边界：

- 源码、脚本、文档、部署模板都在仓库内
- 所有运行时数据都收敛到 runtime 根目录
- 严禁把运行时数据重新放回仓库根目录

脚本对 runtime 根目录的真实探测顺序如下：

1. 优先使用仓库内 `repo/runtime/`
2. 若仓库内不存在有效 runtime，则退回仓库同级 `../runtime/`

当前 Linux 现状使用的是仓库同级 runtime：

```text
/srv/sub2api/
  repo/
  runtime/
    stable/
    backups/
```

通用运行时结构如下：

```text
runtime/
  stable/
    .env
    data/
    postgres_data/
    redis_data/
  dev/
    .env
    data/
    postgres_data/
    redis_data/
  backups/
    <timestamp>/
      runtime/
      com.sub2api.autostart.plist
```

说明：

- `runtime/stable` 是稳定环境，默认服务端口 `8080`
- `runtime/dev` 是开发环境，服务端口 `127.0.0.1:8081`
- `runtime/backups` 是默认运行时备份目录
- `runtime/` 整体不进 git
- Linux 当前使用 `systemd` 托管 `stable`；需要固定节点出口时应统一接入 `/srv/egress-control`，但必须先以 `systemctl` 和 `127.0.0.1:19180/health` 核对该主机是否已安装并运行，不得仅按文档假设存在
- Mac 当前使用 `autostart/launchd` 负责登录后自动恢复 stable 栈

前端访问地址：

- 稳定环境前端（本机）：`http://127.0.0.1:8080/`
- 稳定环境前端（局域网）：`http://<本机局域网IP>:8080/`
- 开发环境前端：`http://127.0.0.1:8081/`
- egress-control 健康检查（仅已安装主机）：`http://127.0.0.1:19180/health`

## 分支与维护模式

默认分支和用途如下：

- `main`
  用于稳定集成，只部署用户确认可保留的功能
- `upstream-main`
  只镜像 `upstream/main`，禁止直接开发
- `feature/*`
  日常功能开发分支，从 `main` 切出
- `sync/upstream-YYYYMMDD[-vX.Y.Z]`
  同步上游时的临时分支。可以从 `main` 合入上游基准，也可以以上游基准为底重新叠加本地定制；方向不是约束，功能完整性、来源可追溯和验证结果才是交付标准。上游基准可以是 `upstream-main`，也可以是任务明确指定的 tag/release 对应提交。

工作规则：

- 不要在 `upstream-main` 上开发
- 尽量不要直接在 `main` 上做功能开发
- 新功能优先从 `main` 切 `feature/*`
- 常规跟随 `upstream/main` 时，使用统一脚本更新 `upstream-main` 并创建同步分支；任务明确指定 tag/release 时，以该 tag 对应提交作为同步基准，仍复用既有同步、验证和提交流程

## 上游同步与冲突处理原则

以下原则适用于 `main` 与上游基准的同步、`sync/upstream-*` 分支上的冲突处理，以及同类的 `merge`、`rebase`、`cherry-pick`、`revert` 冲突。

### 核心结果

1. 上游基准包含的功能与本地仍需保留的定制功能必须同时存在，不能因选择某一侧文件而静默丢失。
2. 如果上游已经实现、吸收或重构了与本地定制相同的能力，以上游实现为基准；仅补回上游尚未覆盖且仍有明确需求的本地差异。
3. 如果两边是不同能力，则必须同时保留，并核对类型、调用链、配置、迁移、接口和界面是否仍然一致。
4. 无法确认两边是否属于同一能力，或无法确认取舍影响时，必须停止自动处理，保留冲突并等待人工裁决；禁止猜测性解决。

### 实施约束

- 合并方向不是硬性约束：可以从本地 `main` 合入上游，也可以以上游基准为底叠加本地定制。无论采用哪种方式，都必须保留明确、可审计的上游基准和本地定制来源。
- 任务明确指定 tag/release 时，只同步该 tag 对应提交；除非用户明确要求，不得吸收其后的 `upstream/main` 提交。
- 核心业务文件、配置文件、依赖注入文件和大型前端视图发生冲突时，禁止未经逐项审查直接使用 `ours`、`theirs` 或整文件覆盖。必须按功能和调用链合并。
- 生成代码不得作为普通业务代码手工拼接。先解决生成源和依赖图，再使用项目规定的生成工具重新生成，并确认生成结果无陈旧引用。
- 已经发布或可能落库的迁移文件名和内容视为不可变。不得通过重命名或修改旧迁移解决编号冲突；新变更使用新的迁移文件，并在数据库副本上验证升级路径。
- 迁移编号规则：历史迁移（包括上游带来的 `001–899999` 文件）视为冻结，不改名、不改内容；本地新增迁移统一使用 `900001+` 保留区间，文件名必须为 `NNNNNN_local_description.sql`，并按 `backend/migrations/NEXT_LOCAL_MIGRATION` 连续递增。新增本地迁移后必须递增该文件，并通过 `scripts/check-migration-numbering`；禁止再创建重复的 `225_*`、`226_*` 等普通前缀。
- 前后端接口必须核对路由、请求参数、响应类型和实际调用方，不能只确认 TypeScript 或 Go 类型存在。
- 同步涉及 `frontend/src/i18n/locales/` 的结构性调整时，必须验证受影响组件引用的翻译键在中英文最终合并后的语言树中均可解析；不能只依赖将 `t()` mock 为键名的组件测试。

### 完成标准

每次上游同步在交付前都必须证明：

1. 当前结果包含任务指定的精确上游基准，且未意外吸收范围外的上游提交。
2. 本地定制清单已逐项核对；同功能取上游、不同功能两边保留、疑义项经过人工裁决。
3. 所有冲突标记已清除，生成代码已重新生成，版本号与指定 release 一致。
4. 后端编译和测试、前端类型检查和测试、迁移升级验证、配置与脚本静态检查均有实际通过证据；不能以“应该能通过”代替验证。
5. 同步结果经过独立代码审查后才能合回 `main` 或进入部署流程。

## 重要约束

- 不要把 `runtime/` 下的文件加入 git
- 不要删除或覆盖用户的运行时数据，除非用户明确要求
- 修改稳定环境相关内容时，优先保证 `stable` 可恢复
- 不得恢复旧 `door-gateway` 双轨链路；Sub2API 需要节点出口时只能接入 `/srv/egress-control`
- Linux 侧先检查 `systemd` 与 `/etc/systemd/system/sub2api-stable.service`；如主机已安装 egress-control，再同时检查 `egress-control.service`、`egress-control-docker-bridge.service`
- Mac 侧要同时考虑 `LaunchAgents`、`colima`、`autostart`、`~/Library/LaunchAgents/com.sub2api.autostart.plist`
- 如果调整脚本接口，必须同步更新 `docs/LOCAL_DEVELOPMENT_MAINTENANCE.md`
- 迁移编号静态检查：`scripts/check-migration-numbering`；完整回归：`scripts/tests/check-migration-numbering-test.sh` 或 `make migration-check`
- 这套仓库服务的是双机同步模式：Linux 通常负责提交并按需推送，Mac 从 `origin` 拉取同步更新

## 完成前检查

在声称完成之前，至少确认：

1. `git status` 是否干净或是否只剩预期改动
2. 如涉及 stable/dev 运行面，相关服务是否真的可访问
3. 如在 Linux 上操作稳定环境，至少检查 `./scripts/sub2api-local stable status`、`./scripts/sub2api-local systemd status`；只有主机已安装 egress-control 时才要求 `http://127.0.0.1:19180/health`
4. 如在 Mac 上操作自动恢复链路，至少检查 `./scripts/sub2api-local autostart status`
5. 变更是否已经提交本地 git
