# Kada

A short link management and analytics platform. Supports multiple sign-in methods, link folders and tags, custom domains, access passwords, expiry times, UTM templates, and a click analytics dashboard.

The backend is written in Go, the frontend in Next.js, and click events are processed asynchronously through Kafka.

## Features

- Phone / email / WeChat sign-in (JWT)
- Create, edit, delete, and bulk-manage short links
- Custom short codes and custom domains
- Access passwords and expiry times
- Folders, tags, and workspace management
- UTM parameters and templates
- Link preview, QR codes, and CSV export
- Click analytics dashboard (overview, platform breakdown, daily trend, visitor detail)
- Open API tokens

## Tech stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.26, Gin, GORM |
| Storage | PostgreSQL 16, Redis 7 |
| Messaging | Kafka 3.8 |
| Frontend | Next.js 16, React 19, TypeScript, SWR, Tailwind |
| Deployment | Docker Compose, Nginx, systemd |

## Architecture

Layered as Handler -> Service -> Infra, using a modular monolith plus a standalone Kafka worker.

Click events are persisted asynchronously through Kafka and increment counters; when Kafka is unavailable this degrades to writing directly to the database.

The database schema is defined by the GORM models in `backend/internal/domain/entity` and applied by
`backend/cmd/migrate` (AutoMigrate). There are no SQL migration files: the structs are the single source
of truth, and AutoMigrate only adds missing tables/columns/indexes, so it is safe to re-run.

## Getting started

Requirements: Go 1.26+, Node 22+, Docker.

```bash
# 1. Prepare environment variables and set a strong random JWT_SECRET
cp .env.example .env

# 2. Start all services (the API applies the schema on startup)
make docker-up

# 3. Apply the schema only (also runs on every deploy, before the API restarts)
make db-migrate
```

Local development:

```bash
make dev        # backend on :8080
make dev-fe     # frontend on :3000
```

## Environment variables

| Variable | Description |
|----------|-------------|
| JWT_SECRET | Required, a strong random secret |
| POSTGRES_PASSWORD | Required, must be changed in production |
| DB_AUTO_MIGRATE | Let the API server apply the schema on startup (default true; set false when using cmd/migrate) |
| SMS_ACCESS_KEY_ID / SMS_ACCESS_KEY_SECRET | Alibaba Cloud SMS |
| SMS_SIGN_NAME / SMS_TEMPLATE_CODE | SMS signature and template |

## Common commands

`make test`, `make build`, `make lint`, `make db-migrate`, `make docker-up`, `make docker-logs`

## Documentation

| Document | Contents |
|----------|----------|
| [docs/design.md](docs/design.md) | Architecture, data model, request flows, design decisions, known limitations |
| [docs/contributing.md](docs/contributing.md) | Local setup, workflow, commit conventions, pull requests |
| [docs/code-style.md](docs/code-style.md) | Go, TypeScript and database conventions used in this repository |

## Deployment

GitHub Actions runs lint, tests, build, and deployment automatically. To deploy manually:

```bash
make deploy DEPLOY_HOST=root@your-server
make deploy-fe DEPLOY_HOST=root@your-server
```

## Layout

```text
backend/   Go backend (cmd + internal)
frontend/  Next.js frontend
docs/      Design, contributing and code-style documentation
nginx/     Reverse proxy configuration
deploy/    Deployment scripts
docker-compose.yml
Makefile
```
