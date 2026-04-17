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

## 分类导航

当前仓库内的信息按下面方式分流：

- 项目事实与协作约束：保留在本文件
- 常用命令、运维入口、开发流程、同步流程：移至根目录 [`常用命令.md`](/Users/meilinwang/Projects/sub2api/常用命令.md)
- 本地运维细节说明：见 [`docs/LOCAL_DEVELOPMENT_MAINTENANCE.md`](/Users/meilinwang/Projects/sub2api/docs/LOCAL_DEVELOPMENT_MAINTENANCE.md)

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
