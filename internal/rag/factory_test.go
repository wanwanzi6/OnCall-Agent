package rag

import (
	"context"
	"testing"

	"oncall-agent/internal/infra/config"
	embedderimpl "oncall-agent/internal/rag/embedder"
	indexerimpl "oncall-agent/internal/rag/indexer"
)

func TestNewEmbedderDefaultReturnsMock(t *testing.T) {
	cfg := baseFactoryConfig()

	embedder, err := NewEmbedder(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewEmbedder returned error: %v", err)
	}
	if _, ok := embedder.(*embedderimpl.MockEmbedder); !ok {
		t.Fatalf("embedder type = %T, want *MockEmbedder", embedder)
	}
}

func TestNewVectorStoreDefaultReturnsMemory(t *testing.T) {
	cfg := baseFactoryConfig()

	store, err := NewVectorStore(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewVectorStore returned error: %v", err)
	}
	if _, ok := store.(*indexerimpl.MemoryVectorStore); !ok {
		t.Fatalf("store type = %T, want *MemoryVectorStore", store)
	}
}

func TestNewEmbedderRejectsUnknownProvider(t *testing.T) {
	cfg := baseFactoryConfig()
	cfg.RAG.EmbedderProvider = "unknown"

	if _, err := NewEmbedder(context.Background(), cfg); err == nil {
		t.Fatal("expected error")
	}
}

func TestNewVectorStoreRejectsUnknownProvider(t *testing.T) {
	cfg := baseFactoryConfig()
	cfg.RAG.VectorStoreProvider = "unknown"

	if _, err := NewVectorStore(context.Background(), cfg); err == nil {
		t.Fatal("expected error")
	}
}

func TestNewEmbedderRejectsMissingDashScopeAPIKey(t *testing.T) {
	cfg := baseFactoryConfig()
	cfg.RAG.EmbedderProvider = EmbedderProviderDashScope
	cfg.Embedding.DashScope.APIKey = ""

	if _, err := NewEmbedder(context.Background(), cfg); err == nil {
		t.Fatal("expected error")
	}
}

func TestNewVectorStoreRejectsInvalidMilvusConfig(t *testing.T) {
	cfg := baseFactoryConfig()
	cfg.RAG.VectorStoreProvider = VectorStoreProviderMilvus
	cfg.Milvus.Address = ""

	if _, err := NewVectorStore(context.Background(), cfg); err == nil {
		t.Fatal("expected error")
	}
}

func baseFactoryConfig() config.Config {
	return config.Config{
		RAG: config.RAGConfig{
			EmbeddingDim:        64,
			DefaultTopK:         3,
			EmbedderProvider:    EmbedderProviderMock,
			VectorStoreProvider: VectorStoreProviderMemory,
		},
		Embedding: config.EmbeddingConfig{
			DashScope: config.DashScopeEmbeddingConfig{
				Model:      "text-embedding-v4",
				Dimensions: 1024,
			},
		},
		Milvus: config.MilvusConfig{
			Address:     "localhost:19530",
			Database:    "agent",
			Collection:  "oncall_knowledge",
			VectorField: "vector",
		},
	}
}
