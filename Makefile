.PHONY: help dev dev-fe build build-fe test lint docker-up docker-down docker-logs \
        docker-build db-migrate db-create db-reset sqlc-gen install-tools clean \
        lint-ci lint-fe-ci test-race build-pkg release-dry-run

help:  ## 显示帮助信息
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ========== 开发 ==========

dev:  ## 启动 Go 后端
	cd backend && go run ./cmd/server/main.go

dev-fe:  ## 启动前端开发服务器
	cd frontend && npm run dev

dev-all:  ## 同时启动前后端
	@echo "启动后端 (端口 8080)..."
	cd backend && go run ./cmd/server/main.go &
	@echo "启动前端 (端口 3000)..."
	cd frontend && npm run dev &
	@wait

# ========== 编译 ==========

build:  ## 编译 Go 后端
	cd backend && CGO_ENABLED=0 go build -o bin/server ./cmd/server/main.go && echo "✅ Backend built: backend/bin/server"

build-fe:  ## 编译前端
	cd frontend && npm run build && echo "✅ Frontend built"

test:  ## 运行 Go 测试
	cd backend && go test ./... -v

test-fe:  ## 运行前端测试（如果有）
	cd frontend && npm test 2>/dev/null || echo "No frontend tests configured"

lint:  ## Go 代码检查
	cd backend && go vet ./...

lint-fe:  ## 前端代码检查
	cd frontend && npm run lint 2>/dev/null || echo "No frontend lint configured"

# ========== Docker ==========

docker-up:  ## 启动全部 Docker 服务
	@echo "🐳 Starting Docker Compose..."
	docker compose up -d
	@echo "✅ Services: Nginx:80, API:8080, Frontend:3000, Postgres:5432"

docker-down:  ## 停止全部 Docker 服务
	docker compose down

docker-logs:  ## 查看所有 Docker 日志
	docker compose logs -f

docker-build:  ## 构建所有 Docker 镜像
	docker compose build

docker-rebuild:  ## 重新构建并启动
	docker compose up -d --build

# ========== 数据库 ==========

db-migrate:  ## 运行数据库迁移
	cd backend && go run ./cmd/migrate/

db-create:  ## 创建新的迁移文件 (make db-create NAME=add_xxx)
	cd backend && touch db/migrations/$(shell date +%s)_$(NAME).up.sql && \
	touch db/migrations/$(shell date +%s)_$(NAME).down.sql && \
	echo "Migration files created"

db-reset:  ## 重置数据库（危险操作！）
	@echo "⚠️  This will delete ALL data!"
	@read -p "Are you sure? [y/N] " -r reply && [ "$$reply" = "y" ] && \
	docker compose down -v && docker compose up -d postgres redis && \
	sleep 3 && cd backend && go run ./cmd/migrate/

# ========== 工具安装 ==========

install-tools:  ## 安装开发工具
	go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# ========== 部署 ==========
# 目标服务器不再硬编码，使用时显式传入：
#   make deploy DEPLOY_HOST=root@1.2.3.4
DEPLOY_HOST ?=

check-host:
	@test -n "$(DEPLOY_HOST)" || (echo "❌ 请设置 DEPLOY_HOST，例如: make deploy DEPLOY_HOST=root@1.2.3.4"; exit 1)

deploy: build check-host  ## 编译并部署到服务器（热更新）
	scp backend/bin/server $(DEPLOY_HOST):/opt/kada/backend/bin/server.new
	ssh $(DEPLOY_HOST) "mv /opt/kada/backend/bin/server.new /opt/kada/backend/bin/server && systemctl restart kada-api && echo '✅ Deployed'"

deploy-fe: build-fe-pkg  ## 编译并部署前端到服务器（热更新）
	ssh $(DEPLOY_HOST) "cd /opt/kada/frontend && rm -rf .next/static node_modules server.js package.json static 2>/dev/null; tar xzf /tmp/kada-fe-standalone.tar.gz && mkdir -p .next && tar xzf /tmp/kada-fe-static.tar.gz && mv static .next/static && rm -f /tmp/kada-fe-*.tar.gz && systemctl restart kada-frontend"
	@echo "✅ Frontend deployed"

build-fe-pkg: build-fe check-host  ## 打包前端
	cd frontend/.next/standalone && tar czf /tmp/kada-fe-standalone.tar.gz .
	cd frontend && tar czf /tmp/kada-fe-static.tar.gz -C .next static/
	scp /tmp/kada-fe-standalone.tar.gz /tmp/kada-fe-static.tar.gz $(DEPLOY_HOST):/tmp/
	@echo "✅ Packages uploaded"

deploy-nginx: check-host  ## 更新Nginx配置
	scp nginx/nginx-prod.conf $(DEPLOY_HOST):/opt/kada/nginx/
	ssh $(DEPLOY_HOST) "docker restart kada-nginx"

deploy-all: deploy deploy-fe  ## 同时部署后端和前端

# ========== CI/CD 辅助 ==========

lint-ci:  ## 运行 golangci-lint（本地 CI 模拟）
	cd backend && golangci-lint run --timeout=5m ./...

lint-fe-ci:  ## 运行前端 lint（本地 CI 模拟）
	cd frontend && npm run lint

test-race:  ## 运行 Go 测试 + 竞态检测（CI 模式）
	cd backend && go test ./... -v -count=1 -race -coverprofile=coverage.out

build-pkg: build build-fe  ## 构建并打包（模拟 CI deploy）
	cd frontend && mkdir -p ../deploy-pkg && \
		tar czf ../deploy-pkg/kada-fe-standalone.tar.gz -C .next/standalone . && \
		tar czf ../deploy-pkg/kada-fe-static.tar.gz -C .next static/
	@echo "✅ Packages ready in deploy-pkg/"

release-dry-run:  ## 模拟 release 流程（测试 Docker 构建）
	docker build -t kada-backend:test ./backend
	docker build -t kada-frontend:test ./frontend
	@echo "✅ Docker images built locally"

# ========== 清理 ==========

clean:  ## 清理构建文件
	rm -rf backend/bin/ frontend/.next/
	docker compose down -v 2>/dev/null || true
