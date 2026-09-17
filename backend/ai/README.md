# Kada AI 服务

Kada 平台的 AI 助手后端，独立部署的 Python FastAPI 服务（LangChain + DeepSeek）。
它不直接暴露公网：所有请求由 Go 网关（`internal/handler/ai`）认证后转发进来，
Python 只监听 `127.0.0.1`。

## 架构

```
浏览器 → Go 网关(/api/ai/*, JWT 认证) → 本服务(/v1/*, 仅内网) → DeepSeek
                                              ├─ PostgreSQL：会话记录（真相源）
                                              └─ Redis：会话热缓存（cache-aside）
```

- Go 网关从 JWT 取出 `user_id`，注入 `X-Kada-User-ID` 头后转发；**客户端伪造的头会被覆盖**
- 流式对话走 SSE：`token`（逐字）/ `done` / `error` 三种事件

## 快速开始

前置：PostgreSQL 16（已启用）、Redis 已运行，`DEEPSEEK_API_KEY` 已设置。

```bash
pip install -r requirements.txt
cd backend/ai
python -m app.main          # 监听 127.0.0.1:8000
```

## 环境变量

| 变量 | 说明 | 默认 |
| --- | --- | --- |
| `DEEPSEEK_API_KEY` | DeepSeek API 密钥（必填） | 无 |
| `DASHSCOPE_API_KEY` | 阿里云 DashScope（RAG embedding 预留） | 无 |
| `PostgreSQL_URL` | SQLAlchemy 异步连接串 | `postgresql+asyncpg://kada:kada@127.0.0.1:5432/kada_ai` |
| `REDIS_URL` | Redis 连接串 | `redis://127.0.0.1:6379/0` |

其余配置（模型名、缓存 TTL、RAG 预置项）见 `app/config.py`。

## 接口契约（仅内网，Go 网关透传）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/v1/chat` | SSE 流式对话，请求 `{conversation_id?, message}` |
| GET | `/v1/conversations` | 当前用户的会话列表 |
| GET | `/v1/conversations/{id}/messages` | 某会话的历史消息 |
| DELETE | `/v1/conversations/{id}` | 删除会话 |
| GET | `/healthz` | 健康检查 |

所有接口通过 `X-Kada-User-ID` 头区分用户（由 Go 网关注入）。

## 存储

- **PostgreSQL**（SQLModel ORM）：`ai_conversations` / `ai_messages` 表，会话记录是真相源
- **Redis**：缓存每个会话最近的消息，TTL 1 小时，读多写少的 cache-aside 模式

## 已知规划

- RAG 知识库问答：向量存储计划用 PostgreSQL 的 **pgvector** 扩展（不引入独立 Milvus），
  配置已预留 `EMBEDDING_MODEL` / `DASHSCOPE_API_KEY` / `DOC_DIR` 等
