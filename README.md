# Kada

A short link management and analytics platform. Supports link folders and tags, custom domains, access passwords, expiry times, UTM templates, a click analytics dashboard, and an AI assistant that answers questions about your own links.

The backend is written in Go, the frontend in Next.js, click events are processed asynchronously through Kafka, and the AI assistant is a separate Python service behind the Go gateway.

## Features

- Phone number + SMS code sign-in (JWT), with a graphical challenge and per-phone / per-IP send limits
- Create, edit, delete, and bulk-manage short links
- Custom short codes and custom domains
- Access passwords and expiry times
- Folders, tags, and workspace management
- UTM parameters and templates
- Link preview, QR codes, and CSV export
- Click analytics dashboard (overview, platform breakdown, daily trend, visitor detail)
- Open API tokens
- AI assistant (RAG over the project knowledge base, tool calls that act as the signed-in user)
- Light and dark themes

## Tech stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.26, Gin, GORM |
| Storage | PostgreSQL 16 (pgvector for the knowledge base), Redis 7 |
| Messaging | Kafka 3.8 |
| Frontend | Next.js 16, React 19, TypeScript, SWR, Tailwind |
| AI service | Python 3.11, FastAPI, LangChain, DeepSeek + Aliyun Bailian |
| Deployment | Docker Compose, Nginx, systemd, GitHub Actions |

## Architecture

Layered as Handler -> Service -> Infra, using a modular monolith plus a standalone Kafka worker.

Click events are persisted asynchronously through Kafka and increment counters; when Kafka is unavailable this degrades to writing directly to the database.

The database schema is defined by the GORM models in `backend/internal/domain/entity` and applied by
`backend/cmd/migrate` (AutoMigrate). There are no SQL migration files: the structs are the single source
of truth, and AutoMigrate only adds missing tables/columns/indexes, so it is safe to re-run.

The AI assistant runs as its own Python service (`backend/ai`) that is never exposed publicly: the Go
gateway authenticates the caller and proxies `/api/ai/*` to it. See [backend/ai/README.md](backend/ai/README.md)
and [docs/ai-deployment.md](docs/ai-deployment.md).

## Getting started

Requirements: Go 1.26+, Node 22+, Docker.

```bash
# 1. Prepare environment variables and set a strong random JWT_SECRET
cp .env.example .env

# 2. Start all services (the API applies the schema on startup)
docker compose up -d

# 3. Apply the schema only (also runs on every deploy, before the API restarts)
cd backend && go run ./cmd/migrate/
```

Local development:

```bash
cd backend && go run ./cmd/server/main.go   # backend on :8080
cd frontend && npm run dev                  # frontend on :3000
```

## Environment variables

| Variable | Description |
|----------|-------------|
| JWT_SECRET | Required, a strong random secret |
| POSTGRES_PASSWORD | Required, must be changed in production |
| DB_AUTO_MIGRATE | Let the API server apply the schema on startup (default true; set false when using cmd/migrate) |
| SMS_ACCESS_KEY_ID / SMS_ACCESS_KEY_SECRET | Alibaba Cloud SMS. Required in production: phone + SMS code is the only sign-in method |
| SMS_SIGN_NAME / SMS_TEMPLATE_CODE | The system-granted SMS signature and template from the Aliyun PNVS console |
| AI_BASE_URL / AI_INTERNAL_SECRET | Where the Go gateway finds the AI service, and the shared secret that makes it accept only gateway requests |
| DEEPSEEK_API_KEY / aliyun | AI service only (chat model key, and the Bailian embedding key under the literal name `aliyun`) |

In production these come from GitHub repository secrets and are written into `/opt/kada/ai/ai.env` and
`/opt/kada/backend/.env` by the deploy job; see [docs/ai-deployment.md](docs/ai-deployment.md).

## Common commands

```bash
cd backend && go test ./... -v        # backend tests
cd backend && go vet ./...            # backend vet
cd frontend && npm run lint           # ESLint
cd frontend && npx tsc --noEmit       # type check
cd backend && go run ./cmd/migrate/   # apply the schema
docker compose up -d                  # start the dev stack
docker compose logs -f                # follow the logs
```

## Documentation

| Document | Contents |
|----------|----------|
| [docs/design.md](docs/design.md) | Architecture, data model, request flows, design decisions, known limitations |
| [docs/contributing.md](docs/contributing.md) | Local setup, workflow, commit conventions, pull requests |
| [docs/code-style.md](docs/code-style.md) | Go, TypeScript and database conventions used in this repository |
| [docs/ai-deployment.md](docs/ai-deployment.md) | Pushing the AI feature and what the server needs |
| [backend/ai/README.md](backend/ai/README.md) | AI service architecture, environment variables, API contracts, tools |

## Deployment

GitHub Actions runs lint, tests, builds, and the deployment itself on every push to `main`: the Go
binaries and the frontend build are uploaded, the AI image is built on the host, and the runtime env
files are written from repository secrets by `deploy/upsert-env.sh` before any service is touched. A
missing required secret fails the deployment while the previous build is still serving.

## Layout

```text
backend/   Go backend (cmd + internal), plus the Python AI service in backend/ai
frontend/  Next.js frontend
docs/      Design, contributing and code-style documentation
nginx/     Reverse proxy configuration
deploy/    Deployment scripts
docker-compose.yml
```
