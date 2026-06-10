package rag

import (
	"context"
	"fmt"
	"strings"

	"oncall-agent/internal/infra/config"
	"oncall-agent/internal/rag/embedder"
	"oncall-agent/internal/rag/indexer"
)

const (
	EmbedderProviderMock      = "mock"
	EmbedderProviderDashScope = "dashscope"

	VectorStoreProviderMemory = "memory"
	VectorStoreProviderMilvus = "milvus"
)

func NewEmbedder(ctx context.Context, cfg config.Config) (Embedder, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.RAG.EmbedderProvider))
	if provider == "" {
		provider = EmbedderProviderMock
	}
	switch provider {
	case EmbedderProviderMock:
		return embedder.NewMockEmbedder(cfg.RAG.EmbeddingDim), nil
	case EmbedderProviderDashScope:
		return embedder.NewDashScopeEmbedder(ctx, cfg.Embedding.DashScope)
	default:
		return nil, fmt.Errorf("unsupported rag embedder provider: %s", provider)
	}
}

func NewVectorStore(ctx context.Context, cfg config.Config) (VectorStore, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.RAG.VectorStoreProvider))
	if provider == "" {
		provider = VectorStoreProviderMemory
	}
	switch provider {
	case VectorStoreProviderMemory:
		return indexer.NewMemoryVectorStore(cfg.RAG.DefaultTopK), nil
	case VectorStoreProviderMilvus:
		return indexer.NewMilvusVectorStore(ctx, cfg.Milvus, cfg.RAG.DefaultTopK, vectorDimension(cfg))
	default:
		return nil, fmt.Errorf("unsupported rag vector store provider: %s", provider)
	}
}

func vectorDimension(cfg config.Config) int {
	if strings.EqualFold(strings.TrimSpace(cfg.RAG.EmbedderProvider), EmbedderProviderDashScope) {
		return cfg.Embedding.DashScope.Dimensions
	}
	return cfg.RAG.EmbeddingDim
}
