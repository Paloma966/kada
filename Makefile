.PHONY: help dev dev-fe build build-fe test lint docker-up docker-down docker-logs \
        docker-build db-migrate db-create db-reset sqlc-gen install-tools clean

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

# ========== sqlc ==========

sqlc-gen:  ## 生成 sqlc 代码
	cd backend && sqlc generate

# ========== 工具安装 ==========

install-tools:  ## 安装开发工具
	go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# ========== 清理 ==========

clean:  ## 清理构建文件
	rm -rf backend/bin/ frontend/.next/
	docker compose down -v 2>/dev/null || true
