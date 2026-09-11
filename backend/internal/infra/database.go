package infra

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/chun/kada-backend/internal/domain/entity"
	"github.com/chun/kada-backend/internal/infra/schema"
)

// NewDB opens the PostgreSQL connection pool through GORM and verifies it with a ping.
//
// AutoMigrate is deliberately NOT called here: the caller decides (see Migrate) so that the API server,
// the Kafka worker and any future CLI can each choose whether they are allowed to change the schema.
func NewDB(databaseURL string) (*gorm.DB, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is empty")
	}

	cfg := &gorm.Config{
		// Map PostgreSQL SQLSTATEs onto gorm.ErrDuplicatedKey / gorm.ErrForeignKeyViolated so the
		// services can branch on errors.Is instead of matching driver-specific error codes.
		TranslateError: true,
		Logger:         gormLogger(),
	}

	db, err := gorm.Open(postgres.Open(databaseURL), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to access the underlying connection pool: %w", err)
	}

	// Connection pool. MaxOpenConns used to be pgxpool's default (greater than the number of CPUs);
	// these values are explicit so the behavior is the same on every machine.
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)
	sqlDB.SetConnMaxLifetime(time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("✅ Database connected")
	return db, nil
}

// gormLogger quiets GORM's per-statement logging in release mode; outside release it logs slow queries
// and errors, which is what a developer wants while working on the API.
func gormLogger() logger.Interface {
	level := logger.Warn
	if os.Getenv("GIN_MODE") != "release" {
		level = logger.Info
	}
	return logger.Default.LogMode(level)
}

// Migrate brings the schema up to date with the entity definitions.
//
// This replaces the golang-migrate SQL files in db/migrations: the structs are now the single source of
// truth, so a column that exists in the database but not in code (or the reverse) can no longer happen.
// AutoMigrate only creates missing tables, columns, indexes and constraints; it never drops anything.
//
// Order matters for a database created from the original raw-SQL migrations: the legacy constraint names
// have to be reconciled first, otherwise AutoMigrate aborts while trying to drop a constraint that exists
// under PostgreSQL's generated name instead of the one GORM derives from the model.
func Migrate(db *gorm.DB) error {
	if err := schema.ReconcileLegacyConstraints(db); err != nil {
		return err
	}
	if err := db.AutoMigrate(entity.Models()...); err != nil {
		return fmt.Errorf("auto migration failed: %w", err)
	}
	log.Println("Database schema is up to date")
	return nil
}

// CloseDB closes the underlying connection pool.
func CloseDB(db *gorm.DB) {
	if db == nil {
		return
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("failed to access the connection pool while closing: %v", err)
		return
	}
	if err := sqlDB.Close(); err != nil {
		log.Printf("failed to close the database connection: %v", err)
		return
	}
	log.Println("Database connection closed")
}
