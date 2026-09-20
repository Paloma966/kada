# Kada AI 服务

Kada 平台的 AI 助手后端，独立部署的 Python FastAPI 服务（LangChain + DeepSeek）。
它不直接暴露公网：所有请求由 Go 网关（`internal/handler/ai`）认证后转发进来，
Python 只监听 `127.0.0.1`。

## 架构

```
浏览器 → Go 网关(/api/ai/*, JWT 认证) → 本服务(/v1/*, 仅内网) → DeepSeek
                                              ├─ PostgreSQL：会话记录（真相源）+ pgvector 知识库
                                              ├─ Redis：会话热缓存（cache-aside）
                                              └─ MCP 子服务(stdio) → 回调 Go /api/*（查统计、建短链）
```

- Go 网关从 JWT 取出 `user_id`，注入 `X-Kada-User-ID` 头后转发；**客户端伪造的同名头会被覆盖**
- 流式对话走 SSE：`token`（逐字）/ `done` / `error` 三种事件
- 单会话模式：每个用户只保留一个当前会话，重新开始会物理删除旧会话

## 快速开始

前置：PostgreSQL 16（带 pgvector 扩展）、Redis 已运行，密钥见下方环境变量。

```bash
pip install -r requirements.txt
cd backend/ai
python -m app.main          # 监听 127.0.0.1:8000
```

知识库文档入库（把 `docs/` 下的 md/txt 向量化写入 pgvector）：

```bash
python -m app.scripts.ingest_docs
```

## 容器部署（生产）

生产环境随仓库根目录的 Docker Compose 一起启动，不需要单独运行本服务：

1. 在根目录 `.env` 填入 `DEEPSEEK_API_KEY`、`aliyun`、`KADA_API_TOKEN`（见根目录 `.env.example`）
2. `docker compose up -d` 会按 `backend/ai/Dockerfile` 构建本服务；PostgreSQL 使用 pgvector 镜像，并在数据卷首次初始化时自动创建 `kada_ai` 库并启用 vector 扩展
3. 首次启动后入库知识库文档：
   ```bash
   docker compose exec ai python -m app.scripts.ingest_docs
   ```
4. `KADA_API_TOKEN` 要等平台首次启动后，在「设置 → API Token」里创建（明文仅显示一次），填入 `.env` 后执行 `docker compose restart ai`

容器内的连接地址由 compose 注入：数据库 `postgres:5432`、Redis `redis:6379`、MCP 回调 `http://backend:8080`；
Go 网关通过 `AI_BASE_URL=http://ai:8000` 找到本服务，nginx 已为 `/api/ai/` 关闭缓冲以保证 SSE 逐字输出。

> `deploy/postgres/initdb/` 下的建库脚本只在数据卷**首次初始化**时执行；在已有 PostgreSQL 数据卷上升级，需要手动执行一次该 SQL。

## 环境变量

只有密钥从环境变量读取，其余参数（模型名、连接串、缓存 TTL 等）的默认值写在 `app/config.py`。

| 变量 | 说明 | 默认 |
| --- | --- | --- |
| `DEEPSEEK_API_KEY` | DeepSeek API 密钥（必填） | 无 |
| `aliyun` | 阿里云 DashScope API Key，用于 RAG 的 embedding | 无 |
| `KADA_API_TOKEN` | Kada 长效 API Token，MCP 回调 Go 后端时认证用 | 无 |

> 注意 embedding 的环境变量名约定为 `aliyun`（与百炼账号对应），不是 `DASHSCOPE_API_KEY`。
> 密钥不要写进代码或提交到仓库。

## 接口契约（仅内网，Go 网关透传）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/v1/chat` | SSE 流式对话，请求 `{conversation_id?, message}` |
| GET | `/v1/session/current` | 当前用户最新会话及其消息，没有则返回空 |
| POST | `/v1/session/restart` | 删除当前用户所有旧会话，新建一个空会话 |
| GET | `/healthz` | 健康检查 |

所有接口通过 `X-Kada-User-ID` 头区分用户（由 Go 网关注入，Python 不信任客户端直传）。

## 存储

- **PostgreSQL**（SQLModel ORM）：`ai_conversations` / `ai_messages` 表，会话记录是真相源；
  删除会话时消息靠外键 `ON DELETE CASCADE` 连带删除
- **pgvector**（langchain-postgres）：`langchain_pg_*` 表存知识库切片与向量，RAG 检索 top-k 片段
- **Redis**：缓存每个会话最近的消息（默认 50 条、TTL 1 小时），读多写少的 cache-aside；
  Redis 不可用时自动降级为直接查库

## 工具能力

- 本地工具（`app/service/tools.py`）：查时间、加法等随进程运行的 `@tool`
- MCP 工具（`app/mcp_server/kada_server.py`）：以 stdio 子进程方式被拉起，
  回调 Go 后端查询短链总览、创建短链；MCP 连不上时 AI 降级为纯对话，不影响主流程
