# kirocli2

一个基于 Gin 的 OpenAI/Anthropic 兼容网关服务，将请求转发到 Amazon Q，并支持多账号 token 池轮换。

## 功能

- OpenAI 兼容接口：`/v1/chat/completions`
- Anthropic 兼容接口：`/v1/messages`、`/v1/messages/count_tokens`
- 模型列表接口：`/v1/models`
- Bearer / `x-api-key` 鉴权
- 多 refresh token 并发池与自动刷新
- Docker / Docker Compose 部署

## 接口概览

- `POST /v1/chat/completions`
- `POST /v1/messages`
- `POST /v1/messages/count_tokens`
- `GET /v1/models`
- `POST /debug/token`（无鉴权）
- `POST /debug/anthropic2q`（无鉴权）
- `GET /admin`（管理面板页面，无需鉴权）

## 运行前准备

需要 Go 1.24+，或直接使用 Docker。

### 1) 启动方式（推荐：先启动，再在管理面板配置）

现在支持**最少环境变量启动**：

- 默认会自动创建并使用 `data/config.json`
- 若未设置 `ADMIN_TOKEN` 和 `BEARER_TOKEN`，启动时会自动生成一次性 `Bootstrap admin token`（打印在日志中）
- 首次登录后请尽快在管理面板中更新 `ADMIN_TOKEN`
- 进入 `/admin` 后可直接在面板中填写 `OIDC_URL`、`AMAZON_Q_URL`、账号来源等配置，点击更新后会自动持久化到 `data/config.json`
- 手动录入账号会持久化到 `data/api_accounts.json`

### 2) 环境变量（可选）

| 变量名 | 是否必填 | 说明 |
| --- | --- | --- |
| `CONFIG_PATH` | 否 | 配置文件路径（默认 `data/config.json`） |
| `ADMIN_TOKEN` | 否 | 管理面板 token（不填将自动生成 bootstrap token） |
| `BEARER_TOKEN` | 否 | 对外 API 鉴权 token（可在管理面板设置） |
| `OIDC_URL` | 否 | 使用 refresh token 换取 access token 的 OIDC 地址（可在管理面板设置） |
| `AMAZON_Q_URL` | 否 | Amazon Q 对话接口地址（可在管理面板设置） |
| `ACCOUNT_SOURCE` | 否 | 账号来源，`manual`（默认）/`csv`/`api` |
| `ACCOUNTS_CSV_PATH` | `ACCOUNT_SOURCE=csv` 时建议配置 | 账号 CSV 路径 |
| `ACCOUNT_API_URL` | `ACCOUNT_SOURCE=api` 时必填 | 账号池 API 地址 |
| `ACCOUNT_API_TOKEN` | `ACCOUNT_SOURCE=api` 时必填 | 账号池 API 密钥（`X-Passkey`） |
| `ACCOUNT_CATEGORY_ID` | 否 | 拉取 API 账号时的分类 ID（默认 `3`） |
| `ACTIVE_TOKEN_COUNT` | 否 | 启动时激活 token 数量（默认 `10`） |
| `MAX_REFRESH_ATTEMPT` | 否 | token 刷新重试次数（默认 `3`） |
| `PORT` | 否 | 服务端口（默认 `4000`） |
| `GIN_MODE` | 否 | `release` / `debug`（默认 `release`） |
| `PROXY_URL` | 否 | HTTP/HTTPS 代理地址 |
| `ADMIN_TOKEN` | 否 | 管理面板鉴权 token，不填时默认复用 `BEARER_TOKEN`（若两者都不填则自动生成 bootstrap token） |

> 仍可使用 `.env` 管理变量，但已不再要求必须预先配置完整变量。

### 3) CSV 账号文件（`ACCOUNT_SOURCE=csv`）

CSV 需要至少 4 列，且首行为表头，启用行第一列需为 `True`（当前实现大小写敏感，仅识别 `True`）：

`enabled,refresh_token,client_id,client_secret`

## 本地运行

```bash
go mod download
go run .
```

默认监听 `:4000`。

## 管理面板（支持云端部署）

服务内置管理面板，无需额外前端构建，容器/云端部署后可直接访问：

- 页面：`GET /admin`
- 状态：`GET /admin/api/status`
- 手动录入账号：`POST /admin/api/accounts`
- 手动测试账号：`POST /admin/api/accounts/test`
- 手动刷新活跃 token：`POST /admin/api/tokens/refresh`
- 运行时更新配置：`POST /admin/api/config`

鉴权方式（适用于 `/admin/api/*` 请求）：

- 请求头 `x-admin-token: <ADMIN_TOKEN>`
- 或 `Authorization: Bearer <ADMIN_TOKEN>`

若未设置 `ADMIN_TOKEN`，默认使用 `BEARER_TOKEN` 作为管理口令；若二者都没设置，则请使用启动日志里的 `Bootstrap admin token`。

## Docker 运行

### 方式一：Docker Compose

```bash
docker compose up -d --build
```

> 若你之前使用 `./resources` 挂载账号缓存，请迁移到 `./data`（新默认路径）。

### 方式二：Docker 手动构建

```bash
docker build -t kirocli2:latest .
docker run --rm -p 4000:4000 --env-file .env kirocli2:latest
```

## 自动构建 Docker 镜像（GitHub Actions）

仓库内已提供工作流：`.github/workflows/docker-image.yml`

- `pull_request` 到 `main`：仅执行镜像构建校验（不推送）
- `push` 到 `main`：自动构建并推送镜像到 `ghcr.io/<owner>/<repo>`
- 打 `v*` 标签：自动构建并推送版本标签镜像
- 支持手动触发 `workflow_dispatch`

如需推送到 GHCR，请确保仓库开启 `Packages` 权限（工作流已配置 `packages: write`）。
