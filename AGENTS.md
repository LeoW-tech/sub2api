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

1. 跟踪原始仓库最新 `upstream/main`
2. 在本地 `main` 上叠加用户自己的定制功能
3. 使用双环境运行，避免开发环境影响稳定环境

## 分类导航

当前仓库内的信息按下面方式分流：

- 项目事实与协作约束：保留在本文件
- 常用命令、运维入口、开发流程、同步流程：见 [`常用命令.md`](常用命令.md)
- 本地运维细节说明：见 [`docs/LOCAL_DEVELOPMENT_MAINTENANCE.md`](docs/LOCAL_DEVELOPMENT_MAINTENANCE.md)

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
      com.sub2api.autostart.plist
      com.sub2api.door-gateway.plist
```

说明：

- `runtime/stable` 是稳定环境，默认服务端口 `8080`
- `runtime/dev` 是开发环境，服务端口 `127.0.0.1:8081`
- `runtime/backups` 是默认运行时备份目录
- `door-gateway` 配置在 `runtime/stable/door-gateway.json`
- `door-gateway` worker 数据在 `runtime/stable/door-workers/`
- `runtime/` 整体不进 git
- Linux 当前使用 `systemd` 托管 `stable + door-gateway`
- Mac 当前使用 `autostart/launchd` 负责登录后自动恢复

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

## 冲突处理原则

- 适用范围包括 `main` 与 `upstream-main` 的同步合并、`sync/upstream-*` 分支上的冲突处理，以及同类的 `merge`、`cherry-pick`、`revert` 冲突。
- 本地已有且仍需保留的定制功能，如果上游本次更新没有覆盖该能力，冲突时优先保留本地定制。
- 如果本地定制对应的能力，上游本次更新已经实现、吸收或以新的结构重构覆盖，冲突时以上游实现为准，再按需重新评估是否补回少量仍然必要的本地差异。
- 如果无法明确判断两边是否属于同一能力，或无法确认取舍后是否会影响当前运行面与既有行为，就停止自动处理，保留冲突点，等待人工裁定。

## 重要约束

- 不要把 `runtime/` 下的文件加入 git
- 不要删除或覆盖用户的运行时数据，除非用户明确要求
- 修改稳定环境相关内容时，优先保证 `stable` 可恢复
- 修改 `door-gateway` 时，要同时考虑配置路径、日志路径以及当前平台的托管方式
- Linux 侧要同时考虑 `systemd`、`/etc/systemd/system/sub2api-stable.service`、`/etc/systemd/system/sub2api-door-gateway.service`
- Mac 侧要同时考虑 `LaunchAgents`、`colima`、`autostart`、`~/Library/LaunchAgents/com.sub2api.autostart.plist`、`~/Library/LaunchAgents/com.sub2api.door-gateway.plist`
- 如果调整脚本接口，必须同步更新 `docs/LOCAL_DEVELOPMENT_MAINTENANCE.md`
- 这套仓库服务的是双机同步模式：Linux 通常负责提交并按需推送，Mac 从 `origin` 拉取同步更新

## 完成前检查

在声称完成之前，至少确认：

1. `git status` 是否干净或是否只剩预期改动
2. 如涉及 stable/dev 运行面，相关服务是否真的可访问
3. 如在 Linux 上操作稳定环境，至少检查 `./scripts/sub2api-local stable status`、`./scripts/sub2api-local systemd status` 与 `http://127.0.0.1:19080/health`
4. 如在 Mac 上操作自动恢复链路，至少检查 `./scripts/sub2api-local autostart status` 与 `http://127.0.0.1:19080/health`
5. 变更是否已经提交本地 git
