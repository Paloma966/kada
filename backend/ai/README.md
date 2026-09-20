# Kada AI 服务

Kada 平台的 AI 助手后端，独立部署的 Python FastAPI 服务（LangChain + DeepSeek）。
它不直接暴露公网：所有请求由 Go 网关（`internal/handler/ai`）认证后转发进来，
Python 只监听 `127.0.0.1`。

## 架构

```
浏览器 → Go 网关(/api/ai/*, JWT 认证) → 本服务(/chat、/conversations/*, 仅内网) → DeepSeek
                                              ├─ PostgreSQL：会话记录（真相源）+ pgvector 知识库
                                              ├─ Redis：会话热缓存（cache-aside）
                                              ├─ 业务工具(进程内) → 带当前用户 JWT 调 Go /api/*（查统计、建短链）
                                              └─ MCP 子服务(stdio) → 预留：外部通用工具（搜索、文件等）
```

- Go 网关从 JWT 取出 `user_id`，注入 `X-Kada-User-ID` 头后转发；**客户端伪造的同名头会被覆盖**
- 本服务只接受**来自网关**的请求：网关代理时注入 `X-Internal-Secret`（值来自 `AI_INTERNAL_SECRET`），
  本服务据此确认调用方是网关 —— 这是 `X-Kada-User-ID` 有资格被当作身份断言的前提（见 `app/route/deps.py`）
- 业务工具以**当前登录用户**的身份执行：网关原样转发的 `Authorization` 被继续交给 Go 的 `/api/*`，
  本服务不解析、不校验、不缓存凭据，权限判定始终只有 Go 一处
- 流式对话走 SSE：`token`（逐字）/ `done` / `error` 三种事件
- 单会话模式：每个用户只保留一个当前会话，重新开始会物理删除旧会话

## 快速开始

前置：PostgreSQL 16（带 pgvector 扩展）、Redis 已运行，密钥见下方环境变量。

```bash
pip install -r requirements.txt
cd backend/ai
python -m app.main          # 监听 127.0.0.1:8000
```

本机直连调试时不必配 `AI_INTERNAL_SECRET`（为空即关闭来源校验，启动会打印一行提示）；
此时请求不带 `Authorization`，业务工具会明确回一句"没有携带登录凭据"。

知识库文档入库（把 `docs/` 下的 md/txt 向量化写入 pgvector）：

```bash
python -m app.scripts.ingest_docs
```

## 容器部署

### 整机一套（本地开发 / 单机全栈）

生产环境也可以随仓库根目录的 Docker Compose 一起启动：

1. 在根目录 `.env` 填入 `DEEPSEEK_API_KEY`、`aliyun`、`AI_INTERNAL_SECRET`（见根目录 `.env.example`）
2. `docker compose up -d` 会按 `backend/ai/Dockerfile` 构建本服务；PostgreSQL 使用 pgvector 镜像，并在数据卷首次初始化时自动创建 `kada_ai` 库并启用 vector 扩展
3. 首次启动后入库知识库文档：
   ```bash
   docker compose exec ai python -m app.scripts.ingest_docs
   ```

compose 会把同一个 `AI_INTERNAL_SECRET` 同时给 `backend` 和 `ai` 两个容器；数据库 `postgres:5432`、Redis `redis:6379`、
业务工具回调 `http://backend:8080`；Go 网关通过 `AI_BASE_URL=http://ai:8000` 找到本服务，
nginx 已为 `/api/ai/` 关闭缓冲以保证 SSE 逐字输出。

> `deploy/postgres/initdb/` 下的建库脚本只在数据卷**首次初始化**时执行。
> 业务工具不需要长效令牌，所以这里没有"启动后再去平台建 Token 并重启"这一步。

### 接入已有 PostgreSQL/Redis 的服务器（生产实际用法）

生产服务器上 Go API / 前端是 systemd 原生进程，nginx 是 Docker 容器，PostgreSQL 和 Redis 已经在本机运行。
这种环境下**不要**用根目录的 compose 起 ai：它的 `depends_on` 会连带拉起自己的 postgres 和 redis，
与已经在 5432/6379 上监听的实例撞端口。改用 `deploy/docker-compose.ai.yml`——它只包含 ai 一个服务，
没有 `depends_on`，因此不可能拉起别的容器。

首次部署（各执行一次）：

1. **数据库**：把 `deploy/setup-ai-db.sh` 拷到服务器上执行一次
   （`scp deploy/setup-ai-db.sh <host>:/tmp/ && ssh <host> "bash /tmp/setup-ai-db.sh"`）。
   它创建 `kada_ai` 库并在**这个库**（不是 Go 的 `kada` 库）里启用 vector 扩展，可重复执行。
   报 `pgvector is not available` 时：原生 PostgreSQL 装 `postgresql-16-pgvector` 并重启，
   Docker 则换成 `pgvector/pgvector:pg16` 镜像。
2. **连接串**：服务器上 `mkdir -p /opt/kada/ai`，把 `deploy/ai.env.example` 拷成 `/opt/kada/ai/ai.env` 并
   `chmod 600`（仓库不在服务器上就先从本地 `scp deploy/ai.env.example <host>:/opt/kada/ai/ai.env`），
   填入 `POSTGRES_URL` / `REDIS_URL` / `KADA_API_BASE` 这三个连接串即可。
   三个密钥（`DEEPSEEK_API_KEY` / `aliyun` / `AI_INTERNAL_SECRET`）不用手填：部署作业用
   `deploy/upsert-env.sh` 从 GitHub Secrets 写进去，缺任何一个都会让部署失败并说明是哪一个。
   `deploy/deploy-ai.sh` 在构建镜像前会再校验一次 `POSTGRES_URL`、`DEEPSEEK_API_KEY` 与 `aliyun`：
   否则会起一个启动正常、健康检查通过、但每个对话请求都在模型侧报错的空壳服务，比拒绝部署难查得多。
   其中 `POSTGRES_URL` 最不能省——`app/config.py` 的回退值是带猜测密码的开发 DSN，缺了它容器会在
   `init_db()` 里崩，而那时镜像已经构建完了。`REDIS_URL` / `KADA_API_BASE` 不校验：它们的默认值
   （`redis://127.0.0.1:6379/0`、`http://localhost:8080`）正是本机的实际情况，Redis 本身也是可选的。
3. **网关侧同一个密钥**：同样由 GitHub Secret 写进 `/opt/kada/backend/.env`，不需要手动对齐。
   两边不一致时本服务会对网关的请求回 401，AI 页面整体不可用——现在两边出自同一个值，不会再不一致。
4. **启动**：推 `main` 后 CI 自动部署（同步密钥 → 构建镜像 → 重启容器 → 探活 → 失败回滚到上一个镜像）；
   手动等价命令是服务器上的 `bash /opt/kada/ai/deploy-ai.sh`。
5. **入库知识库**：`docker exec kada-ai python -m app.scripts.ingest_docs`
   每次执行都会先删掉同名 collection 再全量重建，所以只在文档变更后跑（会消耗 embedding 额度）。
   没入库不会让对话挂掉（检索失败降级为"没有参考资料"），但回答里就没有平台资料。

容器约定：

- `network_mode: host`：容器里的 `127.0.0.1` 就是宿主机，所以连接串沿用 `backend/.env` 的写法；
  systemd 里的 Go 网关按 `AI_BASE_URL` 默认值 `http://127.0.0.1:8000` 就能找到它；
  端口不进 Docker 的 iptables 链，ufw 依然管得住（published 端口在 Linux 上是绕过 ufw 的）。
- 启动命令显式绑定 `127.0.0.1:8000`，只允许本机调用。
- 镜像在服务器上构建，Python 3.11 和全部依赖都在镜像里，服务器不需要装 Python；
  `PIP_INDEX_URL` 默认清华源，海外机器用 `PIP_INDEX_URL=https://pypi.org/simple docker compose build`。
- 容器名固定为 `kada-ai`：日志 `docker logs kada-ai`，进容器 `docker exec -it kada-ai bash`。

> 交接/运维用的完整清单（合并前准备、验收命令、回滚、排障速查）见
> [`docs/ai-deployment.md`](../../docs/ai-deployment.md)。

### 排障对照

| 现象 | 多半原因 |
| --- | --- |
| `/api/ai/*` 返回 502 | 容器没起来或没监听 8000：`docker logs kada-ai`。`kada_ai` 库缺失会让启动阶段就崩（`init_db` 在 lifespan 里，数据库是硬依赖，Redis 才是可选的） |
| 接口返回 401 `only the Go gateway may call this service` | 两边 `AI_INTERNAL_SECRET` 不一致（或网关侧为空、服务侧配了）：改成同一个值后分别重启 `kada-api` 与 `kada-ai` |
| 回答里总是"知识库中没有相关资料"，日志有 `[RAG] 检索失败` | 没跑入库脚本、`aliyun` 密钥错、pgvector 没启用 |
| 工具回"无法执行：本次请求没有携带登录凭据" | 请求没带 `Authorization`（多半是绕过网关直连 Python 调试）；从平台页面正常使用时不该出现 |
| 工具报 HTTP 401 / 403 | 用户的登录状态已失效（token 过期或账号被停用）：重新登录即可，**不需要重启服务**——凭据按请求转发，不做缓存 |
| 日志有 `[AUTH] 未配置 AI_INTERNAL_SECRET` | 来源校验被关闭了：本机任何进程都能伪造 `X-Kada-User-ID`。仅限本地开发，生产必须配上 |
| 每条消息首字都很慢 | 未入库时仍会先调一次 DashScope embedding 做检索 |

## 环境变量

只有密钥和连接地址从环境变量读取，其余参数（模型名、缓存 TTL 等）的默认值写在 `app/config.py`。

| 变量 | 说明 | 默认 |
| --- | --- | --- |
| `DEEPSEEK_API_KEY` | DeepSeek API 密钥（必填） | 无 |
| `aliyun` | 阿里云 DashScope API Key，用于 RAG 的 embedding | 无 |
| `KADA_API_BASE` | Go 后端地址，业务工具回调用 | `http://localhost:8080` |
| `AI_INTERNAL_SECRET` | 与 Go 网关共享的服务密钥，网关用它证明"这次请求来自网关" | 无（空 = 不校验，仅本机开发） |

> 注意 embedding 的环境变量名约定为 `aliyun`（与百炼账号对应），不是 `DASHSCOPE_API_KEY`。
> 密钥不要写进代码或提交到仓库。
> 本服务**不持有任何长效 API Token**：工具的身份来自每个聊天请求自带的用户 JWT。
> `AI_INTERNAL_SECRET` 是**服务间**密钥，不是用户凭据：它只证明"调用方是网关"。泄露它并不足以操作用户的短链
> （那仍然需要有效的用户 JWT），但足以伪造 `X-Kada-User-ID` 去读或删除别人的对话历史 —— 所以它依然要按密钥保管。

## 接口契约（仅内网，Go 网关透传）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/chat` | SSE 流式对话，请求 `{conversation_id?, message}` |
| GET | `/conversations/current` | 当前用户最新会话及其消息，没有则返回空 |
| POST | `/conversations/restart` | 删除当前用户所有旧会话，新建一个空会话 |
| GET | `/healthz` | 健康检查（**不校验** `X-Internal-Secret`：compose 与部署脚本要用它探活，它只返回状态和模型名） |

- 上表的路径是**本服务的路径**；对外是 `/api/ai` 加同样的后缀（`/api/ai/chat`、`/api/ai/conversations/current`），
  网关把自己的挂载前缀剥掉后转发进来。两边都不带版本段，与项目里其他接口一致
- 所有业务接口（`/chat`、`/conversations/*`）都要求 `X-Internal-Secret` 与 `AI_INTERNAL_SECRET` 一致
- `X-Kada-User-ID`（由 Go 网关注入）决定会话归属，Python 不信任客户端直传
- `Authorization`（网关原样透传）是业务工具回调 Go 时使用的用户凭据

## 存储

- **PostgreSQL**（SQLModel ORM）：`ai_conversations` / `ai_messages` 表，会话记录是真相源；
  删除会话时消息靠外键 `ON DELETE CASCADE` 连带删除
- **pgvector**（langchain-postgres）：`langchain_pg_*` 表存知识库切片与向量，RAG 检索 top-k 片段
- **Redis**：缓存每个会话最近的消息（默认 50 条、TTL 1 小时），读多写少的 cache-aside；
  Redis 不可用时自动降级为直接查库

## 工具能力

- 通用工具（`app/service/tools.py`）：查时间、加法这类不需要用户身份的进程内 `@tool`
- 业务工具（`app/service/kada_tools.py`）：查短链总览、建短链。**以当前登录用户的身份执行**——
  凭据由 chat 路由放进 `contextvars`（按 asyncio Task 隔离，并发请求不会串号），
  再以 `Authorization: Bearer <用户 JWT>` 转交给 Go 的 `/api/*`
- MCP 工具（`app/mcp_server/kada_server.py`）：stdio 子进程，**目前没有注册任何工具**，是留给
  外部通用能力（搜索、文件等）的扩展点。业务工具不放这里的原因写在 `kada_tools.py` 的模块说明里：
  stdio 是跨请求共享的长连接，工具参数又由模型生成，拿不到请求级凭据
- 工具失败不拖垮对话：工具内部把失败转成文字返回（如"查询失败（HTTP 401）"），
  检索失败降级为"无参考资料"，只有模型侧异常才以 `error` 事件结束本轮
