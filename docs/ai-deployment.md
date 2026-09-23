# Deploying the AI service

The AI assistant is the Go gateway plus a Python service in the `kada-ai` container (see
[`backend/ai/README.md`](../backend/ai/README.md)). This document is the handover for it: what has to exist
on the host once, what a deploy does, how to verify it, and how to recover.

Merging to `main` deploys it. The work left for the host is the one-time setup below and the verification
afterwards.

## One-time setup (on the host, before the first deploy)

### 1. The `kada_ai` database and the vector extension

`deploy/setup-ai-db.sh` creates the database and enables `vector` in it. It is idempotent, keeps existing
data, and works out for itself whether PostgreSQL is a container or a native process:

```bash
scp deploy/setup-ai-db.sh root@<host>:/tmp/
ssh root@<host> "bash /tmp/setup-ai-db.sh"
```

Which amounts to:

```sql
CREATE DATABASE kada_ai;                      -- errors if it already exists, which is fine
\connect kada_ai
CREATE EXTENSION IF NOT EXISTS vector;
```

- `pgvector is not available` means the extension is not installed: on native PostgreSQL install
  `postgresql-16-pgvector` and restart it, on Docker switch to the `pgvector/pgvector:pg16` image.
- The extension belongs in `kada_ai`, not in the Go business database `kada`. The init script
  (`deploy/postgres/initdb/01-create-ai-db.sql`) only runs when the data volume is created for the first
  time, so a host that already holds data has to use the script above.

### 2. `/opt/kada/ai/ai.env`

```bash
mkdir -p /opt/kada/ai
scp deploy/ai.env.example root@<host>:/opt/kada/ai/ai.env
ssh root@<host> "chmod 600 /opt/kada/ai/ai.env && vi /opt/kada/ai/ai.env"
```

(or, on the host with the repository checked out: `cp deploy/ai.env.example /opt/kada/ai/ai.env`)

Three connection strings, none of them secret:

| Variable | Value |
| --- | --- |
| `POSTGRES_URL` | `postgresql+asyncpg://kada:<database password>@127.0.0.1:5432/kada_ai` |
| `REDIS_URL` | `redis://127.0.0.1:6379/0` |
| `KADA_API_BASE` | `http://127.0.0.1:8080` |

Take the password from `DATABASE_URL` in `/opt/kada/backend/.env`, with the database name changed from
`kada` to `kada_ai`.

Leave `DEEPSEEK_API_KEY`, `aliyun` and `AI_INTERNAL_SECRET` empty: the deploy job writes them from GitHub
Secrets, and `--set` with an empty value never overwrites what is already in the file.

`POSTGRES_URL` is the one value that must not be left to a default: `backend/ai/app/config.py` falls back
to a development DSN with a guessed password, so an empty one would produce a container that starts and
then crash-loops inside `init_db()` - after the image had been built. `deploy/deploy-ai.sh` closes that
hole by refusing to build unless the file exists and `POSTGRES_URL`, `DEEPSEEK_API_KEY` and `aliyun` are
filled in, and warns rather than refuses when `AI_INTERNAL_SECRET` is empty. `REDIS_URL` and
`KADA_API_BASE` are deliberately not required: their defaults are what this host already runs.

### 3. GitHub repository secrets

Settings -> Secrets and variables -> Actions:

| Secret | Purpose | Written to | If missing |
| --- | --- | --- | --- |
| `DEEPSEEK_API_KEY` | chat model key | `ai.env` | deploy fails |
| `DASHSCOPE_API_KEY` | Aliyun Bailian embedding key | `ai.env`, as `aliyun` | deploy fails |
| `AI_INTERNAL_SECRET` | gateway shared secret, `openssl rand -hex 32` | `ai.env` **and** `backend/.env` | deploy fails |
| `SMS_ACCESS_KEY_ID` / `SMS_ACCESS_KEY_SECRET` / `SMS_SIGN_NAME` / `SMS_TEMPLATE_CODE` | Aliyun SMS | `backend/.env` | deploy fails - these four are the only way anyone signs in |

The deploy job syncs them from GitHub Secrets with `deploy/upsert-env.sh` before it replaces any service,
and stops with the name of the missing key if one is still empty afterwards. That is deliberately louder
than the alternative: a service that starts, passes its health check and then fails every request is the
worst way to find out that a key never arrived.

`--require` checks the file on the host, not whether the secret exists, which has two consequences:

- a value configured by hand in the host's `.env` satisfies it, so a host that was set up before the
  secrets existed keeps deploying;
- a value that exists only on the host is lost when the machine is rebuilt - which is the reason to add the
  secret anyway. A non-fatal notice in the job summary says so while any of the four SMS credentials is
  unmanaged, and disappears once they are set.

### 4. nginx, and the old API token

The `/api/ai/` location in `nginx/nginx-prod.conf` turns proxy buffering off. Without it the SSE answer
arrives in one piece instead of streaming, or the connection dies at 60 seconds:

```bash
docker exec kada-nginx nginx -T 2>/dev/null | grep -c 'location /api/ai/'   # expect 1
```

If it is missing, copy the configuration over and restart the container:

```bash
scp nginx/nginx-prod.conf root@<host>:/opt/kada/nginx/
ssh root@<host> "docker restart kada-nginx"
```

Also revoke the old long-lived API token in the platform's settings (Settings -> API Token). Nothing reads
it any more - the tools act as the signed-in user - but it is a real token, and it stays in the git history.

## What a deploy does

A pull request runs the checks and never the deploy job (`if: github.ref == 'refs/heads/main' &&
github.event_name == 'push'`). Merging to `main` runs, in order:

1. **Sync secrets** from GitHub Secrets into `ai.env` and `backend/.env`. A required value that is still
   empty fails the deploy here, while the previous build keeps serving.
2. **Go API**: back the running binary up, apply the schema, restart `kada-api`, health-check it and roll
   the binary back on failure - then replace and restart the frontend.
3. **AI service**: copy `backend/ai` to the host, build the image there, restart the `kada-ai` container,
   probe it, and roll back to `kada-ai:previous` if it never becomes healthy.

5-15 minutes; most of it step 3 on a cold cache (`python:3.11-slim` plus the dependencies).

> Between step 2 and the end of step 3 the AI page returns 404. The route rename dropped `/v1` on all three
> tiers at once, so the new Go binary and the old AI container disagree about paths for those few minutes.
> It heals itself when step 3 finishes and the user sends one more message. Short links are unaffected.

## After the deploy: verify

```bash
# 1. the container is up, bound to loopback only
docker ps --filter name=kada-ai
ss -ltnp | grep 8000                      # 127.0.0.1:8000

# 2. health check, no secret needed
curl -s http://127.0.0.1:8000/healthz     # {"status":"ok","service":"kada-ai","model":"deepseek-flash"}

# 3. the gateway guard: no secret, no entry
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8000/conversations/current   # 401

# 4. the link site is untouched
curl -s http://127.0.0.1:8080/api/health
```

5. In a browser: sign in, open the AI page, ask how many short links there are. The answer should stream a
   character at a time and cite real data, and "start over" should clear the conversation and open a new one.

## Knowledge base (once, and after the documentation changes)

```bash
docker exec kada-ai python -m app.scripts.ingest_docs
```

Chat works without it: retrieval then degrades to the answer that the knowledge base holds nothing relevant
(`知识库中没有相关资料`) and logs `[RAG] 检索失败`. Re-running rebuilds the collection.

## Maintenance

```bash
docker logs -f kada-ai                                            # logs
docker restart kada-ai                                            # restart only
cd /opt/kada/ai && docker compose build && docker compose up -d   # rebuild from the source on the host
```

## Rollback

| Component | How |
| --- | --- |
| AI container | `docker tag kada-ai:previous kada-ai:deploy && docker compose -f /opt/kada/ai/docker-compose.yml up -d --force-recreate` |
| Go API | Every deploy backs the running binary up to `/opt/kada/backend/backups/server.<timestamp>`: `cp /opt/kada/backend/backups/server.<newest> /opt/kada/backend/bin/server && systemctl restart kada-api` |
| Frontend | Nothing is kept on the host: `git revert` locally and re-run the pipeline, or unpack the previous build on the server |

The route rename ties the three tiers to each other, so rolling one of them back on its own leaves the AI
page at 404. Roll the whole thing back to the previous commit.

## Troubleshooting

| Symptom | Cause |
| --- | --- |
| The deploy stops at `POSTGRES_URL is empty in /opt/kada/ai/ai.env` | The connection string was never filled in. That check sits before the image build on purpose: otherwise the container starts on the guessed development DSN and dies in `init_db()`, ten minutes into the build. `deploy/ai.env.example` has the shape, and the password is `DATABASE_URL` from `backend/.env` with the database name changed to `kada_ai`. |
| `/api/ai/*` returns 502 | The container is not up, or not listening on 8000 (`docker logs kada-ai`). A missing `kada_ai` database crashes it during startup. |
| 401 `only the Go gateway may call this service` | `AI_INTERNAL_SECRET` differs between `ai.env` and `backend/.env`, or the gateway side is empty. Both now come from one GitHub Secret, so re-running the deploy aligns them. |
| The log has `[AUTH] 未配置 AI_INTERNAL_SECRET` | The origin check is off, and any process on the host can forge `X-Kada-User-ID`. Local development only; production must set it. |
| The AI page opens but every question fails | `DEEPSEEK_API_KEY` or `aliyun` is empty or wrong. `deploy-ai.sh` refuses to build in that state; on an already running service, correct the secret and re-run the deploy. |
| Answers always say the knowledge base holds nothing relevant, and the log has `[RAG] 检索失败` | The ingest script was never run, the `aliyun` key is wrong, or pgvector is not enabled. |
| A tool returns HTTP 401/403 while chat itself works | The user's sign-in expired; they sign in again. No service restart. |
| A tool answers that the request carried no credentials | The request reached Python directly instead of through the gateway. Local debugging only. |
| **Nobody can sign in** (no code arrives), and the log has `SMS service disabled, NOBODY CAN SIGN IN: missing SMS ...` | `SMS_SIGN_NAME` or `SMS_TEMPLATE_CODE` is missing from `backend/.env`. Neither value is in the repository: they are the pair the Aliyun PNVS console grants (Phone Number Verification Service -> SMS verification -> Overview). On this account the signature is `恒创联众` and the template code `100001`. Both were once hard-coded in the source and were removed by a refactor; the history is in `docs/design.md` §9.1. |
| The captcha endpoint returns `failed to send SMS: ... (provider code: ...)` | The signature or the template was not approved, or the AccessKey is disabled or out of credit. The log line carries `sign_name=` and `template_code=`, which have to come from the same account and be used as a pair. |

Architecture, environment variables, API contracts and tool capabilities are in
[`backend/ai/README.md`](../backend/ai/README.md).

## What this feature changed

- **Tools act as the signed-in user.** Querying statistics and creating a short link no longer use a
  long-lived service token; they pass on the JWT that the gateway already forwards. Authorization lives in
  the Go API alone, so a disabled or expired user takes effect immediately, with no restart.
- **The AI service accepts only the gateway.** The gateway injects `X-Internal-Secret`
  (`AI_INTERNAL_SECRET`), and Python checks it before it will treat the `X-Kada-User-ID` header as an
  identity claim.
- **No version segment in the routes.** Public: `/api/ai/chat`, `/api/ai/conversations/current`,
  `/api/ai/conversations/restart`. Inside the service: `/chat`, `/conversations/current`,
  `/conversations/restart`. The gateway strips the mount prefix.
- **One extra container.** `kada-ai` (`deploy/docker-compose.ai.yml`) uses the host network and binds only
  `127.0.0.1:8000`, so it takes no public port and `ufw` still governs it.
