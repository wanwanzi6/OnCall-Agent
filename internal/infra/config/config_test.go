package config

import (
	"path/filepath"
	"testing"
)

func TestLoadDefaultsAndEnvOverride(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("MOCK_ENABLED", "false")
	t.Setenv("KNOWLEDGE_UPLOAD_DIR", "tmp/uploads")
	t.Setenv("KNOWLEDGE_MAX_FILE_SIZE_BYTES", "1024")
	t.Setenv("RAG_CHUNK_SIZE", "400")
	t.Setenv("RAG_CHUNK_OVERLAP", "50")
	t.Setenv("RAG_EMBEDDING_DIM", "32")
	t.Setenv("RAG_DEFAULT_TOP_K", "5")
	t.Setenv("RAG_EMBEDDER_PROVIDER", "dashscope")
	t.Setenv("RAG_VECTOR_STORE_PROVIDER", "milvus")
	t.Setenv("DASHSCOPE_API_KEY", "test-key")
	t.Setenv("DASHSCOPE_EMBEDDING_MODEL", "text-embedding-v4")
	t.Setenv("DASHSCOPE_EMBEDDING_DIM", "1024")
	t.Setenv("DASHSCOPE_EMBEDDING_TIMEOUT", "15s")
	t.Setenv("MILVUS_ADDRESS", "127.0.0.1:19530")
	t.Setenv("MILVUS_DATABASE", "testdb")
	t.Setenv("MILVUS_COLLECTION", "test_collection")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.App.Env != "test" {
		t.Fatalf("env = %q, want test", cfg.App.Env)
	}
	if cfg.Server.Port != 9090 {
		t.Fatalf("port = %d, want 9090", cfg.Server.Port)
	}
	if cfg.Mock.Enabled {
		t.Fatal("mock should be disabled by env")
	}
	if cfg.Knowledge.UploadDir != "tmp/uploads" {
		t.Fatalf("upload dir = %q", cfg.Knowledge.UploadDir)
	}
	if cfg.Knowledge.MaxFileSizeBytes != 1024 {
		t.Fatalf("max size = %d, want 1024", cfg.Knowledge.MaxFileSizeBytes)
	}
	if cfg.RAG.ChunkSize != 400 || cfg.RAG.ChunkOverlap != 50 || cfg.RAG.EmbeddingDim != 32 || cfg.RAG.DefaultTopK != 5 {
		t.Fatalf("rag config = %+v", cfg.RAG)
	}
	if cfg.RAG.EmbedderProvider != "dashscope" || cfg.RAG.VectorStoreProvider != "milvus" {
		t.Fatalf("providers = %+v", cfg.RAG)
	}
	if cfg.Embedding.DashScope.APIKey != "test-key" || cfg.Embedding.DashScope.Dimensions != 1024 {
		t.Fatalf("dashscope config = %+v", cfg.Embedding.DashScope)
	}
	if cfg.Milvus.Address != "127.0.0.1:19530" || cfg.Milvus.Collection != "test_collection" {
		t.Fatalf("milvus config = %+v", cfg.Milvus)
	}
}

func TestLoadExampleConfigWithDurations(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "..", "configs", "config.example.yaml"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.RAG.EmbedderProvider != "mock" || cfg.RAG.VectorStoreProvider != "memory" {
		t.Fatalf("providers = %+v", cfg.RAG)
	}
	if cfg.Embedding.DashScope.Timeout == 0 || cfg.Milvus.Timeout == 0 {
		t.Fatalf("timeouts should be parsed: embedding=%s milvus=%s", cfg.Embedding.DashScope.Timeout, cfg.Milvus.Timeout)
	}
}
