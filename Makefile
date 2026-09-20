.PHONY: help dev dev-fe build build-fe test lint docker-up docker-down docker-logs \
        docker-build db-migrate db-reset install-tools clean \
        lint-ci lint-fe-ci test-race build-pkg release-dry-run

help:  ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ========== Development ==========

dev:  ## Run the Go API locally
	cd backend && go run ./cmd/server/main.go

dev-fe:  ## Run the frontend dev server
	cd frontend && npm run dev

dev-all:  ## Run the API and the frontend together
	@echo "Starting the backend (port 8080)..."
	cd backend && go run ./cmd/server/main.go &
	@echo "Starting the frontend (port 3000)..."
	cd frontend && npm run dev &
	@wait

# ========== Build ==========

build:  ## Build the Go API (backend/bin/server)
	cd backend && CGO_ENABLED=0 go build -o bin/server ./cmd/server/main.go && echo "Backend built: backend/bin/server"

build-fe:  ## Build the frontend
	cd frontend && npm run build && echo "Frontend built"

test:  ## Run the Go tests
	cd backend && go test ./... -v

test-fe:  ## Run the frontend tests
	cd frontend && npm test 2>/dev/null || echo "No frontend tests configured"

lint:  ## Run go vet
	cd backend && go vet ./...

lint-fe:  ## Run ESLint
	cd frontend && npm run lint 2>/dev/null || echo "No frontend lint configured"

# ========== Docker ==========

docker-up:  ## Start the full stack with Docker Compose
	@echo "Starting Docker Compose..."
	docker compose up -d
	@echo "Services: Nginx:80, API:8080, Frontend:3000, Postgres:5432"

docker-down:  ## Stop all Docker Compose services
	docker compose down

docker-logs:  ## Follow the Docker Compose logs
	docker compose logs -f

docker-build:  ## Build all Docker images
	docker compose build

docker-rebuild:  ## Rebuild and restart the Docker Compose stack
	docker compose up -d --build

# ========== Database ==========
# The schema is managed by GORM AutoMigrate (models live in internal/domain/entity/models.go).
# After changing a model, run `make db-migrate` to add the missing tables, columns and indexes.
# AutoMigrate is additive: it never drops anything, and it is safe to run repeatedly.

db-migrate:  ## Apply the database schema (GORM AutoMigrate)
	cd backend && go run ./cmd/migrate/

db-reset:  ## Destroy the database and rebuild the schema (destructive!)
	@echo "This will delete ALL data!"
	@read -p "Are you sure? [y/N] " -r reply && [ "$$reply" = "y" ] && \
	docker compose down -v && docker compose up -d postgres redis && \
	sleep 3 && cd backend && go run ./cmd/migrate/

# ========== Tooling ==========

install-tools:  ## Install the development tools
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

# ========== Deployment ==========
# The target server is never hardcoded; pass it explicitly:
#   make deploy DEPLOY_HOST=root@1.2.3.4
DEPLOY_HOST ?=

check-host:
	@test -n "$(DEPLOY_HOST)" || (echo "Set DEPLOY_HOST, for example: make deploy DEPLOY_HOST=root@1.2.3.4"; exit 1)

deploy: build check-host  ## Build and deploy the API (hot update)
	scp backend/bin/server $(DEPLOY_HOST):/opt/kada/backend/bin/server.new
	ssh $(DEPLOY_HOST) "mv /opt/kada/backend/bin/server.new /opt/kada/backend/bin/server && systemctl restart kada-api && echo 'Deployed'"

deploy-fe: build-fe-pkg  ## Build and deploy the frontend (hot update)
	ssh $(DEPLOY_HOST) "cd /opt/kada/frontend && rm -rf .next/static node_modules server.js package.json static 2>/dev/null; tar xzf /tmp/kada-fe-standalone.tar.gz && mkdir -p .next && tar xzf /tmp/kada-fe-static.tar.gz && mv static .next/static && rm -f /tmp/kada-fe-*.tar.gz && systemctl restart kada-frontend"
	@echo "Frontend deployed"

build-fe-pkg: build-fe check-host  ## Package and upload the frontend build
	cd frontend/.next/standalone && tar czf /tmp/kada-fe-standalone.tar.gz .
	cd frontend && tar czf /tmp/kada-fe-static.tar.gz -C .next static/
	scp /tmp/kada-fe-standalone.tar.gz /tmp/kada-fe-static.tar.gz $(DEPLOY_HOST):/tmp/
	@echo "Packages uploaded"

deploy-nginx: check-host  ## Update the nginx configuration
	scp nginx/nginx-prod.conf $(DEPLOY_HOST):/opt/kada/nginx/
	ssh $(DEPLOY_HOST) "docker restart kada-nginx"

# One-time preparation of the AI database on the server: creates kada_ai and enables pgvector in it.
# Safe to re-run. Needs the server's PostgreSQL to have pgvector available first.
setup-ai-db: check-host  ## Create the AI database + pgvector on the server (one-time)
	scp deploy/setup-ai-db.sh $(DEPLOY_HOST):/tmp/kada-setup-ai-db.sh
	ssh $(DEPLOY_HOST) "bash /tmp/kada-setup-ai-db.sh"

# The AI service is a Python environment rather than a binary: the source is shipped and the image is
# built on the host by deploy/deploy-ai.sh - the same script CI runs, so the two paths cannot drift.
# It needs /opt/kada/ai/ai.env and the kada_ai database to exist; see backend/ai/README.md.
deploy-ai: check-host  ## Deploy the Python AI service (image built on the server)
	cd backend/ai && tar czf /tmp/kada-ai-src.tar.gz --exclude='*__pycache__*' --exclude='*.pyc' .
	scp /tmp/kada-ai-src.tar.gz deploy/docker-compose.ai.yml deploy/deploy-ai.sh $(DEPLOY_HOST):/tmp/
	ssh $(DEPLOY_HOST) "mkdir -p /opt/kada/ai && cp /tmp/docker-compose.ai.yml /opt/kada/ai/docker-compose.yml && cp /tmp/deploy-ai.sh /opt/kada/ai/deploy-ai.sh && bash /opt/kada/ai/deploy-ai.sh"
	@echo "AI service deployed"

deploy-all: deploy deploy-fe deploy-ai  ## Deploy the API, the frontend and the AI service

# ========== CI helpers ==========

lint-ci:  ## Run golangci-lint (what CI runs)
	cd backend && golangci-lint run --timeout=5m ./...

lint-fe-ci:  ## Run ESLint (what CI runs)
	cd frontend && npm run lint

test-race:  ## Run the Go tests with the race detector (what CI runs)
	cd backend && go test ./... -v -count=1 -race -coverprofile=coverage.out

build-pkg: build build-fe  ## Build the release artifacts (simulates the CI deploy job)
	cd frontend && mkdir -p ../deploy-pkg && \
		tar czf ../deploy-pkg/kada-fe-standalone.tar.gz -C .next/standalone . && \
		tar czf ../deploy-pkg/kada-fe-static.tar.gz -C .next static/
	@echo "Packages ready in deploy-pkg/"

release-dry-run:  ## Build the Docker images locally
	docker build -t kada-backend:test ./backend
	docker build -t kada-frontend:test ./frontend
	@echo "Docker images built locally"

# ========== Cleanup ==========

clean:  ## Remove build output
	rm -rf backend/bin/ frontend/.next/
	docker compose down -v 2>/dev/null || true
