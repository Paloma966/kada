# Kada

[English](README.md) | **简体中文**

短链接管理与分析平台。支持链接文件夹与标签、自定义域名、访问密码、过期时间、UTM 模板、点击分析看板，以及一个能就你自己链接提问的 AI 助手。

后端用 Go 编写，前端用 Next.js，点击事件经 Kafka 异步处理，AI 助手是位于 Go 网关之后的独立 Python 服务。

## 功能

- 手机号 + 短信验证码登录（JWT），带图形验证码与按手机号 / 按 IP 的发送限流
- 短链接的创建、编辑、删除与批量管理
- 自定义短码与自定义域名
- 访问密码与过期时间
- 文件夹、标签与工作区管理
- UTM 参数与 UTM 模板
- 链接预览、二维码与 CSV 导出
- 点击分析看板（总览、平台分布、每日趋势、访客明细）
- 开放 API 令牌
- AI 助手（基于项目知识库的 RAG，工具调用以当前登录用户的身份执行）
- 明暗两套主题

## 技术栈

| 层级 | 技术 |
|-------|------------|
| 后端 | Go 1.26、Gin、GORM |
| 存储 | PostgreSQL 16（知识库使用 pgvector）、Redis 7 |
| 消息队列 | Kafka 3.8 |
| 前端 | Next.js 16、React 19、TypeScript、SWR、Tailwind |
| AI 服务 | Python 3.11、FastAPI、LangChain、DeepSeek + 阿里云百炼 |
| 部署 | Docker Compose、Nginx、systemd、GitHub Actions |

## 架构

分层为 Handler -> Service -> Infra，采用模块化单体，另有一个独立的 Kafka worker。

点击事件经 Kafka 异步落库并累加计数；Kafka 不可用时降级为直接写入数据库。

数据库表结构由 `backend/internal/domain/entity` 下的 GORM 模型定义，由 `backend/cmd/migrate`
（AutoMigrate）应用。仓库里没有 SQL 迁移文件：结构体是唯一事实来源，AutoMigrate 只补齐缺失的
表 / 列 / 索引，因此可以安全地重复执行。

AI 助手是独立的 Python 服务（`backend/ai`），从不对外暴露：Go 网关校验调用方身份后，把
`/api/ai/*` 反向代理给它。见 [backend/ai/README.md](backend/ai/README.md) 与
[docs/ai-deployment.md](docs/ai-deployment.md)。

## 快速开始

环境要求：Go 1.26+、Node 22+、Docker。

```bash
cp .env.example .env
docker compose up -d
```

本地开发：

```bash
cd backend && go run ./cmd/migrate/         # 仅应用表结构，API 启动时会自动应用
cd backend && go run ./cmd/server/main.go   # 8080
cd frontend && npm run dev                  # 3000
```

## 环境变量

| 变量 | 说明 |
|----------|-------------|
| JWT_SECRET | 必填，强随机密钥 |
| POSTGRES_PASSWORD | 必填，生产环境必须修改 |
| DB_AUTO_MIGRATE | 是否让 API 服务在启动时应用表结构（默认 true；使用 cmd/migrate 时设为 false） |
| SMS_ACCESS_KEY_ID / SMS_ACCESS_KEY_SECRET | 阿里云短信。生产环境必填：手机号 + 短信验证码是唯一的登录方式 |
| SMS_SIGN_NAME / SMS_TEMPLATE_CODE | 阿里云 PNVS 控制台为你分配的短信签名与模板 |
| AI_BASE_URL / AI_INTERNAL_SECRET | Go 网关在何处找到 AI 服务，以及使其只接受网关请求的共享密钥 |
| DEEPSEEK_API_KEY / aliyun | 仅 AI 服务使用（对话模型密钥，以及以字面量名 `aliyun` 存放的百炼 embedding 密钥） |

生产环境下这些值来自 GitHub 仓库 secrets，由部署任务写入 `/opt/kada/ai/ai.env` 与
`/opt/kada/backend/.env`；见 [docs/ai-deployment.md](docs/ai-deployment.md)。

## 目录结构

```text
backend/   Go 后端（cmd + internal），Python AI 服务在 backend/ai
frontend/  Next.js 前端
docs/      设计、贡献与代码风格文档
nginx/     反向代理配置
deploy/    部署脚本
docker-compose.yml
```
