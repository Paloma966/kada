# Kada

一个短链接管理与数据分析平台。支持多种登录方式、短链分组与标签、自定义域名、访问密码、过期时间、UTM 模板与点击数据看板。

后端使用 Go，前端使用 Next.js，点击事件通过 Kafka 异步处理。

## 功能

- 手机号 / 邮箱 / 微信登录（JWT）
- 短链接创建、编辑、删除、批量操作
- 自定义短码与自定义域名
- 访问密码、过期时间
- 文件夹、标签、工作区管理
- UTM 参数与模板
- 链接预览、二维码、CSV 导出
- 点击数据看板（总览、平台分布、每日趋势、访客明细）
- 开放 API Token

## 技术栈

| 层 | 技术 |
|----|------|
| 后端 | Go 1.26、Gin、pgx |
| 存储 | PostgreSQL 16、Redis 7 |
| 消息 | Kafka 3.8 |
| 前端 | Next.js 16、React 19、TypeScript、SWR、Tailwind |
| 部署 | Docker Compose、Nginx、systemd |

## 架构

分层为 Handler -> Service -> Infra，采用模块化单体 + 独立 Kafka worker。

点击事件经 Kafka 异步落库并累加计数，Kafka 不可用时降级为直写数据库。

## 快速开始

环境要求：Go 1.26+、Node 22+、Docker。

```bash
# 1. 准备环境变量，填入强随机 JWT_SECRET
cp .env.example .env

# 2. 启动全部服务
make docker-up

# 3. 运行数据库迁移
make db-migrate
```

本地开发：

```bash
make dev        # 后端 :8080
make dev-fe     # 前端 :3000
```

## 环境变量

| 变量 | 说明 |
|------|------|
| JWT_SECRET | 必填，强随机密钥 |
| POSTGRES_PASSWORD | 必填，生产需修改 |
| SMS_ACCESS_KEY_ID / SMS_ACCESS_KEY_SECRET | 阿里云短信 |
| SMS_SIGN_NAME / SMS_TEMPLATE_CODE | 短信签名与模板 |

## 常用命令

`make test`、`make build`、`make lint`、`make db-migrate`、`make docker-up`、`make docker-logs`

## 部署

GitHub Actions 自动执行 lint、测试、构建与部署。手动部署：

```bash
make deploy DEPLOY_HOST=root@your-server
make deploy-fe DEPLOY_HOST=root@your-server
```

## 目录

```text
backend/   Go 后端（cmd + internal）
frontend/  Next.js 前端
nginx/     反向代理配置
deploy/    部署脚本
docker-compose.yml
Makefile
```
