// Package ai is the assistant: the knowledge base it answers from, the model it answers with and the
// tools it may call.
//
// It runs inside the API process and replaces a separate Python service (backend/ai) that this gateway
// used to reverse-proxy to. The model and embedding clients come from Eino (github.com/cloudwego/eino),
// which is also where the tool-calling loop lives; the knowledge base is PostgreSQL's pgvector, kept in
// the same database as everything else.
package ai

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/embedding/dashscope"
)

// defaultEmbeddingModel is DashScope's text-embedding-v3, which produces the 1024 dimensions
// entity.AIKnowledgeChunk declares. The two have to agree: a column of another length accepts the insert
// and then matches nothing.
const defaultEmbeddingModel = "text-embedding-v3"

// Embedder turns text into vectors.
//
// It is deliberately narrower than eino's embedding.Embedder, whose variadic options no test fake can
// satisfy exactly - and the knowledge base only ever needs this one call.
type Embedder interface {
	EmbedStrings(ctx context.Context, texts []string) ([][]float64, error)
}

// NewEmbedder builds the DashScope (Aliyun Bailian) embedder the knowledge base is indexed with.
//
// An empty key is an error rather than a client that fails on first use: without it the assistant would
// still answer, just from no retrieved context at all, which looks exactly like a knowledge base that
// holds nothing relevant.
func NewEmbedder(ctx context.Context, apiKey, model string) (Embedder, error) {
	if apiKey == "" {
		return nil, errors.New("the embedding API key is empty")
	}
	if model == "" {
		model = defaultEmbeddingModel
	}

	inner, err := dashscope.NewEmbedder(ctx, &dashscope.EmbeddingConfig{
		APIKey:  apiKey,
		Model:   model,
		Timeout: 30 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build the embedding client: %w", err)
	}
	return dashscopeEmbedder{inner: inner}, nil
}

type dashscopeEmbedder struct {
	inner *dashscope.Embedder
}

func (e dashscopeEmbedder) EmbedStrings(ctx context.Context, texts []string) ([][]float64, error) {
	return e.inner.EmbedStrings(ctx, texts)
}

// vectorLiteral renders a vector in pgvector's text form ("[0.1,0.2]"), which is how a bound parameter
// reaches a vector column and the <=> operator.
//
// FormatFloat with 'f' and no precision keeps every digit the float has without switching to exponent
// notation, which pgvector also accepts but no reader expects.
func vectorLiteral(vector []float64) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, value := range vector {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(value, 'f', -1, 64))
	}
	b.WriteByte(']')
	return b.String()
}
