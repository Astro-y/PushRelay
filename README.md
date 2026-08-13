<p align="center">
  <img src="web/public/brand/pushrelay-app-icon.png" width="96" alt="PushRelay Logo" />
</p>

<h1 align="center">PushRelay</h1>

<p align="center">
  自托管的消息中转、规则路由与定时通知平台
</p>

<p align="center">
  <a href="https://github.com/Astro-y/PushRelay/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/Astro-y/PushRelay/ci.yml?branch=main&label=CI&logo=github" alt="CI" /></a>
  <img src="https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white" alt="Go 1.25" />
  <img src="https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=111827" alt="React 19" />
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-Apache--2.0-blue.svg" alt="Apache 2.0 License" /></a>
</p>

<p align="center">
  <a href="#features">功能</a> ·
  <a href="#channels">渠道</a> ·
  <a href="#workflow">工作方式</a> ·
  <a href="#deployment">部署</a> ·
  <a href="#development">本地开发</a> ·
  <a href="#security">安全</a>
</p>

PushRelay 接收统一 Webhook，根据安全、可视化的规则匹配事件，并通过消息模板异步投递到微信、飞书、Telegram 等渠道。它同时提供持久队列、失败重试、执行日志和分钟级定时调度，适合在个人服务器、NAS 或 1Panel 上单实例部署。

> [!TIP]
> 让只支持通用 Webhook 的服务，也能把一条事件按规则转换并发送到多个通知渠道。

<a id="features"></a>

## 功能亮点

- 📥 **统一入口**：接受 `GET`、`POST`、`PUT`、`PATCH`，支持 JSON、表单和纯文本载荷。
- 🧭 **可视化路由**：支持嵌套 AND/OR、字段比较、包含、正则、数值范围和集合匹配。
- 🧩 **灵活编排**：渠道保存凭据，模板描述消息结构，目标组组合多个投递目标。
- 📬 **可靠投递**：SQLite WAL 持久队列、at-least-once 语义、退避重试、死信与手动重投。
- 🗓️ **定时通知**：支持单次、日、周、月、年和标准 5 段 Cron，可独立设置 IANA 时区。
- 🔎 **完整日志**：串联事件、规则、投递状态、响应码、耗时、重试次数和错误摘要。
- 🔐 **管理安全**：Argon2id 密码、数据库 Session、CSRF、TOTP 两步验证及一次性恢复码。
- 🔑 **外部登录**：支持 Pocket ID OIDC Discovery、Authorization Code、PKCE、state 和 nonce。
- 🛡️ **数据保护**：渠道凭据、TOTP Secret 和持久化载荷使用 AES-256-GCM 加密。
- 📦 **简单部署**：React 生产资源内嵌到 Go，可使用单容器、单进程、单端口运行。

### 适用场景

| 场景 | 示例 |
| --- | --- |
| 监控告警 | 将 Uptime、NAS、服务器或业务告警分发到值班渠道 |
| CI/CD 通知 | 按仓库、分支或构建状态路由发布结果 |
| 跨渠道中转 | 把只有通用 Webhook 的系统转换为飞书、企业微信或 Telegram 消息 |
| 周期提醒 | 按时区发送日报、周报、账单、证书到期或维护提醒 |

<a id="channels"></a>

## 支持渠道

| 渠道 | 支持内容 |
| --- | --- |
| 钉钉群机器人 | 文本、Markdown、ActionCard、加签 |
| 企业微信群机器人 | 文本、Markdown |
| 企业微信应用 | Access Token 与应用消息 |
| 飞书群机器人 | 文本、富文本、签名校验 |
| Telegram Bot | 文本、Markdown、HTML |
| Discord Webhook | 内容与 Embed |
| Bark | 标题、正文、分组、声音及高级参数 |
| 通用 Webhook | GET/POST、自定义 Header、Query 和 JSON Body |

> “微信”指企业微信群机器人和企业微信应用，不包含个人微信非官方协议。

<a id="workflow"></a>

## 工作方式

| ① 触发 | ② 路由与编排 | ③ 可靠投递 |
| --- | --- | --- |
| Webhook / 定时任务 | 事件 → 规则匹配 → 目标组 | 渠道 + 模板 → 持久队列 |

- **发送成功**：投递到企业微信、飞书、Telegram 等目标渠道并记录结果。
- **发送失败**：按退避策略自动重试，最终失败进入死信并保留完整日志。

<a id="deployment"></a>

## 部署

| 方式 | 适合场景 | 运行形态 |
| --- | --- | --- |
| **Docker Compose（推荐）** | 快速部署与更新 | 单容器、单端口、命名卷持久化 |
| **Linux 原生** | 不使用 Docker 的服务器 | systemd 托管单个 Go 进程 |

> [!IMPORTANT]
> `APP_ENCRYPTION_KEY` 与 SQLite 数据需要一起备份。丢失或更换密钥后，已加密的渠道凭据和消息正文将无法恢复。

### Docker Compose（推荐）

需要提前安装 Docker Engine 和 Docker Compose 插件。仓库内置的 `compose.yaml` 会直接拉取 GHCR 镜像，不需要在服务器上安装 Go、Node.js 或 pnpm：

```yaml
services:
  pushrelay:
    image: ghcr.io/astro-y/pushrelay:latest
    container_name: pushrelay
    restart: unless-stopped
    environment:
      HTTP_ADDR: :4426
      DATABASE_PATH: /data/pushrelay.db
      WEB_ORIGIN: https://push.example.com
      APP_ENCRYPTION_KEY: REPLACE_WITH_32_BYTE_BASE64_KEY
    volumes:
      - pushrelay-data:/data
    ports:
      - "127.0.0.1:4426:4426"

volumes:
  pushrelay-data:
```

启动并查看初始化 Token：

```bash
docker compose pull
docker compose up -d
docker compose logs -f pushrelay
```

服务绑定宿主机 `127.0.0.1:4426`，SQLite 数据保存在 Docker 命名卷 `pushrelay-data`。首次启动日志会输出一次性 `setup_token`，打开访问地址并输入该 Token 即可创建管理员。

使用域名访问时，请把 Compose 中的 `WEB_ORIGIN` 改为浏览器实际访问的完整地址，例如 `https://push.example.com`。Pocket ID 登录完成后会返回该地址。

常用运维命令：

```bash
docker compose logs -f pushrelay   # 实时查看服务日志，按 Ctrl+C 退出查看
docker compose restart pushrelay   # 重启 PushRelay 容器
docker compose pull                # 拉取最新版本镜像
docker compose up -d               # 后台启动；拉取新镜像后执行可完成更新
docker compose down                # 停止并移除容器，不会删除 pushrelay-data 数据卷
```

> [!NOTE]
> 第一次发布镜像后，需要在 GitHub 的 Package settings 中把 `pushrelay` 容器包设为 Public。公开后服务器无需登录 GHCR 即可拉取。

<details>
<summary><strong>镜像标签与自动发布</strong></summary>

`.github/workflows/publish-container.yml` 使用 GitHub Actions 构建并发布 `linux/amd64`、`linux/arm64` 镜像：

- 推送到 `main`：发布 `latest` 和 `sha-xxxxxxx`。
- 推送 `v1.2.3` 标签：发布 `1.2.3`、`1.2` 和对应的 SHA 标签。
- 在 Actions 页面手动运行：按当前分支或标签重新构建镜像。

</details>

<details>
<summary><strong>展开 Linux 原生部署步骤</strong></summary>

需要提前安装 Go 1.25、Node.js 22.13 或更高版本、pnpm 和 Git。

#### 1. 下载并配置

```bash
git clone https://github.com/Astro-y/PushRelay.git
cd PushRelay
cp .env.example .env
openssl rand -base64 32
nano .env
```

将 `.env` 修改为：

```dotenv
HTTP_ADDR=127.0.0.1:4426
DATABASE_PATH=/opt/pushrelay/data/pushrelay.db
APP_ENCRYPTION_KEY=上一步生成的Base64密钥
WEB_ORIGIN=https://push.example.com
TRUSTED_PROXY_CIDRS=
WORKER_CONCURRENCY=8
```

`WEB_ORIGIN` 必须与浏览器访问地址完全一致。部署后请单独备份 `APP_ENCRYPTION_KEY`，不要随意更换。

#### 2. 构建

```bash
corepack enable
cd web
pnpm install --frozen-lockfile
pnpm run build
cd ..
mkdir -p internal/webui/dist
cp -a web/dist/. internal/webui/dist/
go build -trimpath -ldflags="-s -w" -o pushrelay ./cmd/server
```

#### 3. 安装 systemd 服务

```bash
sudo useradd --system --home /opt/pushrelay --shell /usr/sbin/nologin pushrelay
sudo install -d -o pushrelay -g pushrelay -m 0750 /opt/pushrelay/data
sudo install -o root -g root -m 0755 pushrelay /opt/pushrelay/pushrelay
sudo install -o root -g root -m 0644 openapi.yaml /opt/pushrelay/openapi.yaml
sudo install -o root -g pushrelay -m 0640 .env /opt/pushrelay/.env
```

创建 `/etc/systemd/system/pushrelay.service`：

```ini
[Unit]
Description=PushRelay notification relay
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=pushrelay
Group=pushrelay
WorkingDirectory=/opt/pushrelay
EnvironmentFile=/opt/pushrelay/.env
ExecStart=/opt/pushrelay/pushrelay
Restart=on-failure
RestartSec=5
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/pushrelay/data

[Install]
WantedBy=multi-user.target
```

启动并查看日志：

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now pushrelay
sudo systemctl status pushrelay
sudo journalctl -u pushrelay -f
```

首次启动日志会输出一次性 `setup_token`。打开访问地址，输入该 Token 创建管理员。

</details>

<a id="development"></a>

## 本地开发

复制根目录配置文件并编辑：

```bash
cp .env.example .env
openssl rand -base64 32
nano .env
```

本地 `.env` 至少需要：

```dotenv
HTTP_ADDR=:4426
DATABASE_PATH=./data/pushrelay.db
APP_ENCRYPTION_KEY=上一步生成的Base64密钥
WEB_ORIGIN=http://localhost:5173
```

启动后端：

```bash
set -a
source .env
set +a
go run ./cmd/server
```

另开一个终端启动前端：

```bash
cd web
pnpm install --frozen-lockfile
pnpm run dev
```

打开 `http://localhost:5173`。

## Webhook 接入

每个入口会生成独立的不可猜测 URL：

```text
POST https://push.example.com/hooks/{token}
```

示例请求：

```bash
curl -X POST "https://push.example.com/hooks/your-token" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: build-2026-001" \
  -d '{
    "title": "Build completed",
    "status": "success",
    "repository": "Astro-y/PushRelay"
  }'
```

成功后返回 `202 Accepted`：

```json
{
  "event_id": "...",
  "status": "accepted",
  "duplicate": false,
  "matched_rules": ["..."]
}
```

### HMAC 签名

开启 HMAC 后，请求方需要发送：

- `X-Push-Timestamp`: 当前 Unix 秒。
- `X-Push-Signature`: `sha256=<hex>`。

签名原文：

```text
timestamp + "." + raw_body
```

服务允许 5 分钟时间误差。HMAC Secret 只在创建入口或轮换令牌时完整显示一次。

## 模板上下文

模板可以读取：

- `body`
- `raw_body`
- `headers`
- `query`
- `form`
- `meta`

示例：

```json
{
  "msgtype": "text",
  "text": {
    "content": "{{ get \"body.message\" | default \"收到新事件\" }}"
  }
}
```

内置函数包括 `get`、`default`、`toJSON`、`truncate`、`dateInZone` 和 `urlquery`。模板使用 `missingkey=error`，不允许执行任意代码、访问文件或主动发起网络请求。

## 配置

启动所需环境变量：

| 变量 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `APP_ENCRYPTION_KEY` | 是 | 无 | 32 字节 Base64 或 64 位十六进制密钥 |
| `HTTP_ADDR` | 否 | `:4426` | HTTP 监听地址 |
| `DATABASE_PATH` | 否 | `./data/pushrelay.db` | SQLite 文件路径 |
| `WEB_ORIGIN` | 否 | `http://localhost:5173` | 浏览器公开访问地址，用于 CORS、CSRF 与 Pocket ID 登录返回 |
| `SETUP_TOKEN` | 否 | 自动生成 | 可选的固定初始化 Token |
| `WORKER_CONCURRENCY` | 否 | `8` | 后台投递 Worker 数量，范围 1–64 |
| `TRUSTED_PROXY_CIDRS` | 否 | 空 | 预留的可信代理 CIDR 列表 |

默认时区、正文保留天数、元数据保留天数、私网 Webhook 策略及 Pocket ID 配置均可在管理后台修改，无需写入 `.env`。

<a id="security"></a>

## 安全说明

- 默认阻止通用出站 Webhook 访问环回、私网、链路本地和云元数据地址。
- 入站 Webhook 可配置 URL Token、HMAC-SHA256、IP/CIDR 白名单和 24 小时幂等键。
- 管理 Session 使用 HttpOnly、SameSite Cookie；修改操作需要 CSRF Token。
- 密码使用 Argon2id；渠道凭据和持久化消息正文使用 AES-256-GCM 加密。
- 日志不会返回完整渠道 Secret、HMAC Secret 或 Pocket ID Client Secret。
- 请同时备份 SQLite 数据卷和 `APP_ENCRYPTION_KEY`，缺少其中任意一个都无法完整恢复服务。

## 开发与验证

```bash
go test ./...
go vet ./...

cd web
pnpm run typecheck
pnpm run lint
pnpm run build
pnpm exec playwright install chromium
pnpm run test:e2e
```

CI 会验证 Go 测试、前端生产构建、Playwright 端到端流程和单容器 Docker 镜像构建。

## 项目结构

```text
cmd/server/             Go 服务入口
internal/app/           HTTP API、认证、队列 Worker 与调度器
internal/channels/      通知渠道适配器
internal/rules/         可视化规则条件树
internal/schedule/      周期与 Cron 调度
internal/store/         SQLite 持久化
internal/templatex/     安全消息模板
internal/webui/         Go 内嵌 React 生产资源
web/                    React、TypeScript、Vite、shadcn/ui
openapi.yaml            OpenAPI 契约
```

## 当前范围

- 单管理员、单实例、SQLite。
- 不包含个人微信非官方协议、邮件、短信、附件转发或入站 IM 对话。
- 多实例部署需要迁移到 PostgreSQL 和外部队列后再启用。

## 反馈与贡献

欢迎通过 [Issues](https://github.com/Astro-y/PushRelay/issues) 报告问题或提出建议，也欢迎提交 [Pull Request](https://github.com/Astro-y/PushRelay/pulls)。提交代码前请先运行“开发与验证”中的检查命令。

## 致谢

项目的渠道、模板与目标组思路受到 [MoePush](https://github.com/beilunyang/moepush) 启发。PushRelay 使用 Go 从头实现，并在此基础上加入 Webhook 中转、规则路由、持久队列和定时调度。

## License

本项目使用 [Apache License 2.0](LICENSE)。
