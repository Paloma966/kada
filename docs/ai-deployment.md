# Kada AI 功能：提交推送与服务器部署

本文覆盖两件事：**你把代码推到 GitHub**（第一部分），以及**合并后同学在服务器上要做什么**（第二
部分）。两者是连着的——合并到 `main` 就会自动部署，所以第一部分里"合并前必须完成的准备"不能跳过。

改动范围：AI 助手（Go 网关 + Python 服务 + 前端 AI 页面）。短链、统计等原有功能的接口未改动。

---

## 一、推送到 GitHub（你来做）

### 1. 合并前必须先完成的三件事

合并到 `main` = 立即部署。**没有下面三项，部署会在第 0 步就停下**（第 0 步同步密钥，失败时旧版本仍在
运行，站点不受影响，只是这次部署不会完成）：

1. **建 AI 数据库**（只需一次）：

   ```bash
   scp deploy/setup-ai-db.sh root@<服务器IP>:/tmp/
   ssh root@<服务器IP> "bash /tmp/setup-ai-db.sh"
   ```

   它创建 `kada_ai` 库并在该库内启用 `vector` 扩展，可重复执行。
   报 `pgvector is not available` → 原生 PostgreSQL 装 `postgresql-16-pgvector` 并重启，
   Docker 则换成 `pgvector/pgvector:pg16` 镜像。

2. **配 `/opt/kada/ai/ai.env` 里的连接串**（服务器上，只需一次）：

   ```bash
   mkdir -p /opt/kada/ai
   scp deploy/ai.env.example root@<服务器IP>:/opt/kada/ai/ai.env
   ssh root@<服务器IP> "chmod 600 /opt/kada/ai/ai.env && vi /opt/kada/ai/ai.env"
   ```

   只需要填三个**非密钥**的连接串：`POSTGRES_URL`、`REDIS_URL`、`KADA_API_BASE`。
   `DEEPSEEK_API_KEY`、`aliyun`、`AI_INTERNAL_SECRET` 不用手填——第 3 步配好后，部署时会自动写进去。

3. **在 GitHub 仓库加密钥**（Settings → Secrets and variables → Actions）：

   | Secret | 用途 | 写到哪 | 缺失时 |
   | --- | --- | --- | --- |
   | `DEEPSEEK_API_KEY` | 对话模型密钥 | `ai.env` 的 `DEEPSEEK_API_KEY` | **部署失败** |
   | `DASHSCOPE_API_KEY` | 阿里云百炼 Embedding 密钥 | `ai.env` 的 `aliyun` | **部署失败** |
   | `AI_INTERNAL_SECRET` | 网关共享密钥，`openssl rand -hex 32` 生成 | **两边都写**：`ai.env` 与 `backend/.env` | **部署失败** |
   | `SMS_ACCESS_KEY_ID` / `SMS_ACCESS_KEY_SECRET` / `SMS_SIGN_NAME` / `SMS_TEMPLATE_CODE` | 阿里云短信 | `backend/.env` | 部署继续，但**没人能登录**，每次部署都会告警 |

   部署作业的第 0 步用 `deploy/upsert-env.sh` 把这些值幂等写入上面两个文件，**在替换任何服务之前**。
   缺 `--require` 的键（AI 那三个）就在这一步失败并打印是哪一个，旧版本继续对外服务——比"服务起来了、
   健康检查过了、然后每个请求都失败"好排查得多。

   **SMS 那四个是 `--warn` 而不是 `--require`，这是个取舍**：签名的模板都要在阿里云 PNVS 控制台审核，
   通常要等几天，在那之前卡死部署会让无关的修复也发不出去。但"不阻塞"不等于"能忽略"：

   - runner 上有一个单独的检查步骤（部署前），缺哪个就在 GitHub Actions 上打**黄色告警**并在
     Job Summary 里列出补哪个 Secret；
   - 服务器侧的部署日志里也会打一遍同样的告警；
   - 但对产品来说它们仍然不是可选项——**短信验证码是唯一的登录方式**，缺了它们站点就是"服务正常但
     没人能登录"，而且 release 模式下验证码不会打到日志里。**阿里云签名/模板一批下来，把四个 Secret
     填上重跑一次部署即可，不需要登服务器。**

   另外两项检查（不属于阻塞项，但会导致体验降级）：

   - 服务器 nginx 要有 `/api/ai/` 的 location（关闭缓冲，SSE 才能逐字输出）。没有就按第二部分里的
     nginx 步骤更新。
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
git add .github/workflows/ci.yml deploy docker-compose.yml .env.example backend/.env.example
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

0. **同步密钥**：从 GitHub Secrets 把 `DEEPSEEK_API_KEY` / `DASHSCOPE_API_KEY` / `AI_INTERNAL_SECRET`
   与四个 `SMS_*` 写入服务器上的 `ai.env` 与 `backend/.env`。AI 那三个缺任一就立刻失败退出；
   SMS 缺失只告警（部署前有一个 runner 侧检查会打告警并在 Job Summary 里列出缺哪个）。
   这一步放在最前面，所以密钥缺失时旧版本仍在服务，站点不受影响。
1. 备份 `kada-api` 二进制 → 应用数据库迁移 → 重启 `kada-api` → 健康检查（失败则回滚二进制）
2. 替换并重启前端
3. **同步 `backend/ai` 源码到服务器 → 在服务器上构建 AI 镜像 → 重启 `kada-ai` 容器 → 探活 →
   失败自动回滚到上一个镜像**

整个 deploy 作业约 5–15 分钟，大部分时间在第 3 步首次构建镜像（拉 `python:3.11-slim` + 装依赖）。

> **注意：第 1 步之后到第 3 步完成之间，AI 页面会返回 404。** 因为这次路由改名是"三端一起改"，
> 新 Go 二进制只认新路径，而旧 AI 容器只认旧路径。窗口只有几分钟，第 3 步成功后自愈，用户重新
> 发一条消息即可。短链主站不受影响。

**必须新增 GitHub Secret**：除了已有的 `SERVER_HOST`、`SERVER_USER`、`SSH_PRIVATE_KEY`、
`DATABASE_URL`（`SITE_URL` 变量可选），还要加上第 1 节表格里的七个（`DEEPSEEK_API_KEY`、
`DASHSCOPE_API_KEY`、`AI_INTERNAL_SECRET`、四个 `SMS_*`）。其中**前三个必须先配**，否则部署会失败；
四个 `SMS_*` 缺失只告警。它们都由流水线写进服务器的 `ai.env` 与 `backend/.env`，所以以后改密钥只需要
改 Secret 再重跑部署，不用再登服务器。

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

**② 配 AI 的连接串（密钥交给流水线）**

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

另外三个（`DEEPSEEK_API_KEY`、`aliyun`、`AI_INTERNAL_SECRET`）**留空即可**：部署作业会用
`deploy/upsert-env.sh` 从 GitHub Secrets 写进去，`--set` 遇到空值不会覆盖已有内容，
`--require` 会在值仍然缺失时让部署失败并说明缺哪个。

**③ 网关侧的同名密钥：不用手配**

`AI_INTERNAL_SECRET` 也由同一个 Secret 写进 `/opt/kada/backend/.env`。以前它需要在两个文件里各填一遍、
值还必须完全一致，不一致时 AI 页面整体返回 401 `only the Go gateway may call this service`——现在两边
出自同一个 Secret，不可能再对不上。

`deploy-ai.sh` 在构建镜像前会再校验一次：`/opt/kada/ai/ai.env` 不存在、或 `DEEPSEEK_API_KEY` / `aliyun`
为空，就直接拒绝构建——服务"能起来、健康检查能过、然后每个请求都失败"是最难排查的一种坏法。

另外确认 nginx 配置里有 `/api/ai/` 段（关闭缓冲，否则 SSE 打字效果消失、60 秒断流）：

```bash
docker exec kada-nginx nginx -T 2>/dev/null | grep -c 'location /api/ai/'   # 应为 1
```

没有就在服务器上更新配置并重启 nginx 容器：

```bash
scp nginx/nginx-prod.conf root@<服务器>:/opt/kada/nginx/
ssh root@<服务器> "docker restart kada-nginx"
```

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
| 前端 | 无自动备份：在本地 `git revert` 后重跑流水线（推荐），或在服务器上重新解包上一版构建 |

> 这次的路由改名让三端互相绑定：只回滚其中一端会让 AI 页面 404。要回滚就整体回到上一个 commit。

### 排障速查

| 现象 | 原因 |
| --- | --- |
| `/api/ai/*` 返回 502 | AI 容器没起来或没监听 8000。`docker logs kada-ai`；`kada_ai` 库缺失会让启动阶段就崩 |
| 返回 401 `only the Go gateway may call this service` | `ai.env` 与 `backend/.env` 里的 `AI_INTERNAL_SECRET` 不一致。现在两边都由同一个 GitHub Secret 写入，重跑一次部署即可对齐 |
| 日志有 `[AUTH] 未配置 AI_INTERNAL_SECRET` | 密钥为空，来源校验被关闭，仅限本机开发，生产必须配上 |
| AI 页面能打开但每次提问都失败 | `ai.env` 里的 `DEEPSEEK_API_KEY` / `aliyun` 为空或写错。部署时 `deploy-ai.sh` 会先拒绝这种状态；已经跑起来的话，改 GitHub Secret 后重跑部署 |
| 回答总是"知识库中没有相关资料"，日志有 `[RAG] 检索失败` | 没跑入库脚本、`aliyun` 密钥错、或 pgvector 未启用 |
| 工具报 HTTP 401/403，聊天正常 | 用户的登录已过期，重新登录即可（**不需要重启服务**） |
| 工具回"无法执行：本次请求没有携带登录凭据" | 请求绕过了网关直连 Python（本地调试才会出现） |
| **没人能登录**（验证码收不到），日志有 `❌ SMS service disabled, NOBODY CAN SIGN IN: missing SMS …` | `backend/.env` 里少 `SMS_SIGN_NAME` 或 `SMS_TEMPLATE_CODE`。**这两个值不在仓库里**，去阿里云 PNVS 控制台取；本账号的签名是 `kada`。它们曾经被硬编码在源码里、也曾在 `backend/.env.example` 里，都在重构中被清掉了——完整经过见 `docs/design.md` §9.1 |
| 验证码接口返回 `failed to send SMS: … (provider code: …)` | 签名/模板没通过、AccessKey 被停用或欠费。日志里那一行带 `sign_name=` / `template_code=`，两者必须来自同一个账号且成对使用 |

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
