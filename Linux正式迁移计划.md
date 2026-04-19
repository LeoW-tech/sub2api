# Sub2API Linux 正式迁移计划

## 1. 目标与使用方式

本文是本仓库从当前 macOS 本地运行环境迁移到局域网内 Linux 正式节点的执行手册，供三类角色共用：

- `A`：mac 机器上的 AI
- `B`：人类
- `C`：Linux 机器上的 AI

本文不是概要说明，而是一份可执行手册。执行时必须遵守以下规则：

- 所有阶段按角色拆分，不混合角色。
- `A` 和 `C` 能完成的动作，不分配给 `B`。
- `B` 只负责必须由人类完成的授权、登录、最终外部验收和网络侧确认。
- 每个阶段先看“完成条件”，再按角色顺序执行。
- 任一角色在本阶段达到“失败时暂停点”后，必须停止推进，等待上一个阻塞点被消除。

## 2. 锁定的技术决策

以下决策已经锁定，本次迁移不再临时改方案：

- 首发公网入口采用 `Cloudflare Named Tunnel`。
- 正式访问域名采用 `api.<主域名>`。
- Linux 正式部署目录采用 `/srv/sub2api`。
- Sub2API 继续采用 `Docker Compose` 运行。
- 不复用 Linux 现有宿主机 PostgreSQL 14 与 Redis。
- `door-gateway` 继续迁移，但作为 Linux 宿主机服务运行。
- 首发不引入本地 `Caddy` 或 `Nginx`。
- 首发不做路由器 `80/443` 端口转发。
- 首发不把 Sub2API 直接暴露到公网宿主机端口。
- 首发不在整站前增加 Cloudflare Access，以免影响朋友直接访问前台和 API。

## 3. 当前已知环境摘要

### 3.1 mac 源端现状

- 仓库根目录：`/Users/meilinwang/Projects/sub2api`
- 当前仓库分支：`main`
- 当前稳定运行时目录：`runtime/stable`
- 当前开发运行时目录：`runtime/dev`
- 当前稳定环境核心资产：
  - `runtime/stable/.env`
  - `runtime/stable/data/`
  - `runtime/stable/postgres_data/`
  - `runtime/stable/redis_data/`
  - `runtime/stable/door-gateway.json`
  - `runtime/stable/door-workers/`
- 当前本地维护入口：
  - [`scripts/sub2api-local`](/Users/meilinwang/Projects/sub2api/scripts/sub2api-local)
  - [`docs/LOCAL_DEVELOPMENT_MAINTENANCE.md`](/Users/meilinwang/Projects/sub2api/docs/LOCAL_DEVELOPMENT_MAINTENANCE.md)
- 当前 Compose 运行基础：
  - [`deploy/local/docker-compose.runtime.yml`](/Users/meilinwang/Projects/sub2api/deploy/local/docker-compose.runtime.yml)

### 3.2 Linux 目标端现状

- 系统：Ubuntu 22.04.5 LTS
- 内核：`5.15.0-173-generic`
- CPU：`x86_64`，`16` 逻辑核心
- 内存：`31Gi`
- 根分区剩余空间：约 `523G`
- 时区：`Asia/Shanghai`
- 主机名：`lim`
- 当前局域网 IP：`192.168.31.214`
- Docker：未安装
- Docker Compose：未安装
- `cloudflared`：未安装
- Node.js：已安装，`v20.20.0`
- npm：已安装，`10.8.2`
- git：已安装
- rsync：已安装
- tar：已安装
- systemd：可用
- 防火墙：`ufw` 与 `firewalld` 当前都未生效
- SELinux：禁用
- 宿主机 PostgreSQL：已运行，监听 `127.0.0.1:5432`
- 宿主机 Redis：已运行，监听 `127.0.0.1:6379`
- 宿主机 `clash`：已运行，监听 `127.0.0.1:7890`、`127.0.0.1:7891`、`127.0.0.1:9090`
- `80/443/8080/19080` 当前未被占用

### 3.3 网络与公网入口判断

- Linux 主机位于家庭局域网内，当前只能确定主动外连 `443/TCP` 正常。
- 当前无法仅凭宿主机确认是否存在 CGNAT。
- 当前不具备“适合立即做稳定公网直连”的充分证据。
- 结论：首发公网入口优先使用 `Cloudflare Tunnel`，后续再评估是否升级为端口转发 + 本地反代。

## 4. 目标架构

### 4.1 首发正式架构

```text
朋友/外部用户
    ↓
https://api.<主域名>
    ↓
Cloudflare DNS + Named Tunnel
    ↓
Linux 宿主机 cloudflared(systemd)
    ↓
127.0.0.1:8080 上的 Sub2API 容器服务
    ↓
容器内 PostgreSQL / Redis

Linux 宿主机 door-gateway(systemd)
    ↑
Sub2API 容器通过 host.docker.internal:19080 访问
```

### 4.2 Linux 目录布局

```text
/srv/sub2api/
  repo/                    仓库代码
  runtime/
    stable/
      .env
      data/
      postgres_data/
      redis_data/
      door-gateway.json
      door-workers/
  backups/                 迁移包、回滚包、校验文件
  door-gateway/
    config/
    logs/
    workers/
  bin/                     mihomo 等宿主机二进制
```

### 4.3 首发监听策略

- `sub2api`：仅绑定 `127.0.0.1:8080`
- `postgres`：不对宿主机暴露端口
- `redis`：不对宿主机暴露端口
- `door-gateway`：绑定 `192.168.31.214:19080`
- `cloudflared`：通过本地回环访问 `127.0.0.1:8080`

## 5. 角色定义与边界

### 5.1 A：mac 机器上的 AI

负责内容：

- 校验源端当前运行状态
- 执行源端备份
- 打包和整理迁移交付物
- 生成 Linux 侧需要的配置参考
- 在切换窗口内冻结源端并做最终增量包
- 记录源端回滚命令和恢复顺序

禁止内容：

- 不负责 Cloudflare 登录
- 不负责 Linux 端安装 Docker 或 `cloudflared`
- 不负责 Linux 宿主机 service 创建

### 5.2 B：人类

负责内容：

- 登录 Cloudflare 并提供必要授权
- 确认正式主域名
- 如需要，确认 Linux 端 `sudo` 安装授权
- 如需要，在路由器上给 Linux 做 DHCP 保留
- 进行外部网络访问验收
- 邀请朋友完成真实访问验收

禁止内容：

- 不手工改 Linux 配置文件
- 不手工解包运行时目录
- 不手工搭 Docker Compose
- 不手工部署 `door-gateway`

### 5.3 C：Linux 机器上的 AI

负责内容：

- 安装 Docker Engine 与 Compose v2
- 安装 `cloudflared`
- 创建 `/srv/sub2api` 目录树
- 还原运行时目录
- 调整 Linux 专用 `.env`
- 调整 Linux 专用 `door-gateway.json`
- 安装 `door-gateway` systemd 服务
- 安装 `cloudflared` systemd 服务
- 启动 Sub2API 与 `door-gateway`
- 执行本地与外部连通性验证

禁止内容：

- 不改变架构决策
- 不擅自改成宿主机 PostgreSQL/Redis 承载
- 不擅自改成公网直连或本地 Caddy/Nginx 首发

## 6. 前置条件清单

### 6.1 必须在开始前确认的输入

- `主域名`：由 `B` 确认，例如 `example.com`
- `正式子域名`：固定为 `api.<主域名>`
- `Linux sudo`：`B` 同意 `C` 使用 sudo 安装运行面
- `Linux 固定局域网地址`：建议继续使用 `192.168.31.214`
- `Cloudflare 账号可操作`：`B` 可登录并创建 Tunnel 与 DNS 记录
- `迁移停机窗口`：建议预留 `30-60` 分钟

### 6.2 源端必须准备的迁移交付物

- 仓库代码快照
- `runtime/stable/.env`
- `runtime/stable/data/`
- `runtime/stable/postgres_data/`
- `runtime/stable/redis_data/`
- `runtime/stable/door-gateway.json`
- `runtime/stable/door-workers/`
- 如有必要的本地维护脚本说明，一并打包

### 6.3 首发必须避免的事项

- 不复用 Linux 现有宿主机 PostgreSQL
- 不复用 Linux 现有宿主机 Redis
- 不复用 `/home/lim/clash/clash` 作为 Sub2API 专属 `door-gateway` 内核
- 不让 Sub2API 直接监听 `0.0.0.0:8080`
- 不在未完成本地健康检查前启用 Cloudflare Tunnel

## 7. 分阶段迁移步骤

---

## Phase 0：冻结方案与角色边界

### 完成条件

- 正式架构、角色边界、正式域名形态和首发入口方案已经锁定。
- 三类角色都知道自己只执行自己的步骤。

### A 步骤组

**目标**

- 将当前计划文档定稿为源端与目标端共同遵循的执行基线。

**输入**

- 本文档
- 当前仓库结构

**输出**

- 一份可供 A、B、C 共用的迁移手册

**执行动作**

1. 保持本文档为唯一正式执行手册。
2. 确认源端不再继续扩展迁移范围。
3. 将“首发不引入 Caddy/Nginx”“首发使用 Cloudflare Tunnel”作为固定前提。

**成功判定**

- 后续任何步骤均以本文档为准，不再出现并行方案。

**失败时暂停点**

- 如果出现“想临时改成端口转发”“想复用宿主机数据库”等新方案，暂停迁移并重新审议。

### B 步骤组

**目标**

- 明确主域名和人工介入边界。

**输入**

- Cloudflare 账号
- 你的域名

**输出**

- 确认后的 `主域名`
- 确认后的 `api.<主域名>`

**执行动作**

1. 确认本次正式域名采用 `api.<主域名>`。
2. 确认主域名本身暂不承载 Sub2API。
3. 确认自己只负责授权、登录和最终验收，不进入 Linux 手工改配置。

**成功判定**

- `api.<主域名>` 已被确定为唯一正式入口。

**失败时暂停点**

- 若主域名本身也想同步承载官网或其他站点，暂停并先重新设计域名策略。

### C 步骤组

**目标**

- 确认 Linux 端将按固定架构执行，不擅自变更部署方式。

**输入**

- 本文档
- Linux 环境采集结果

**输出**

- 可执行的 Linux 端任务边界

**执行动作**

1. 接受 `/srv/sub2api` 为正式目录。
2. 接受 Docker Compose 为正式运行面。
3. 接受 `cloudflared + systemd` 为正式入口层。
4. 接受 `door-gateway` 作为宿主机服务。

**成功判定**

- C 后续不会擅自切换到宿主机原生部署、端口转发或复用本机数据库。

**失败时暂停点**

- 如果 Linux 环境出现无法安装 Docker 或 `cloudflared` 的硬性限制，暂停迁移。

---

## Phase 1：源端导出准备

### 完成条件

- 源端运行状态已记录。
- 第一份完整迁移包已生成。
- 最终切换时所需的交付物清单已整理完毕。

### A 步骤组

**目标**

- 在不打断当前服务的前提下，准备第一次完整迁移包和交付清单。

**输入**

- 源端仓库
- `runtime/stable`
- 当前稳定服务状态

**输出**

- 第一份完整迁移包
- 迁移交付清单
- 回滚命令草案

**执行动作**

1. 校验源端运行健康：
   - `./scripts/sub2api-local stable logs`
   - `./scripts/sub2api-local door status`
   - `curl http://127.0.0.1:8080/health`
   - `curl http://127.0.0.1:19080/health`
2. 记录当前可用能力：
   - 管理员可登录
   - 至少一个普通用户可访问
   - 至少一个 API key 可调用
   - 至少一个依赖专属出口的账号可正常出流
3. 在不停止源端的前提下打第一份完整包，建议打包内容：
   - 仓库代码快照
   - `runtime/stable/.env`
   - `runtime/stable/data/`
   - `runtime/stable/postgres_data/`
   - `runtime/stable/redis_data/`
   - `runtime/stable/door-gateway.json`
   - `runtime/stable/door-workers/`
4. 生成迁移交付清单，列出每个目录的用途与必须性。
5. 记录源端回滚命令顺序：
   - `./scripts/sub2api-local stable up`
   - `./scripts/sub2api-local door restart`

**成功判定**

- 第一份迁移包可列出完整目录结构。
- 交付清单与回滚命令已整理完毕。

**失败时暂停点**

- 如果源端健康检查不通过，暂停导出并先恢复源端稳定性。

### B 步骤组

**目标**

- 为后续授权准备必要人工输入。

**输入**

- Cloudflare 账号

**输出**

- 人工确认后续可登录 Cloudflare

**执行动作**

1. 确认自己能登录 Cloudflare。
2. 确认迁移窗口内可以配合完成授权。

**成功判定**

- Cloudflare 登录未受阻。

**失败时暂停点**

- 如果无法登录 Cloudflare，暂停整个迁移推进。

### C 步骤组

**目标**

- 无动作。

**输入**

- 无

**输出**

- 无

**执行动作**

- 无动作。

**成功判定**

- 无

**失败时暂停点**

- 无

---

## Phase 2：Linux 运行面准备

### 完成条件

- Linux 安装好 Docker Engine、Compose v2 和 `cloudflared`。
- `/srv/sub2api` 目录树已创建完成。
- Linux 具备恢复运行时目录和启动服务的基础能力。

### A 步骤组

**目标**

- 向 C 提供 Linux 侧需要的运行参考。

**输入**

- 源端 Compose 与 `.env`

**输出**

- Linux 专用配置修改参考

**执行动作**

1. 根据源端 `.env` 整理 Linux 侧必须保留的生产参数。
2. 明确 Linux 侧必须改动项：
   - `BIND_HOST=127.0.0.1`
   - 保留 `SERVER_PORT=8080`
   - 保留 JWT、TOTP、数据库、Redis 等固定配置
3. 明确 Linux 不可变更项：
   - 不改为宿主机 PostgreSQL/Redis
   - 不改为 `0.0.0.0:8080`

**成功判定**

- C 拿到 Linux 配置修改参考后，无需再询问架构性问题。

**失败时暂停点**

- 如果发现源端 `.env` 缺失固定密钥，暂停并先确认源端真实运行配置。

### B 步骤组

**目标**

- 为 Linux 安装运行面提供必要授权。

**输入**

- Linux 主机 sudo 权限

**输出**

- 允许 C 安装 Docker 与 `cloudflared`

**执行动作**

1. 确认 C 可使用 sudo 安装基础运行面。
2. 如果需要密码或人工确认，及时提供。

**成功判定**

- C 可以继续执行安装。

**失败时暂停点**

- 如果 Linux 端 sudo 无法使用，暂停迁移。

### C 步骤组

**目标**

- 在 Linux 上建立正式运行面和目录结构。

**输入**

- Linux 系统
- B 的 sudo 授权

**输出**

- Docker
- Compose v2
- `cloudflared`
- `/srv/sub2api` 目录树

**执行动作**

1. 安装 Docker Engine。
2. 安装 Docker Compose v2。
3. 安装 `cloudflared`。
4. 创建以下目录：
   - `/srv/sub2api/repo`
   - `/srv/sub2api/runtime/stable`
   - `/srv/sub2api/backups`
   - `/srv/sub2api/door-gateway/config`
   - `/srv/sub2api/door-gateway/logs`
   - `/srv/sub2api/door-gateway/workers`
   - `/srv/sub2api/bin`
5. 预创建 systemd 管理所需目录和日志目录。
6. 记录 Docker 版本、Compose 版本与 `cloudflared` 版本。

**成功判定**

- Docker 命令可用。
- `docker compose version` 可用。
- `cloudflared --version` 可用。
- `/srv/sub2api` 目录树完整存在。

**失败时暂停点**

- 如果 Docker 或 `cloudflared` 安装失败，暂停迁移，不进入数据传输阶段。

---

## Phase 3：Cloudflare Tunnel 与正式域名准备

### 完成条件

- Cloudflare 上已经创建 `api.<主域名>` 对应的 Named Tunnel。
- Linux 上 `cloudflared` 已具备正式服务配置，但尚未对未验证服务提前切流。

### A 步骤组

**目标**

- 向 B 与 C 输出正式入口约束。

**输入**

- 已锁定的架构决策

**输出**

- Tunnel 与域名约束说明

**执行动作**

1. 告知 C：Tunnel 目标服务是 `http://127.0.0.1:8080`。
2. 告知 B：正式 DNS 入口只使用 `api.<主域名>`。
3. 告知双方：主域名本身不接 Sub2API。

**成功判定**

- B 与 C 都以 `api.<主域名>` 为唯一正式入口开展后续动作。

**失败时暂停点**

- 如果 B 希望同时暴露主域名，暂停并单独重设计。

### B 步骤组

**目标**

- 完成 Cloudflare 侧必须由人类进行的登录和授权动作。

**输入**

- Cloudflare 账号
- 主域名

**输出**

- 可供 C 接入的 Tunnel 管理权限

**执行动作**

1. 登录 Cloudflare 控制台。
2. 确认域名已在 Cloudflare 托管。
3. 为 C 后续创建 Named Tunnel 提供必要授权。
4. 确认正式子域名采用 `api.<主域名>`。

**成功判定**

- Cloudflare 侧没有账号权限障碍。

**失败时暂停点**

- 若 Cloudflare 账号无法操作该域名，暂停迁移。

### C 步骤组

**目标**

- 在 Linux 上完成 Tunnel 配置准备。

**输入**

- B 的 Cloudflare 授权
- `cloudflared`

**输出**

- 已配置的 Named Tunnel
- 指向 `127.0.0.1:8080` 的 tunnel 路由定义

**执行动作**

1. 创建 Cloudflare Named Tunnel。
2. 将公共主机名设为 `api.<主域名>`。
3. 将 tunnel 服务目标设为 `http://127.0.0.1:8080`。
4. 安装 `cloudflared` systemd 服务，但不要在本地服务未通过验证前做正式切流验证。

**成功判定**

- Tunnel 配置已存在。
- `cloudflared` 可由 systemd 管理。

**失败时暂停点**

- 如果 Tunnel 创建失败或证书/凭据未正确落地，暂停迁移。

---

## Phase 4：冷迁移与数据落盘

### 完成条件

- 最终增量包已从 mac 冷迁移到 Linux。
- Linux 上的代码、运行时和门控配置已经落盘到正式目录。

### A 步骤组

**目标**

- 在最小停机窗口内完成最终导出。

**输入**

- 第一份完整迁移包
- 当前源端稳定服务

**输出**

- 最终冷迁移包

**执行动作**

1. 停止源端稳定环境：
   - `./scripts/sub2api-local stable down`
2. 停止源端 `door-gateway`：
   - 使用源端既有方式停止
3. 立即执行最终增量打包，确保以下内容与停机后状态一致：
   - `runtime/stable/.env`
   - `runtime/stable/data/`
   - `runtime/stable/postgres_data/`
   - `runtime/stable/redis_data/`
   - `runtime/stable/door-gateway.json`
   - `runtime/stable/door-workers/`
   - 仓库代码快照
4. 将最终冷迁移包交付给 C。

**成功判定**

- 最终迁移包完成生成且已传输到 Linux。

**失败时暂停点**

- 如果停机后无法生成最终包，暂停并优先保证源端可回滚。

### B 步骤组

**目标**

- 配合传输过程，不直接参与技术操作。

**输入**

- 迁移窗口

**输出**

- 迁移窗口持续可用

**执行动作**

1. 保持迁移窗口内不对源端做额外操作。
2. 如遇 sudo 或账号确认，及时响应。

**成功判定**

- A 与 C 在迁移窗口内未被人工操作打断。

**失败时暂停点**

- 如果迁移窗口被迫中断，暂停切换并准备回滚。

### C 步骤组

**目标**

- 将最终迁移包恢复到 Linux 正式目录。

**输入**

- 最终冷迁移包
- `/srv/sub2api` 目录树

**输出**

- Linux 上已还原的代码与运行时目录

**执行动作**

1. 将仓库代码恢复到 `/srv/sub2api/repo`。
2. 将运行时恢复到 `/srv/sub2api/runtime/stable`。
3. 校验以下路径完整存在：
   - `/srv/sub2api/runtime/stable/.env`
   - `/srv/sub2api/runtime/stable/data`
   - `/srv/sub2api/runtime/stable/postgres_data`
   - `/srv/sub2api/runtime/stable/redis_data`
   - `/srv/sub2api/runtime/stable/door-gateway.json`
   - `/srv/sub2api/runtime/stable/door-workers`
4. 将迁移包副本保留一份到 `/srv/sub2api/backups`。

**成功判定**

- Linux 正式目录已具备完整运行时资产。

**失败时暂停点**

- 如果任意关键目录缺失，暂停启动流程。

---

## Phase 5：服务启动与本地验证

### 完成条件

- Linux 本地 `door-gateway` 正常。
- Linux 本地 Sub2API 容器正常。
- 宿主机内网访问和本机访问都通过。

### A 步骤组

**目标**

- 为 C 提供 Linux 配置修订依据。

**输入**

- 源端 `.env`
- 源端 `door-gateway.json`

**输出**

- Linux 专用修订规则

**执行动作**

1. 明确 Linux `.env` 修订规则：
   - `BIND_HOST=127.0.0.1`
   - 保留 `SERVER_PORT=8080`
   - 保留全部固定密钥和调度配置
2. 明确 Linux `door-gateway.json` 修订规则：
   - `mihomo_binary` 指向 `/srv/sub2api/bin/...`
   - `worker_base_dir` 指向 `/srv/sub2api/door-gateway/workers`
   - 管理接口监听 `192.168.31.214:19080`
   - `sub2api_export_host` 优先使用 `host.docker.internal`

**成功判定**

- C 不需要对 Linux 配置做架构性猜测。

**失败时暂停点**

- 如果源端配置里存在只能在 macOS 运行的硬编码路径，暂停并逐项映射。

### B 步骤组

**目标**

- 在必要时为局域网地址稳定性提供人工确认。

**输入**

- 路由器管理权限

**输出**

- 对 `192.168.31.214` 的稳定性确认

**执行动作**

1. 如有条件，在路由器上为 Linux 设置 DHCP 保留，确保局域网地址保持 `192.168.31.214`。
2. 如暂时无法设置，至少确认当前地址短期不会变。

**成功判定**

- Linux 局域网地址在迁移窗口和首发观察期内稳定。

**失败时暂停点**

- 如果 Linux 局域网地址频繁变化，暂停 `door-gateway` 对接，先固定地址。

### C 步骤组

**目标**

- 完成 Linux 上的本地服务落地与首次健康检查。

**输入**

- `/srv/sub2api/repo`
- `/srv/sub2api/runtime/stable`
- Linux 专用配置修订规则

**输出**

- 已启动的 `door-gateway`
- 已启动的 Sub2API 容器

**执行动作**

1. 修改 Linux 版 `.env`：
   - 设置 `BIND_HOST=127.0.0.1`
   - 保持其他生产参数延续源端
2. 修改 Linux 版 Compose：
   - 确保 `sub2api` 绑定 `127.0.0.1:8080`
   - 为应用容器加入 `extra_hosts`
   - 将 `host.docker.internal` 映射到 `host-gateway`
   - 不向宿主机暴露 `5432/6379`
3. 安装专用 Mihomo/兼容内核到 `/srv/sub2api/bin/`
4. 修改 Linux 版 `door-gateway.json`：
   - 更新 `mihomo_binary`
   - 更新 `worker_base_dir`
   - 管理接口绑定 `192.168.31.214:19080`
5. 安装并启动 `door-gateway` systemd 服务。
6. 验证 `door-gateway`：
   - `curl http://192.168.31.214:19080/health`
   - `curl http://192.168.31.214:19080/export/sub2api`
7. 启动 Sub2API Docker Compose。
8. 验证 Sub2API：
   - `curl http://127.0.0.1:8080/health`
   - 检查容器状态
   - 检查日志中无启动致命错误

**成功判定**

- `door-gateway` 健康检查通过。
- `Sub2API` 健康检查通过。
- Docker 容器中未抢占宿主机 `5432/6379`。

**失败时暂停点**

- 如果 `door-gateway` 未能成功启动，暂停正式入口切流。
- 如果 `Sub2API` 本地健康检查失败，暂停正式入口切流。

---

## Phase 6：正式域名切流与外部验证

### 完成条件

- `api.<主域名>` 已可从外部访问。
- 管理后台、普通访问、API 调用和至少一个代理出流都已验证。

### A 步骤组

**目标**

- 为正式切流提供源端对照基线。

**输入**

- 源端历史运行记录

**输出**

- 对照验证基线

**执行动作**

1. 向 C 提供源端成功案例基线：
   - 一个可登录管理员
   - 一个可访问普通用户
   - 一个有效 API key
   - 一个需要专属出口的账号

**成功判定**

- C 按相同基线执行外部验证。

**失败时暂停点**

- 如果源端原始对照对象失效，暂停朋友侧验收，先修正验证样本。

### B 步骤组

**目标**

- 配合完成 Cloudflare 正式入口验证与真实外网验收。

**输入**

- Cloudflare 控制台
- 外部网络环境

**输出**

- 正式域名可访问性的人类确认

**执行动作**

1. 确认 `api.<主域名>` 已指向 Tunnel。
2. 从非局域网环境访问 `https://api.<主域名>/health`。
3. 从非局域网环境访问正式站点。
4. 邀请至少一位朋友进行真实访问验收。

**成功判定**

- 从外部网络成功访问正式域名。
- 至少一位朋友能够实际访问。

**失败时暂停点**

- 如果正式域名仅在内网可访问，暂停宣布切换完成。

### C 步骤组

**目标**

- 启动 Tunnel 并完成技术侧外部验证。

**输入**

- 本地健康通过的 Sub2API
- 本地健康通过的 `door-gateway`
- Cloudflare Named Tunnel

**输出**

- 正式可用的 `api.<主域名>`

**执行动作**

1. 启动 `cloudflared` systemd 服务。
2. 验证 `cloudflared` 运行状态。
3. 验证正式入口：
   - `https://api.<主域名>/health`
4. 验证业务面：
   - 管理员登录
   - 普通用户访问
   - 一次普通 API 调用
   - 一次流式 API 调用
5. 验证代理能力：
   - 导入 `door-gateway` 导出的代理
   - 至少一个依赖专属出口的账号成功出流

**成功判定**

- 正式域名健康检查通过。
- 登录、普通调用、流式调用和代理出流全部通过。

**失败时暂停点**

- 如果 Cloudflare Tunnel 运行正常但正式域名请求失败，暂停并先修正 Tunnel 或 DNS。
- 如果正式域名正常但代理出流失败，暂停对外宣布完成。

---

## Phase 7：回滚预案确认

### 完成条件

- 回滚路径明确且可执行。
- Cloudflare 入口切回、源端恢复、数据回退三件事都有明确顺序。

### A 步骤组

**目标**

- 保留源端可恢复能力。

**输入**

- 源端完整备份
- 源端仓库与运行时

**输出**

- 源端回滚基础

**执行动作**

1. 不删除源端运行时目录。
2. 保留最终迁移前的完整备份。
3. 保留源端恢复命令：
   - `./scripts/sub2api-local stable up`
   - `./scripts/sub2api-local door restart`

**成功判定**

- 即使 Linux 迁移失败，源端仍可重新启动。

**失败时暂停点**

- 如果源端备份丢失，暂停切换完成声明。

### B 步骤组

**目标**

- 掌握人工侧回切入口。

**输入**

- Cloudflare 控制台

**输出**

- 人工可执行的回切能力

**执行动作**

1. 确认自己知道如何停用 Tunnel 或移除正式子域名路由。
2. 确认自己知道何时需要启动回滚。

**成功判定**

- B 能在需要时快速配合回切。

**失败时暂停点**

- 如果 B 无法操作 Cloudflare，暂停宣布迁移稳定。

### C 步骤组

**目标**

- 明确 Linux 侧回滚动作。

**输入**

- Linux 服务状态

**输出**

- 可执行的技术回滚顺序

**执行动作**

1. 记录回滚顺序：
   - 停止 `cloudflared`
   - 停止 Linux 上 Sub2API Compose
   - 停止 Linux 上 `door-gateway`
2. 如 Linux 首发失败，不继续尝试在线修复导致长时间不可用，优先让入口回切。
3. 保留 Linux 侧迁移日志和错误输出，供后续二次迁移复盘。

**成功判定**

- 发生问题时可快速关闭 Linux 对外入口。

**失败时暂停点**

- 如果无法独立停止 Linux 服务，暂停首发。

---

## Phase 8：迁移后观察期与二期优化

### 完成条件

- 首发服务稳定运行。
- 首发后观察期内没有高优先级故障。
- 二期优化方向已经明确，但不影响首发结果。

### A 步骤组

**目标**

- 对迁移结果做源端对照复盘。

**输入**

- 迁移日志
- 验证结果

**输出**

- 迁移复盘结论

**执行动作**

1. 对照源端基线检查 Linux 侧是否达到同等能力。
2. 整理迁移中暴露出的配置硬编码、脚本假设和可移植性问题。
3. 输出后续可选优化建议。

**成功判定**

- 迁移结果可复盘，可沉淀后续标准化动作。

**失败时暂停点**

- 如果核心能力尚未达成，不进入二期优化讨论。

### B 步骤组

**目标**

- 完成人工观察与朋友侧体验反馈。

**输入**

- 正式域名
- 朋友反馈

**输出**

- 人类体验结论

**执行动作**

1. 在观察期内从外部网络多次访问正式域名。
2. 收集朋友实际访问反馈。
3. 记录是否需要后续增强后台保护或流量治理。

**成功判定**

- 至少确认“可访问、可登录、可调用、体验可接受”。

**失败时暂停点**

- 如果朋友侧频繁访问失败，暂停宣布正式稳定。

### C 步骤组

**目标**

- 输出首发后技术优化路线，但不改变首发架构。

**输入**

- Linux 运行日志
- `cloudflared` 日志
- Docker 容器日志

**输出**

- 二期优化清单

**执行动作**

1. 观察 Docker、Tunnel、`door-gateway` 日志。
2. 确认是否需要优化：
   - `host.docker.internal` 与 `host-gateway` 的长期稳定性
   - `door-gateway` 监听策略
   - 监控、备份和日志归档
3. 将未来升级路线限定为以下可选项：
   - 若家宽公网条件明确稳定，再评估升级为 `Caddy + 80/443` 直连
   - 若未来启用支付，再补 webhook 实战验证
   - 若未来需要官网，再把主域名单独部署给官网

**成功判定**

- 二期优化不会反向破坏首发正式运行结果。

**失败时暂停点**

- 如果观察期内首发架构本身不稳定，暂停进入升级路线。

## 8. 验证清单

### 8.1 本地技术验证

- `door-gateway` 本地健康检查通过
- `Sub2API` 本地健康检查通过
- Docker 容器正常运行
- 宿主机 PostgreSQL/Redis 未被新部署抢占

### 8.2 正式域名验证

- `https://api.<主域名>/health` 正常
- 管理员可登录
- 普通用户可访问
- 普通 API 调用成功
- 流式 API 调用成功

### 8.3 代理能力验证

- `door-gateway` 导出接口正常
- Sub2API 成功导入代理
- 至少一个依赖专属出口的账号成功出流

### 8.4 人类验收

- 从外部网络访问正常
- 至少一位朋友完成真实访问验收

## 9. 回滚方案

### 9.1 回滚触发条件

满足任一条件即可回滚：

- 正式域名无法稳定访问
- 后台无法登录
- 关键 API 调用失败
- 依赖 `door-gateway` 的账号出流失败且短时间无法修复
- Linux 服务状态不稳定，可能长时间中断

### 9.2 回滚顺序

1. `C` 停止 Linux 上 `cloudflared`
2. `B` 确认 Cloudflare 正式入口不再指向失败节点
3. `C` 停止 Linux 上 Sub2API Compose
4. `C` 停止 Linux 上 `door-gateway`
5. `A` 使用源端回滚命令恢复 mac 上稳定环境
6. `A` 验证源端 `stable` 与 `door-gateway` 恢复成功
7. `B` 从外部网络再次验证恢复后的入口

### 9.3 回滚成功标准

- 源端 `http://127.0.0.1:8080/health` 正常
- 源端 `http://127.0.0.1:19080/health` 正常
- 正式对外入口重新可用，或至少业务恢复到迁移前状态

## 10. 后续升级路线

以下内容是迁移完成后的可选优化，不属于首发必须项：

- 如后续确认家宽非 CGNAT 且端口转发稳定，再评估升级为：
  - `Cloudflare DNS` + `Caddy` + 路由器 `80/443` 转发
- 如后续启用支付：
  - 在正式域名下验证 webhook 回调链路
- 如后续需要官网：
  - 将主域名用于官网
  - 保持 `api.<主域名>` 专职承载 Sub2API
- 如后续需要更强后台保护：
  - 单独设计后台保护方案
  - 不直接对整站加访问门槛

## 11. 可直接给 A 使用的子任务清单

- 校验源端 `stable` 健康状态
- 校验源端 `door-gateway` 健康状态
- 记录管理员、普通用户、API key、代理出流基线
- 打包 `runtime/stable`
- 生成迁移交付清单
- 冻结源端并做最终增量包
- 记录源端回滚命令

## 12. 可直接给 C 使用的子任务清单

- 安装 Docker Engine
- 安装 Docker Compose v2
- 安装 `cloudflared`
- 创建 `/srv/sub2api` 目录树
- 还原仓库与运行时目录
- 修改 Linux 专用 `.env`
- 修改 Linux 专用 `door-gateway.json`
- 安装专用 Mihomo/兼容内核
- 创建 `door-gateway` systemd 服务
- 创建 `cloudflared` systemd 服务
- 启动 `door-gateway`
- 启动 Sub2API Docker Compose
- 执行本地和外部健康检查

## 13. 人类最小操作清单

以下项目必须由 `B` 完成，其他事项尽量不分配给人类：

- 提供 Cloudflare 登录和授权
- 确认正式主域名
- 如需要，确认 Linux sudo 安装
- 如有条件，为 Linux 设置 DHCP 保留
- 从外部网络验证正式域名
- 邀请至少一位朋友完成真实访问验收

## 14. 附录：需要传输的文件与目录清单

必须传输：

- 仓库代码快照
- `runtime/stable/.env`
- `runtime/stable/data/`
- `runtime/stable/postgres_data/`
- `runtime/stable/redis_data/`
- `runtime/stable/door-gateway.json`
- `runtime/stable/door-workers/`

建议一并保留：

- 当前迁移前完整备份包
- 迁移交付清单
- 源端回滚命令记录
- Linux 端安装与运行日志

## 15. 文档完成判定

本文满足以下条件即视为可执行：

- 文件位于仓库根目录
- 文件为 Markdown 格式
- 每个阶段都按 `A/B/C` 拆分
- 没有混合角色步骤
- 没有使用 `TODO`、`待定`、`以后再说` 之类占位词
- 包含前置条件、迁移步骤、验证、回滚与后续升级路线
- A 与 C 可以只阅读自己的步骤开展工作
- B 只在真正需要人工介入时才被唤起
