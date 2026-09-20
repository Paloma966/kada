# Kada AI 功能：提交推送与服务器部署

本文覆盖两件事：**你把代码推到 GitHub**（第一部分），以及**合并后同学在服务器上要做什么**（第二
部分）。两者是连着的——合并到 `main` 就会自动部署，所以第一部分里"合并前必须完成的准备"不能跳过。

改动范围：AI 助手（Go 网关 + Python 服务 + 前端 AI 页面）。短链、统计等原有功能的接口未改动。

---

## 一、推送到 GitHub（你来做）

### 1. 合并前必须先完成的三件事

合并到 `main` = 立即部署，而这次部署的第 7 步会启动 AI 容器。**没有下面三项，第 7 步会直接失败**
（API 和前端已经部署成功，只有 AI 页面不可用）：

1. **建 AI 数据库**（只需一次）：
   ```bash
   make setup-ai-db DEPLOY_HOST=root@<服务器IP>
   ```
   它创建 `kada_ai` 库并在该库内启用 `vector` 扩展，可重复执行。
   报 `pgvector is not available` → 原生 PostgreSQL 装 `postgresql-16-pgvector` 并重启，
   Docker 则换成 `pgvector/pgvector:pg16` 镜像。

2. **配 `/opt/kada/ai/ai.env`**（服务器上，只需一次）：
   ```bash
   mkdir -p /opt/kada/ai
   scp deploy/ai.env.example root@<服务器IP>:/opt/kada/ai/ai.env
   ssh root@<服务器IP> "chmod 600 /opt/kada/ai/ai.env && vi /opt/kada/ai/ai.env"
   ```
   填 6 个值：`POSTGRES_URL`、`REDIS_URL`、`KADA_API_BASE`、`DEEPSEEK_API_KEY`、`aliyun`、
   `AI_INTERNAL_SECRET`。其中 `AI_INTERNAL_SECRET` 用 `openssl rand -hex 32` 生成。

3. **网关侧配同一个密钥**（服务器上）：
   ```bash
   ssh root@<服务器IP> "echo 'AI_INTERNAL_SECRET=<上一步那个值>' >> /opt/kada/backend/.env"
   ```
   改动 `backend/.env` 后需要 `systemctl restart kada-api` 才生效——流水线重启 API 时会自动读到，
   所以放在推送前配置即可。**两边不一致时，AI 页面会整体返回 401。**

另外两项检查（不属于阻塞项，但会导致体验降级）：

- 服务器 nginx 要有 `/api/ai/` 的 location（关闭缓冲，SSE 才能逐字输出）。没有就 `make deploy-nginx`。
- 平台「设置 → API Token」里**吊销旧的长效令牌**。代码已经不用它了，但它仍在 git 历史中，是一把真令牌。

### 2. 提交

```bash
cd <本仓库>
git switch -c feat/ai-per-user-identity
git reset                      # 清空暂存区（含 session.py→conversation.py 的重命名），便于分组提交
git status --short             # 确认列表里没有 .env / ai.env / 临时产物
```

按主题分三次提交（提交信息用英文、conventional 前缀、冒号后小写、不加署名尾注）：

```bash
# ① 行为改动：工具以登录用户身份执行 + 服务只认网关 + 路由去版本段
git add backend/ai/app backend/ai/requirements.txt \
        backend/internal/handler/ai backend/config/config.go backend/cmd/server/main.go \
        frontend/src
git commit -m "feat(ai): act as the signed-in user and accept only gateway requests"

# ② CI 与部署：镜像构建、冒烟测试、自动部署
git add .github/workflows/ci.yml deploy Makefile docker-compose.yml .env.example backend/.env.example
git commit -m "ci: build, smoke-test and deploy the AI service"

# ③ 文档
git add backend/ai/README.md docs/design.md docs/ai-deployment.md "backend/ai/docs/Kada项目知识库.md"
git commit -m "docs(ai): document the deployment, routes and credentials"
```

不想拆就一句 `git add -A && git commit -m "feat(ai): ..."`，功能上没区别。

### 3. 推分支、看 CI

```bash
git push -u origin feat/ai-per-user-identity
```

然后在 GitHub 开一个到 `main` 的 Pull Request。**PR 不会触发部署**（deploy 作业有
`if: github.ref == 'refs/heads/main'`），只会跑六个检查作业：

| 作业 | 检查内容 | 预期耗时 |
| --- | --- | --- |
| backend-lint | `golangci-lint`（本次动了 Go 网关） | 1–2 分钟 |
| backend-test | Go 单测（含网关的 5 个用例） | 1–2 分钟 |
| frontend-lint / frontend-build | ESLint、`tsc --noEmit`、`next build` | 2–3 分钟 |
| ai-build | 真的构建 AI 镜像 → 校验 DashScope SDK → 起 PostgreSQL 冒烟测 `/healthz`、网关守卫三类状态码 | 3–6 分钟 |

全绿再合并；有红的先修（这一步是零成本的预检，不碰生产）。

### 4. 合并到 main

合并（普通合并 / squash 都行）后流水线会多出 `deploy` 作业，部署步骤如下：

1. 备份 `kada-api` 二进制 → 应用数据库迁移 → 重启 `kada-api` → 健康检查（失败则回滚二进制）
2. 替换并重启前端
3. **同步 `backend/ai` 源码到服务器 → 在服务器上构建 AI 镜像 → 重启 `kada-ai` 容器 → 探活 →
   失败自动回滚到上一个镜像**

整个 deploy 作业约 5–15 分钟，大部分时间在第 3 步首次构建镜像（拉 `python:3.11-slim` + 装依赖）。

> **注意：第 1 步之后到第 3 步完成之间，AI 页面会返回 404。** 因为这次路由改名是"三端一起改"，
> 新 Go 二进制只认新路径，而旧 AI 容器只认旧路径。窗口只有几分钟，第 3 步成功后自愈，用户重新
> 发一条消息即可。短链主站不受影响。

**不需要新增任何 GitHub Secret**：流水线仍只用已有的 `SERVER_HOST`、`SERVER_USER`、
`SSH_PRIVATE_KEY`、`DATABASE_URL`（`SITE_URL` 变量可选）。AI 的密钥全部放在服务器上。

---

## 二、服务器部署（同学来做）

如果你拿到的是一台**已经在跑 Kada** 的服务器（Go API / 前端是 systemd，nginx 是 Docker 容器，
PostgreSQL 与 Redis 已在本机运行），按下面的顺序做即可。

### 准备阶段（合并前，三件）

**① 建 AI 数据库**

把仓库里的 `deploy/setup-ai-db.sh` 拷到服务器执行（可重复执行，不会覆盖已有数据）：

```bash
scp deploy/setup-ai-db.sh root@<服务器>:/tmp/
ssh root@<服务器> "bash /tmp/setup-ai-db.sh"
```

脚本会自己判断 PostgreSQL 是容器还是原生进程，并给出下一步提示。它做的其实就是：

```sql
CREATE DATABASE kada_ai;                      -- 已存在时会报错，属正常
\connect kada_ai
CREATE EXTENSION IF NOT EXISTS vector;
```

> 注意扩展要装在 **`kada_ai`** 库里，不是 Go 业务库 `kada`。
> `deploy/postgres/initdb/01-create-ai-db.sql` 只在数据卷**首次初始化**时生效，已有数据的服务器
> 必须用上面的脚本。

**② 配 AI 的密钥文件**

```bash
mkdir -p /opt/kada/ai
cp /path/to/deploy/ai.env.example /opt/kada/ai/ai.env
chmod 600 /opt/kada/ai/ai.env
vi /opt/kada/ai/ai.env
```

| 变量 | 填什么 |
| --- | --- |
| `POSTGRES_URL` | `postgresql+asyncpg://kada:<数据库密码>@127.0.0.1:5432/kada_ai` |
| `REDIS_URL` | `redis://127.0.0.1:6379/0` |
| `KADA_API_BASE` | `http://127.0.0.1:8080` |
| `DEEPSEEK_API_KEY` | 对话模型密钥 |
| `aliyun` | 阿里云百炼（DashScope）Embedding 密钥，变量名就是 `aliyun` |
| `AI_INTERNAL_SECRET` | `openssl rand -hex 32` 生成，**要和网关侧一致** |

**③ 让网关带上同一个密钥**

```bash
echo 'AI_INTERNAL_SECRET=<与 ai.env 完全相同的值>' >> /opt/kada/backend/.env
```

配置文件缺失或两边不一致时的表现：`deploy-ai.sh` 会拒绝部署并打印该怎么建文件；两边不一致则
AI 接口返回 401 `only the Go gateway may call this service`。

另外确认 nginx 配置里有 `/api/ai/` 段（关闭缓冲，否则 SSE 打字效果消失、60 秒断流）：

```bash
docker exec kada-nginx nginx -T 2>/dev/null | grep -c 'location /api/ai/'   # 应为 1
```

没有就 `make deploy-nginx`（从本地仓库执行），或在服务器上更新 `/opt/kada/nginx/nginx-prod.conf`
后 `docker restart kada-nginx`。

### 合并后

不需要你在服务器上做任何操作，流水线会自动完成：构建镜像 → 重启容器 → 探活（失败回滚到上一个
镜像）。你只需要看 GitHub Actions 的 `deploy` 作业是否变绿。

### 验收（5 项）

```bash
# ① 容器在跑，端口只绑本机
docker ps --filter name=kada-ai
ss -ltnp | grep 8000                      # 应显示 127.0.0.1:8000

# ② 健康检查（不需要密钥）
curl -s http://127.0.0.1:8000/healthz     # {"status":"ok","service":"kada-ai","model":"deepseek-flash"}

# ③ 网关守卫生效：没有密钥必须被拒
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8000/conversations/current   # 401

# ④ 短链主站没受影响
curl -s http://127.0.0.1:8080/api/health
```

⑤ 浏览器登录平台 → 打开 AI 页面 → 发一句"我有多少条短链"，应逐字流式回答并能引用真实数据；
点"重新开始"能清空并新建会话。

### 部署后：知识库入库（一次）

```bash
docker exec kada-ai python -m app.scripts.ingest_docs
```

不跑也能正常对话，只是回答里没有平台资料（检索失败会降级为"知识库中没有相关资料"并写日志）。
文档更新后需要重跑，每次都会重建同名 collection。

### 日常维护命令

```bash
docker logs -f kada-ai                                   # 看日志
docker restart kada-ai                                   # 只重启
cd /opt/kada/ai && docker compose build && docker compose up -d   # 用现存源码重建
```

### 回滚

| 组件 | 做法 |
| --- | --- |
| AI 容器 | `docker tag kada-ai:previous kada-ai:deploy && docker compose -f /opt/kada/ai/docker-compose.yml up -d --force-recreate` |
| Go API | 流水线每次部署前会把旧二进制备份到 `/opt/kada/backend/backups/server.<时间戳>`：`cp /opt/kada/backend/backups/server.<最新> /opt/kada/backend/bin/server && systemctl restart kada-api` |
| 前端 | 无自动备份：在本地 `git revert` 后重跑流水线，或用 `make deploy-fe` 重新推上一版构建 |

> 这次的路由改名让三端互相绑定：只回滚其中一端会让 AI 页面 404。要回滚就整体回到上一个 commit。

### 排障速查

| 现象 | 原因 |
| --- | --- |
| `/api/ai/*` 返回 502 | AI 容器没起来或没监听 8000。`docker logs kada-ai`；`kada_ai` 库缺失会让启动阶段就崩 |
| 返回 401 `only the Go gateway may call this service` | 两边 `AI_INTERNAL_SECRET` 不一致。改成同一个值后分别重启 `kada-api` 与 `kada-ai` |
| 日志有 `[AUTH] 未配置 AI_INTERNAL_SECRET` | 密钥为空，来源校验被关闭，仅限本机开发，生产必须配上 |
| 回答总是"知识库中没有相关资料"，日志有 `[RAG] 检索失败` | 没跑入库脚本、`aliyun` 密钥错、或 pgvector 未启用 |
| 工具报 HTTP 401/403，聊天正常 | 用户的登录已过期，重新登录即可（**不需要重启服务**） |
| 工具回"无法执行：本次请求没有携带登录凭据" | 请求绕过了网关直连 Python（本地调试才会出现） |

更完整的说明见 `backend/ai/README.md`（架构、环境变量、接口契约、工具能力）。

---

## 三、这次改动做了什么（交接背景）

- **工具以当前登录用户的身份执行**：AI 的"查统计 / 建短链"不再使用一把长效服务令牌，而是把网关
  转发的用户 JWT 原样交给 Go 的 `/api/*`。权限判定只在 Go 一处，用户停用或过期立即生效，无需重启。
- **AI 服务只接受来自网关的请求**：网关注入 `X-Internal-Secret`（`AI_INTERNAL_SECRET`），Python
  校验后才处理，因此 `X-Kada-User-ID` 才有资格被当作身份断言。
- **路由去掉 `/v1`**：对外 `/api/ai/chat`、`/api/ai/conversations/current`、
  `/api/ai/conversations/restart`；服务内部 `/chat`、`/conversations/current`、
  `/conversations/restart`（网关剥掉挂载前缀后转发）。与项目其它接口一致，URL 里不出现版本段。
- **新增一个容器**：`kada-ai`（`deploy/docker-compose.ai.yml`），host 网络、只绑 `127.0.0.1:8000`，
  不占用公网端口，ufw 依然管得住。
