.PHONY: help dev build test docker-up docker-down db-migrate db-create db-reset

help:  ## 显示帮助信息
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ========== 开发 ==========

dev:  ## 启动开发环境（Go 后端热加载）
	cd backend && go run ./cmd/server/main.go

build:  ## 编译 Go 后端
	cd backend && CGO_ENABLED=0 go build -o bin/server ./cmd/server/main.go

test:  ## 运行 Go 测试
	cd backend && go test ./... -v

lint:  ## 代码检查
	cd backend && go vet ./...

# ========== Docker ==========

docker-up:  ## 启动 Docker Compose 环境
	docker compose up -d

docker-down:  ## 停止 Docker Compose 环境
	docker compose down

docker-logs:  ## 查看 Docker 日志
	docker compose logs -f

# ========== 数据库 ==========

db-migrate:  ## 运行数据库迁移
	cd backend && go run github.com/golang-migrate/migrate/v4 \
		-database "$(DATABASE_URL)" \
		-path db/migrations up

db-create:  ## 创建新的迁移文件（使用: make db-create NAME=add_users）
	cd backend && go run github.com/golang-migrate/migrate/v4 \
		create -ext sql -dir db/migrations $(NAME)

db-reset:  ## 重置数据库（删除所有数据！）
	cd backend && go run github.com/golang-migrate/migrate/v4 \
		-database "$(DATABASE_URL)" \
		-path db/migrations down

# ========== sqlc ==========

sqlc-gen:  ## 生成 sqlc 代码
	cd backend && sqlc generate

# ========== 工具安装 ==========

install-tools:  ## 安装开发工具
	go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
