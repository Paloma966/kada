# Kada — 智能短链接管理平台

> 一个前后端分离的短链接 SaaS 平台。支持手机号 / 邮箱 / 微信登录，短链接的创建、管理、分组、标签、自定义域名、访问密码、过期时间、UTM 模板；点击事件通过 Kafka 异步落库，并提供数据看板与访客分析。

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/)
[![Gin](https://img.shields.io/badge/Gin-1.12-008ECF?style=flat-square&logo=go&logoColor=white)](https://github.com/gin-gonic/gin)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?style=flat-square&logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7-DC382D?style=flat-square&logo=redis&logoColor=white)](https://redis.io/)
[![Kafka](https://img.shields.io/badge/Kafka-3.8-231F20?style=flat-square&logo=apachekafka&logoColor=white)](https://kafka.apache.org/)
[![Next.js](https://img.shields.io/badge/Next.js-16-000000?style=flat-square&logo=nextdotjs&logoColor=white)](https://nextjs.org/)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?style=flat-square&logo=docker&logoColor=white)](https://www.docker.com/)

---

## ✨ 功能特性

### 👤 用户与认证
- 手机号 + 验证码登录（自动注册）
- 邮箱 + 密码登录 / 注册
- 微信登录（预留）
- 基于 JWT（HS256）的无状态认证
- 开放 API Token（`kada_` 前缀，仅存储 SHA-256 哈希）

### 🔗 短链接
- 创建 / 编辑 / 删除 / 批量删除短链接
- 自定义短码或随机生成（12 位十六进制，48 bit 熵）
- 自定义域名（需 DNS TXT 验证所有权）
- 可选访问密码（bcrypt 加密）与过期时间
- 支持 UTM 参数（`utm_source/medium/campaign/term/content`）与 UTM 模板
- 移动端定向（`ios_url` / `android_url`）
- 链接预览（自动抓取 OG 元数据）、二维码生成、CSV 导出

### 📁 资源管理
- 文件夹、标签、工作区（多租户隔离）
- 链接批量打标签、按文件夹 / 标签 / 工作区 / 关键词检索

### 📊 点击数据分析
- 总链接数、总点击量概览
- 平台来源分布（微信 / QQ / 微博 / 小红书 / 浏览器）
- 近 30 天每日点击趋势
- 点击事件明细、独立访客列表

### 🛡️ 安全与稳定性
- 短链跳转前协议白名单校验（防 `javascript:` 存储型 XSS）
- 网页抓取 SSRF 防护（内网 / 保留地址黑名单 + DNS 复核）
- JWT 算法混淆防护与弱密钥启动校验
- Redis 滑动窗口限流 + 真实 IP 提取
- 验证码原子消费（防并发重放）与发送频控
- CSV 公式注入防护
- Kafka 点击流异步削峰 + 降级直写，保证点击不丢失

---

## 🏗️ 架构

Kada 采用**模块化单体 + 独立 Kafka Worker** 的架构：

```text
浏览器 / 客户端
      │
      ▼
┌───────────────────────────┐
│        Nginx (80/443)      │
│  /api/  /r/  → Go 后端     │
│  /        → Next.js 前端   │
└─────────────┬──────────────┘
              │
      ┌───────▼────────┐
      │   Go API 服务   │
      │  Handler 层      │
      │  Middleware      │
      │  Service 层      │
      └──┬─────────┬────┘
         │         │
  PostgreSQL    Redis
         │         │
         └──► Kafka ──► Go worker ──► PostgreSQL
```

关键设计：

- **三层分层**：`Handler → Service → Infra`，依赖通过 `main.go` 手工装配，便于测试与扩展。
- **异步削峰**：点击事件先写入 Kafka `clicks` topic，由独立 worker 消费并落库，跳转链路保持轻量。
- **降级兜底**：Kafka 不可用时自动回退直写数据库，保证点击数据不丢失。
- **缓存提速**：短码内容用 Redis Cache-Aside 缓存，热点跳转秒级响应。

---

## 🧰 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.26 · Gin 1.12 · pgx v5 |
| 存储 | PostgreSQL 16 · Redis 7 |
| 消息队列 | Apache Kafka 3.8（segmentio/kafka-go） |
| 认证 | golang-jwt/v5 (HS256) · bcrypt |
| 短信 | 阿里云 dypnsapi SDK |
| 前端 | Next.js 16 · React 19 · TypeScript · SWR · Tailwind CSS 4 |
| 部署 | Docker Compose · Nginx · systemd |
| CI/CD | GitHub Actions |
| 迁移 | golang-migrate |

---

## 📁 目录结构

```text
kada/
├── backend/                  # Go 后端
│   ├── cmd/
│   │   ├── server/           # API 进程
│   │   ├── worker/           # Kafka 消费进程
│   │   └── migrate/          # 数据库迁移
│   ├── config/               # 环境变量配置
│   ├── db/migrations/        # SQL 迁移文件
│   └── internal/
│       ├── domain/           # 领域模型 / 请求响应
│       ├── handler/          # HTTP 层（Gin）
│       ├── service/          # 业务逻辑层
│       ├── infra/            # 基础设施（DB/Redis/SMS/SSRF 防护…）
│       ├── middleware/       # JWT / 限流 / 真实 IP
│       └── mq/               # Kafka 消息封装
├── frontend/                 # Next.js 前端
│   └── src/
│       ├── app/              # 路由页面
│       ├── components/       # UI 组件
│       └── lib/              # api / auth / utils
├── nginx/                    # 反向代理配置
├── deploy/                   # 部署脚本（systemd / Docker）
├── docs/                     # 设计文档
├── docker-compose.yml        # 容器编排
├── Makefile                  # 常用命令
└── .github/workflows/        # CI/CD
```

---

## 🚀 快速开始

### 环境要求

- Go ≥ 1.26
- Node.js ≥ 22
- Docker & Docker Compose（推荐）

### 方式一：Docker Compose（推荐）

```bash
# 1. 准备环境变量
cp .env.example .env
# 编辑 .env，设置强随机 JWT_SECRET（openssl rand -hex 32）

# 2. 启动全部服务（Postgres / Redis / Kafka / API / Worker / 前端 / Nginx）
make docker-up

# 3. 运行数据库迁移
make db-migrate

# 4. 访问
#    前端:   http://localhost
#    API:    http://localhost/api/health
#    Swagger:（可选）
```

### 方式二：本地开发

```bash
# 后端
make dev            # 启动 Go API（:8080）

# 前端（另开终端）
make dev-fe         # 启动 Next.js 开发服务器（:3000）
```

> 本地开发需提前启动 PostgreSQL 与 Redis。可通过 `docker compose up -d postgres redis` 一键拉起。

---

## ⚙️ 环境变量

复制 `.env.example` 为 `.env` 并填写：

| 变量 | 必填 | 说明 |
|------|:---:|------|
| `JWT_SECRET` | ✅ | 强随机签名密钥，使用 `openssl rand -hex 32` 生成；生产禁止默认值 |
| `POSTGRES_PASSWORD` | ✅ | 数据库密码，生产必须修改 |
| `SMS_ACCESS_KEY_ID` | ⛔ | 阿里云短信 AccessKey |
| `SMS_ACCESS_KEY_SECRET` | ⛔ | 阿里云短信 AccessKey Secret |
| `SMS_SIGN_NAME` | ⛔ | 短信签名 |
| `SMS_TEMPLATE_CODE` | ⛔ | 短信模板 Code |

> 未配置短信时，验证码仅打印在日志（开发环境），生产环境绝不输出明文验证码。

---

## 🛠️ 常用命令

| 命令 | 说明 |
|------|------|
| `make dev` | 启动 Go 后端 |
| `make dev-fe` | 启动前端开发服务器 |
| `make build` | 编译后端二进制 |
| `make test` | 运行 Go 测试 |
| `make test-race` | 运行测试 + 竞态检测 |
| `make docker-up` | 启动全部 Docker 服务 |
| `make docker-logs` | 查看 Docker 日志 |
| `make db-migrate` | 运行数据库迁移 |
| `make db-create NAME=xxx` | 创建新的迁移文件 |
| `make lint` | Go 代码检查（go vet） |
| `make lint-ci` | 运行 golangci-lint |
| `make build-pkg` | 构建并打包（模拟 CI deploy） |
| `make release-dry-run` | 本地构建 Docker 镜像 |

---

## 🌐 API 概览

所有接口以 `/api` 为前缀，除登录 / 注册 / 健康检查外均需 `Authorization: Bearer <token>`。

### 认证

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/auth/send-sms-code` | 发送短信验证码 |
| POST | `/api/auth/login-by-phone` | 手机号登录 |
| POST | `/api/auth/login-by-email` | 邮箱登录 |
| POST | `/api/auth/register-by-email` | 邮箱注册 |
| GET/PATCH | `/api/me` | 获取 / 更新当前用户 |

### 短链接

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/links` | 链接列表（分页/搜索/筛选/排序） |
| POST | `/api/links` | 创建短链 |
| GET/PATCH/DELETE | `/api/links/:id` | 详情 / 更新 / 删除 |
| POST | `/api/links/batch-delete` | 批量删除 |
| POST | `/api/links/batch-tag` | 批量打标签 |
| GET | `/api/links/export` | 导出 CSV |
| POST | `/api/links/preview` | 抓取链接预览 |

### 数据看板

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/analytics/overview` | 总览统计 |
| GET | `/api/analytics/platforms` | 平台分布 |
| GET | `/api/analytics/daily` | 每日点击趋势 |
| GET | `/api/analytics/events` | 点击事件明细 |
| GET | `/api/analytics/customers` | 独立访客列表 |

### 资源管理

文件夹 `/api/folders`、标签 `/api/tags`、域名 `/api/domains`、UTM 模板 `/api/utm-templates`、工作区 `/api/workspaces`、API Token `/api/api-tokens`。

### 公开短链跳转

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/r/:code` | 短链重定向 |
| GET | `/r/:code/qrcode` | 生成二维码 |
| POST | `/r/:code/verify-password` | 校验密码后跳转 |
| POST | `/r/:code/click-action` | 记录引导页行为 |

---

## 📦 部署

### CI/CD（GitHub Actions）

- **CI**（push/PR 到 `main`）：golangci-lint、Go 测试（`-race`）、前端 ESLint、前端构建（`tsc` + `next build`）。
- **Deploy**（仅 `main` push）：构建 → 备份 → 运行数据库迁移 → 替换后端 → 健康检查（失败回滚）→ 替换前端 → 重启并验证。
- **Release**（打 tag `v*`）：构建并推送 backend / frontend 镜像到 GitHub Container Registry。

### 服务器部署

```bash
make deploy DEPLOY_HOST=root@your-server
make deploy-fe DEPLOY_HOST=root@your-server
```

也可使用 `deploy/setup.sh` 初始化服务器环境，或用 `deploy/kada-frontend.service` 通过 systemd 托管前端进程。

---

## 🧪 测试

```bash
make test           # Go 单元测试
make test-race      # 竞态检测
make lint-ci        # golangci-lint
```

前端：

```bash
cd frontend && npm test
```

---

## 📝 License

本项目采用私有 / 内部项目授权，未经授权请勿对外发布。

---

## 📄 更多文档

- 产品设计：`docs/superpowers/specs/`
- 技术方案：`docs/superpowers/plans/`
- 面试讲解：`docs/interview-kada.md`

---

由 [Paloma966](https://github.com/Paloma966) 维护 — Made with ❤️
