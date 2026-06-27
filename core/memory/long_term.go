package memory

import (
	"context"
	"fmt"
)

// Embedder converts text to a float64 vector.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float64, error)
}

// SearchResult is a retrieved memory item.
type SearchResult struct {
	ID       string
	Score    float64
	Metadata map[string]any
}

// VectorStore stores and retrieves embedding vectors.
type VectorStore interface {
	Insert(ctx context.Context, id string, vector []float64, metadata map[string]any) error
	Search(ctx context.Context, vector []float64, topK int) ([]SearchResult, error)
}

// LongTermMemory persists and retrieves memories via a VectorStore.
type LongTermMemory struct {
	embedder    Embedder
	vectorStore VectorStore
}

// NewLongTermMemory creates a long-term memory backed by the given stores.
func NewLongTermMemory(embedder Embedder, store VectorStore) *LongTermMemory {
	return &LongTermMemory{embedder: embedder, vectorStore: store}
}

// Remember embeds and stores a memory string.
func (m *LongTermMemory) Remember(ctx context.Context, text string, metadata map[string]any) error {
	if m.embedder == nil || m.vectorStore == nil {
		return fmt.Errorf("long-term memory: embedder and vectorStore must be set")
	}
	vector, err := m.embedder.Embed(ctx, text)
	if err != nil {
		return fmt.Errorf("embedding: %w", err)
	}
	id := fmt.Sprintf("mem-%x", vector[:min(8, len(vector))])
	return m.vectorStore.Insert(ctx, id, vector, metadata)
}

// Recall retrieves the top-k most relevant memories for a query.
func (m *LongTermMemory) Recall(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	if m.embedder == nil || m.vectorStore == nil {
		return nil, fmt.Errorf("long-term memory: embedder and vectorStore must be set")
	}
	vector, err := m.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embedding: %w", err)
	}
	return m.vectorStore.Search(ctx, vector, topK)
}
