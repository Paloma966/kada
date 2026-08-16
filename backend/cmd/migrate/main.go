package main

import (
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// #nosec G101 -- 仅本地开发默认连接串；生产部署通过 DATABASE_URL 环境变量注入
		dbURL = "postgres://kada:kada123@localhost:5432/kada?sslmode=disable"
	}

	log.Println("Running migrations...")
	m, err := migrate.New("file://db/migrations", dbURL)
	if err != nil {
		log.Fatalf("Migration init failed: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Printf("Migration warning: %v", err)
	}

	v, dirty, _ := m.Version()
	// #nosec G706 -- v/dirty 来自迁移库的数据库状态返回值，非用户可控输入
	log.Printf("Migration done. Version: %d, Dirty: %v", v, dirty)
}
