# VPS 迁移运行手册

## 当前状态

2026-08-01 已完成东京 VPS 的系统基线、代码部署和一次生产数据快照恢复。
本次没有修改 Cloudflare、DNS 或正式 API 域名，也没有停止家中 Linux 的 stable 服务。

VPS 当前处于待切换状态：

- PostgreSQL 和 Redis 容器运行且健康；
- Sub2API 已短暂启动并通过 `/health` 检查，随后主动停止；
- `sub2api-stable.service` 已安装，但保持 `disabled/inactive`；
- UFW 只开放 SSH，`80/tcp`、`443/tcp` 和 `8080/tcp` 均未对公网开放；
- 应用只绑定 `127.0.0.1:8080`；
- Cloudflare 和正式 API 域名保持原状。

不要在旧实例仍提供生产服务时启动 VPS 上的 Sub2API。应用启动后会立即运行令牌刷新检查，并启动邮件队列、定时测试和监控等后台任务；即使只运行几十秒也可能触碰上游账号状态。只有在旧实例已冻结后台任务，或明确接受这些副作用时，才能启动 VPS 应用。

## 资源清单

| 项目 | 当前值 |
| --- | --- |
| VPS 公网 IPv4 | `43.165.188.95` |
| 操作系统 | Ubuntu Server 24.04.4 LTS |
| 内核 | `6.8.0-136-generic` |
| 系统盘 | 59 GiB 可用文件系统 |
| 内存 / Swap | 3.6 GiB / 1.9 GiB |
| Docker | 29.1.3 |
| Docker Compose | 2.40.3 |
| 部署提交 | `80eb2d0be16b14c7c5be5bc7792e9e243ea65a53` |
| Sub2API 版本 | 0.1.169 |
| PostgreSQL 镜像 | `postgres:18-alpine` |
| Redis 镜像 | `redis:8-alpine` |

VPS 路径：

```text
/srv/sub2api/repo
/srv/sub2api/runtime/stable
/srv/sub2api/runtime/backups/vps-migration-20260801-184710
```

本机 SSH 私钥路径为 `/home/lim/.ssh/sub2api_tokyo_ed25519`。只记录路径，不把私钥内容复制到仓库或运行手册。

## 已完成的安全基线

- 已安装全部系统安全更新和当前通用内核；
- 使用非 root 用户 `ubuntu` 管理；
- SSH 禁止 root 登录、密码登录和键盘交互登录，只接受密钥；
- UFW 默认拒绝入站，目前只允许 OpenSSH；
- Fail2ban、Docker 和 unattended-upgrades 已启用；
- PostgreSQL、Redis 和 Docker API 没有宿主机公网端口映射；
- Sub2API 的宿主机端口固定为 `127.0.0.1:8080`。

SSH 暂未限制为固定公网来源。家中公网出口不是已确认的固定地址，在没有 VPN 或稳定管理入口前直接限制来源可能导致管理失联。

待切换阶段不要执行 `./scripts/sub2api-local systemd install`。该命令会自动 enable 并 restart `sub2api-stable.service`，从而立即启动应用。VPS 上的 unit 已单独安装并保持禁用，不需要重复运行安装命令。

## 已迁移内容

- 本地 `main` 的当前工作树及 Git 元数据；
- 与当前运行镜像一致的 `sub2api-local:stable` 镜像；
- PostgreSQL 自定义格式逻辑备份；
- Redis RDB 快照；
- `runtime/stable/data` 中的配置和模型定价文件，不包含历史日志；
- stable 环境变量文件，并针对 4 GiB VPS 调整资源参数。

未迁移内容：

- `/srv/sub2api/runtime/backups` 中的历史备份；
- dev 环境、前端 `node_modules` 和其他开发缓存；
- egress-control 服务及其配置；
- Cloudflare Tunnel、Cloudflare DNS 和任何域名记录。

VPS 的 `UPDATE_PROXY_URL` 已清空。数据库中已有的账号级代理记录按快照原样保留，但 VPS 没有对应的 egress 固定出口，依赖这些代理记录的账号在正式切换前必须单独验证或迁移出口方案。

## 正式切换阻塞项

除最终数据同步外，当前还有以下配置阻塞项。任意一项未处理时都不能切换正式域名：

1. `PAYMENT_WEBUI_BASE_URL` 仍指向家中局域网地址，VPS 无法直接访问。正式切换前必须改成 VPS 可达的正式服务地址，或明确禁用依赖它的 true-refresh 流程，并验证支付/刷新链路。仅清空变量不够，当前代码在空值时仍有局域网默认地址。
2. URL 访问策略仍沿用可信内网的宽松配置：allowlist 关闭、允许明文 HTTP、允许私有地址。公网直连前应以启用 allowlist、禁止明文 HTTP、禁止私有地址为默认目标，并填写实际需要的上游主机；如果业务确实需要私网或 HTTP 集成，必须改为明确、最小范围的例外并单独验证，不能保留全局宽松模式。
3. 数据库中的账号级代理记录依赖家中 egress-control。必须迁移所需出口、改为 VPS 可达的代理，或确认相关账号不参与 VPS 调度。
4. Nginx/Caddy、可信公网 TLS 证书、SSE 配置、腾讯云安全组和 UFW 的 80/443 规则尚未配置。这些内容必须在 DNS 变更前通过 `curl --resolve` 验证。

## 数据验证结果

逻辑恢复成功并恢复 95 张 public 表。以下关键表在快照恢复后与本机一致：

| 表 | 行数 |
| --- | ---: |
| `accounts` | 2786 |
| `users` | 119 |
| `api_keys` | 137 |
| `proxies` | 172 |
| `settings` | 881 |
| `schema_migrations` | 243 |

Redis 键包含 TTL，恢复后会自然过期，因此不能用稍后时刻的 `DBSIZE` 要求完全相等。迁移完整性以 RDB 校验成功、Redis 启动加载成功和 PostgreSQL 关键数据一致为准。

## 日常检查

```bash
SSH_KEY=/home/lim/.ssh/sub2api_tokyo_ed25519
VPS=ubuntu@43.165.188.95

ssh -i "$SSH_KEY" "$VPS" 'sudo ufw status verbose'
ssh -i "$SSH_KEY" "$VPS" 'docker ps -a'
ssh -i "$SSH_KEY" "$VPS" \
  'systemctl is-enabled sub2api-stable.service; systemctl is-active sub2api-stable.service'
```

当前预期是 PostgreSQL/Redis 为 `healthy`，Sub2API 为 `exited`，systemd 单元为 `disabled/inactive`。

VPS 应用启动不是无副作用的 smoke test。只有在旧实例已冻结后台任务，或明确接受令牌刷新、定时测试等任务立即运行时，才能执行：

```bash
ssh -i "$SSH_KEY" "$VPS" \
  'cd /srv/sub2api/repo && ./scripts/sub2api-runtime-compose stable up -d sub2api'
ssh -i "$SSH_KEY" "$VPS" 'curl -fsS http://127.0.0.1:8080/health'
ssh -i "$SSH_KEY" "$VPS" \
  'cd /srv/sub2api/repo && ./scripts/sub2api-runtime-compose stable stop sub2api'
```

## 正式切换窗口

以下步骤只能在另行确认的迁移窗口执行，不应提前修改域名：

1. 记录旧实例最后可用状态，并再次制作独立 PostgreSQL 逻辑备份。
2. 进入短暂写入冻结或停止旧应用的后台任务，避免最终快照之后继续产生写入。
3. 重新导出 PostgreSQL；Redis 只作为缓存迁移，必要时重新导出 RDB。
4. 在 VPS 清空目标业务库后恢复最终快照，再次核对关键表和迁移版本。
5. 逐项解决“正式切换阻塞项”，再执行 `sudo systemctl enable --now sub2api-stable.service` 启动 VPS 的 Sub2API，并验证健康、管理登录、固定令牌、支付/刷新链路、代表性上游请求和重启恢复。
6. 配置 Nginx 或 Caddy、可信公网 TLS 证书和 SSE 长流参数；证书不能使用仅供 Cloudflare 代理回源的 Origin CA。
7. 在腾讯云安全组和 UFW 中同步开放 `443/tcp`，仅为签发证书及跳转开放 `80/tcp`。
8. 使用 `curl --resolve` 或等效方法在不改 DNS 的情况下验证域名、证书、SSE 和超时配置。
9. 确认 systemd、容器健康检查、日志和回退记录都正常。
10. 得到明确授权后，仅修改正式 API 子域名到 VPS，并设为 DNS-only；不要修改其他 DNS 记录。
11. 观察错误率、长流、CPU、内存、磁盘和日志；异常时把 API 记录恢复到原 Tunnel 路由，并保留旧实例作为短期回退来源。

正式切换前还必须处理账号级 egress 依赖。不能假设把应用和数据库搬到海外 VPS 后，原先指向家中 `egress-control` 的代理记录会自动可用。
