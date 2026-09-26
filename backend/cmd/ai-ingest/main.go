// Command ai-ingest rebuilds the assistant's knowledge base.
//
// It is a command rather than a startup step: indexing the corpus costs one embedding request per passage,
// and a server restart should not depend on somebody else's API being reachable. Run it after changing
// internal/ai/knowledge or the embedding model.
//
// It is also the only command that creates the knowledge table, because that table's vector column needs
// the pgvector extension - see the comment on entity.AIKnowledgeChunk for why the API itself must not
// require it.
package main

import (
	"context"
	"log"
	"time"

	"github.com/joho/godotenv"

	"github.com/chun/kada-backend/config"
	"github.com/chun/kada-backend/internal/assistant"
	"github.com/chun/kada-backend/internal/domain/entity"
	"github.com/chun/kada-backend/internal/infra"
	"github.com/chun/kada-backend/internal/infra/schema"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	db, err := infra.NewDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer infra.CloseDB(db)

	if err = schema.EnsureExtensions(db); err != nil {
		log.Fatalf("The pgvector extension is not available in this database: %v", err)
	}
	if err = db.AutoMigrate(&entity.AIKnowledgeChunk{}); err != nil {
		log.Fatalf("Creating the knowledge table failed: %v", err)
	}

	// One request per batch, over a corpus small enough to index in seconds; five minutes is a ceiling
	// that a stuck request cannot turn into a command that never returns.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	embedder, err := assistant.NewEmbedder(ctx, cfg.AIEmbeddingAPIKey, cfg.AIEmbeddingModel)
	if err != nil {
		log.Fatalf("Embedding client failed: %v", err)
	}

	docs, err := assistant.Documents()
	if err != nil {
		log.Fatalf("Reading the bundled knowledge base failed: %v", err)
	}

	passages, err := assistant.NewKnowledgeBase(db, embedder).Replace(ctx, docs)
	if err != nil {
		log.Fatalf("Ingesting the knowledge base failed: %v", err)
	}

	log.Printf("Knowledge base rebuilt from %d document(s): %d passages", len(docs), passages)
}
