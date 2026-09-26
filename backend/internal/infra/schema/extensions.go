package schema

import (
	"fmt"

	"gorm.io/gorm"
)

// extensionStatements are the PostgreSQL extensions the models depend on.
//
// A model whose column is tagged `type:vector(1024)` cannot be created before the extension that defines
// the type, in the same way an enum column cannot be created before CREATE TYPE. The extension itself
// ships with the server (the official postgres image does not have it; pgvector/pgvector does, and the
// deploy script installs it for the AI database), so this is about enabling it in each database that
// needs it - CREATE EXTENSION is per database, not per server.
var extensionStatements = []string{
	`CREATE EXTENSION IF NOT EXISTS vector`,
}

// ExtensionStatements returns the idempotent DDL that enables the extensions the models reference.
func ExtensionStatements() []string {
	return append([]string(nil), extensionStatements...)
}

// EnsureExtensions enables the PostgreSQL extensions the models depend on.
//
// It must run BEFORE AutoMigrate, and it is only reached from the paths that are allowed to create the
// tables needing them: `cmd/migrate` deliberately does not call this, so a developer machine whose
// PostgreSQL has no pgvector can still run the API and the migrations (see entity.AIKnowledgeChunk).
func EnsureExtensions(db *gorm.DB) error {
	if db.Name() != "postgres" {
		// CREATE EXTENSION is PostgreSQL-specific.
		return nil
	}
	for _, statement := range extensionStatements {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("failed to enable a required extension: %w", err)
		}
	}
	return nil
}
