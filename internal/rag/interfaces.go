package rag

import (
	"context"

	"oncall-agent/internal/model/domain"
)

type Loader interface {
	Load(ctx context.Context, filePath string) (domain.Document, error)
}

type Splitter interface {
	Split(ctx context.Context, doc domain.Document) ([]domain.Chunk, error)
}

type Embedder interface {
	EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error)
	EmbedQuery(ctx context.Context, text string) ([]float32, error)
}

type VectorStore interface {
	Upsert(ctx context.Context, chunks []domain.Chunk, vectors [][]float32) error
	Search(ctx context.Context, vector []float32, topK int) ([]domain.SearchResult, error)
	DeleteByDocumentID(ctx context.Context, documentID string) error
}
