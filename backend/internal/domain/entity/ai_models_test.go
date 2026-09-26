package entity

import (
	"reflect"
	"testing"
)

// The knowledge base is the one model kept out of Models() (see the comment on AIKnowledgeChunk): it
// needs the pgvector extension, so AutoMigrate on a PostgreSQL without it must not see the model at all.
// These assertions are the price of that exception - the model still has to describe a table the queries
// can use, and it still has to stay out of Models().
func TestAIKnowledgeChunkIsNotAutoMigrated(t *testing.T) {
	for _, model := range Models() {
		if reflect.TypeOf(model) == reflect.TypeOf(&AIKnowledgeChunk{}) {
			t.Fatal("AIKnowledgeChunk is in Models(); its vector column would break AutoMigrate on a " +
				"PostgreSQL without the pgvector extension. cmd/ai-ingest migrates it instead")
		}
	}
}

func TestAIKnowledgeChunkColumns(t *testing.T) {
	s := parse(t, &AIKnowledgeChunk{})

	if s.Table != "ai_knowledge_chunks" {
		t.Errorf("AIKnowledgeChunk maps to table %q, want %q", s.Table, "ai_knowledge_chunks")
	}

	for _, column := range []string{"id", "source", "chunk_index", "content", "embedding", "created_at"} {
		if _, ok := s.FieldsByDBName[column]; !ok {
			t.Errorf("ai_knowledge_chunks is missing column %s", column)
		}
	}

	// The dimension is the part that matters: text-embedding-v3 produces 1024 values by default, and a
	// column declared with another length accepts the insert and then matches nothing.
	embedding, ok := s.FieldsByDBName["embedding"]
	if !ok {
		t.Fatal("ai_knowledge_chunks.embedding: field not found")
	}
	if embedding.DataType != "vector(1024)" {
		t.Errorf("embedding is %q, want %q", embedding.DataType, "vector(1024)")
	}
}
