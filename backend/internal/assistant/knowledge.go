package assistant

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/chun/kada-backend/internal/domain/entity"
)

// KnowledgeBase is the assistant's memory: passages stored with their vectors, searched by cosine
// distance.
type KnowledgeBase struct {
	db       *gorm.DB
	embedder Embedder
}

// NewKnowledgeBase builds the knowledge base on an open database handle.
func NewKnowledgeBase(db *gorm.DB, embedder Embedder) *KnowledgeBase {
	return &KnowledgeBase{db: db, embedder: embedder}
}

// embedBatchSize is how many passages go into one embedding request. DashScope's compatible endpoint
// rejects oversized batches, and ten keeps a request well inside the limit while still amortizing the
// round trip over a whole document.
const embedBatchSize = 10

// Replace rewrites the knowledge base with the passages of the given documents and returns how many
// passages were stored.
//
// A full rebuild, in one transaction: the passages are embedded first (the slow part, and the part that
// talks to somebody else's API) and only then swapped in, so a failure half way through leaves the
// previous knowledge in place rather than a knowledge base missing its second half.
func (k *KnowledgeBase) Replace(ctx context.Context, docs []Document) (int, error) {
	passages, err := k.passages(ctx, docs)
	if err != nil {
		return 0, err
	}

	err = k.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// GORM refuses an unconditional DELETE, and this delete really is unconditional: the table is a
		// cache of the bundled documents, rebuilt from them, with no user data in it.
		if delErr := tx.Where("1 = 1").Delete(&entity.AIKnowledgeChunk{}).Error; delErr != nil {
			return fmt.Errorf("failed to clear the knowledge base: %w", delErr)
		}
		if len(passages) == 0 {
			return nil
		}
		return tx.CreateInBatches(passages, 100).Error
	})
	if err != nil {
		return 0, fmt.Errorf("failed to store the knowledge base: %w", err)
	}
	return len(passages), nil
}

// passages chunks the documents and embeds every passage, without touching the database.
//
// Split out from Replace so the whole embedding path - batching, the row each passage becomes, and what a
// provider that returns the wrong number of vectors does - is testable without a database.
func (k *KnowledgeBase) passages(ctx context.Context, docs []Document) ([]entity.AIKnowledgeChunk, error) {
	var passages []entity.AIKnowledgeChunk

	for _, doc := range docs {
		texts := chunk(doc.Text, passageSize, passageOverlap)
		for start := 0; start < len(texts); start += embedBatchSize {
			end := start + embedBatchSize
			if end > len(texts) {
				end = len(texts)
			}

			vectors, err := k.embedder.EmbedStrings(ctx, texts[start:end])
			if err != nil {
				return nil, fmt.Errorf("failed to embed %s: %w", doc.Source, err)
			}
			if len(vectors) != end-start {
				return nil, fmt.Errorf("embedding %s returned %d vectors for %d passages",
					doc.Source, len(vectors), end-start)
			}

			for i, vector := range vectors {
				passages = append(passages, entity.AIKnowledgeChunk{
					Source:     doc.Source,
					ChunkIndex: start + i,
					Content:    texts[start+i],
					Embedding:  vectorLiteral(vector),
				})
			}
		}
	}
	return passages, nil
}

// Retrieve returns the passages closest to the query, nearest first.
//
// The query is embedded here rather than by the caller, because the vector and the text it came from must
// not drift apart: a passage is only found if the question was embedded with the same model.
//
// There is no vector index. The knowledge base is a couple of dozen passages, where a sequential scan is
// faster than the index maintenance, and an index would be one more thing to keep in step with the
// embedding dimension. If the corpus grows, the statement to add is
// `CREATE INDEX ... USING hnsw (embedding vector_cosine_ops)`.
func (k *KnowledgeBase) Retrieve(ctx context.Context, query string, limit int) ([]string, error) {
	if limit <= 0 {
		return nil, nil
	}

	// A knowledge base built without an embedding key (see NewEmbedder) is a legitimate configuration: the
	// assistant answers from the model alone. Saying so here is what keeps that from being a panic.
	if k.embedder == nil {
		return nil, errors.New("the embedding API key is not configured")
	}

	vectors, err := k.embedder.EmbedStrings(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("failed to embed the question: %w", err)
	}
	if len(vectors) == 0 {
		return nil, errors.New("the embedding service returned no vector for the question")
	}

	// The cast is required: the operator is vector <=> vector, and the bound parameter arrives as text.
	var rows []entity.AIKnowledgeChunk
	err = k.search(k.db.WithContext(ctx), vectorLiteral(vectors[0]), limit).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to search the knowledge base: %w", err)
	}

	passages := make([]string, 0, len(rows))
	for _, row := range rows {
		passages = append(passages, row.Content)
	}
	return passages, nil
}

// search builds the nearest-neighbor query, nearest first.
//
// Split out of Retrieve so the one clause in this package that a reader cannot check by reading - the
// distance operator and its cast - is asserted against the SQL GORM actually renders, in a dry-run test.
//
// The order is a clause.OrderBy and not Order(gorm.Expr(...)): Order() switches on a fixed set of types
// and a bare clause.Expr matches none of them, so it drops the clause and returns the passages in table
// order - a wrong answer that looks like a working search. The dry-run test above is what found that.
func (k *KnowledgeBase) search(query *gorm.DB, vector string, limit int) *gorm.DB {
	return query.
		Model(&entity.AIKnowledgeChunk{}).
		Select("content").
		Order(clause.OrderBy{Expression: clause.Expr{
			// The cast is required: the operator is vector <=> vector, and the bound parameter arrives as text.
			SQL:                "embedding <=> ?::vector",
			Vars:               []any{vector},
			WithoutParentheses: true,
		}}).
		Limit(limit)
}
