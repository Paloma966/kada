package assistant

import (
	"context"
	"errors"
	"strings"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/chun/kada-backend/internal/domain/entity"
)

// newDryRunDB builds a GORM handle that renders SQL without contacting a server. It mirrors the helper in
// internal/service: the queries here are the only place where a mistake is invisible until it runs, and
// the knowledge base needs a PostgreSQL with pgvector, which is not available everywhere this test runs.
func newDryRunDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(postgres.New(postgres.Config{
		// #nosec G101 -- a throwaway DSN that is never dialed: DryRun only renders the SQL.
		DSN: "postgres://kada:invalid@127.0.0.1:1/kada?sslmode=disable",
	}), &gorm.Config{DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("failed to build a dry-run gorm.DB: %v", err)
	}
	return db
}

// fakeEmbedder stands in for the DashScope client: it records what it was asked to embed and hands back
// deterministic vectors, so the whole indexing path can be tested without a key or a network.
type fakeEmbedder struct {
	vectors [][]float64
	err     error
	batches [][]string
}

func (f *fakeEmbedder) EmbedStrings(_ context.Context, texts []string) ([][]float64, error) {
	f.batches = append(f.batches, append([]string(nil), texts...))
	if f.err != nil {
		return nil, f.err
	}
	if f.vectors != nil {
		return f.vectors, nil
	}
	vectors := make([][]float64, len(texts))
	for i := range texts {
		vectors[i] = []float64{float64(i), 0.5}
	}
	return vectors, nil
}

// A document longer than embedBatchSize passages is the case that breaks a naive implementation: the
// caller has to slice it into batches, and the ChunkIndex of the second batch has to keep counting rather
// than restart at zero.
func TestPassagesEmbedEveryChunkInBatches(t *testing.T) {
	text := strings.Repeat("短链接平台的一段知识，用来把知识库撑到多批。\n\n", 250)
	want := chunk(text, passageSize, passageOverlap)
	if len(want) <= embedBatchSize {
		t.Fatalf("the case needs more than %d passages to exercise batching, got %d", embedBatchSize, len(want))
	}

	embedder := &fakeEmbedder{}
	kb := NewKnowledgeBase(newDryRunDB(t), embedder)

	passages, err := kb.passages(context.Background(), []Document{{Source: "kada.md", Text: text}})
	if err != nil {
		t.Fatalf("passages() failed: %v", err)
	}
	if len(passages) != len(want) {
		t.Fatalf("got %d passages, want %d", len(passages), len(want))
	}

	for i, passage := range passages {
		if passage.Source != "kada.md" {
			t.Errorf("passage %d has source %q, want %q", i, passage.Source, "kada.md")
		}
		if passage.ChunkIndex != i {
			t.Errorf("passage %d has chunk index %d, want %d", i, passage.ChunkIndex, i)
		}
		if passage.Content != want[i] {
			t.Errorf("passage %d is %q, want %q", i, passage.Content, want[i])
		}
		if !strings.HasPrefix(passage.Embedding, "[") || !strings.HasSuffix(passage.Embedding, "]") {
			t.Errorf("passage %d has embedding %q, want a pgvector literal", i, passage.Embedding)
		}
	}

	if len(embedder.batches) < 2 {
		t.Fatalf("expected more than one embedding request, got %d", len(embedder.batches))
	}
	embedded := 0
	for i, batch := range embedder.batches {
		if len(batch) > embedBatchSize {
			t.Errorf("batch %d holds %d texts, want at most %d", i, len(batch), embedBatchSize)
		}
		embedded += len(batch)
	}
	if embedded != len(want) {
		t.Errorf("embedded %d texts, want %d", embedded, len(want))
	}
}

// A provider that answers with fewer vectors than it was given texts is the failure that would silently
// store the wrong text against a vector, so it has to be an error rather than a shorter knowledge base.
func TestPassagesRejectsAMismatchedVectorCount(t *testing.T) {
	embedder := &fakeEmbedder{vectors: [][]float64{{1, 2}}}
	kb := NewKnowledgeBase(newDryRunDB(t), embedder)

	_, err := kb.passages(context.Background(), []Document{
		{Source: "kada.md", Text: strings.Repeat("知识。", 400)},
	})
	if err == nil {
		t.Fatal("expected an error when the provider returns fewer vectors than passages")
	}
	if !strings.Contains(err.Error(), "returned 1 vectors") {
		t.Errorf("error does not name the counts: %v", err)
	}
}

func TestPassagesPropagatesAnEmbeddingFailure(t *testing.T) {
	embedder := &fakeEmbedder{err: errors.New("quota exceeded")}
	kb := NewKnowledgeBase(newDryRunDB(t), embedder)

	_, err := kb.passages(context.Background(), []Document{{Source: "kada.md", Text: "知识。"}})
	if err == nil {
		t.Fatal("expected the embedding failure to surface")
	}
	if !strings.Contains(err.Error(), "kada.md") || !errors.Is(err, embedder.err) {
		t.Errorf("error should name the document and wrap the cause: %v", err)
	}
}

// The distance operator is the whole of the retrieval, and it is the one clause a reader cannot verify by
// reading: a missing cast makes PostgreSQL reject the statement, and the wrong operator returns the
// *furthest* passages instead of the nearest.
func TestRetrieveSearchesByCosineDistance(t *testing.T) {
	kb := NewKnowledgeBase(newDryRunDB(t), &fakeEmbedder{})

	stmt := kb.search(newDryRunDB(t), "[0.25,-0.5]", 4).Find(&[]entity.AIKnowledgeChunk{}).Statement
	sql := stmt.SQL.String()

	for _, want := range []string{
		`SELECT "content" FROM "ai_knowledge_chunks"`,
		"ORDER BY embedding <=> $1::vector",
		"LIMIT $2",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("generated SQL is missing %q:\n%s", want, sql)
		}
	}
	if len(stmt.Vars) != 2 {
		t.Errorf("expected 2 bound parameters, got %d: %v", len(stmt.Vars), stmt.Vars)
	}
}

func TestRetrieveEmbedsTheQuestion(t *testing.T) {
	embedder := &fakeEmbedder{}
	kb := NewKnowledgeBase(newDryRunDB(t), embedder)

	if _, err := kb.Retrieve(context.Background(), "怎么创建短链接", 4); err != nil {
		t.Fatalf("Retrieve() failed: %v", err)
	}
	if len(embedder.batches) != 1 || len(embedder.batches[0]) != 1 {
		t.Fatalf("expected exactly one single-text embedding request, got %v", embedder.batches)
	}
	if embedder.batches[0][0] != "怎么创建短链接" {
		t.Errorf("embedded %q, want the question", embedder.batches[0][0])
	}
}

// Nothing to look for, nothing to pay for: a zero limit must not reach the embedding API.
func TestRetrieveWithNoLimitEmbedsNothing(t *testing.T) {
	embedder := &fakeEmbedder{}
	kb := NewKnowledgeBase(newDryRunDB(t), embedder)

	passages, err := kb.Retrieve(context.Background(), "随便问问", 0)
	if err != nil {
		t.Fatalf("Retrieve() failed: %v", err)
	}
	if len(passages) != 0 {
		t.Errorf("got %d passages, want none", len(passages))
	}
	if len(embedder.batches) != 0 {
		t.Errorf("the embedder was called %d times with a zero limit", len(embedder.batches))
	}
}

func TestVectorLiteral(t *testing.T) {
	tests := []struct {
		name   string
		vector []float64
		want   string
	}{
		{name: "plain values", vector: []float64{0.5, -1, 2.25}, want: "[0.5,-1,2.25]"},
		// pgvector parses exponent notation, but a literal that reads as a decimal is what a reader
		// (and a log line) expects, and FormatFloat's 'f' is what keeps it that way.
		{name: "small values stay decimal", vector: []float64{0.0000001}, want: "[0.0000001]"},
		{name: "empty vector", vector: nil, want: "[]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := vectorLiteral(tt.vector); got != tt.want {
				t.Errorf("vectorLiteral() = %q, want %q", got, tt.want)
			}
		})
	}
}
