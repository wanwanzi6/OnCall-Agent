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
	t.Setenv("AIOPS_ALERT_PROVIDER", "prometheus")
	t.Setenv("AIOPS_LOG_PROVIDER", "mock")
	t.Setenv("AIOPS_METRIC_PROVIDER", "mock")
	t.Setenv("AIOPS_MODE", "agent")
	t.Setenv("AIOPS_FALLBACK_TO_RULE", "false")
	t.Setenv("AIOPS_AGENT_MAX_STEPS", "8")
	t.Setenv("AIOPS_AGENT_TIMEOUT", "45s")
	t.Setenv("AIOPS_TIMEOUT", "7s")
	t.Setenv("AIOPS_SOP_TOP_K", "4")
	t.Setenv("LLM_PROVIDER", "openai-compatible")
	t.Setenv("LLM_API_KEY", "llm-test-key")
	t.Setenv("LLM_BASE_URL", "https://llm.local/v1")
	t.Setenv("LLM_MODEL", "test-model")
	t.Setenv("LLM_TIMEOUT", "9s")
	t.Setenv("DASHSCOPE_API_KEY", "test-key")
	t.Setenv("DASHSCOPE_EMBEDDING_MODEL", "text-embedding-v4")
	t.Setenv("DASHSCOPE_EMBEDDING_DIM", "1024")
	t.Setenv("DASHSCOPE_EMBEDDING_TIMEOUT", "15s")
	t.Setenv("MILVUS_ADDRESS", "127.0.0.1:19530")
	t.Setenv("MILVUS_DATABASE", "testdb")
	t.Setenv("MILVUS_COLLECTION", "test_collection")
	t.Setenv("PROMETHEUS_BASE_URL", "http://prometheus.local:9090")
	t.Setenv("PROMETHEUS_TIMEOUT", "3s")

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
	if cfg.AIOps.AlertProvider != "prometheus" || cfg.AIOps.Timeout.String() != "7s" || cfg.AIOps.SOPTopK != 4 {
		t.Fatalf("aiops config = %+v", cfg.AIOps)
	}
	if cfg.AIOps.Mode != "agent" || cfg.AIOps.FallbackToRule || cfg.AIOps.Agent.MaxSteps != 8 || cfg.AIOps.Agent.Timeout.String() != "45s" {
		t.Fatalf("aiops agent config = %+v", cfg.AIOps)
	}
	if cfg.LLM.Provider != "openai-compatible" || cfg.LLM.APIKey != "llm-test-key" || cfg.LLM.BaseURL != "https://llm.local/v1" || cfg.LLM.Model != "test-model" || cfg.LLM.Timeout.String() != "9s" {
		t.Fatalf("llm config = %+v", cfg.LLM)
	}
	if cfg.Embedding.DashScope.APIKey != "test-key" || cfg.Embedding.DashScope.Dimensions != 1024 {
		t.Fatalf("dashscope config = %+v", cfg.Embedding.DashScope)
	}
	if cfg.Milvus.Address != "127.0.0.1:19530" || cfg.Milvus.Collection != "test_collection" {
		t.Fatalf("milvus config = %+v", cfg.Milvus)
	}
	if cfg.Prometheus.BaseURL != "http://prometheus.local:9090" || cfg.Prometheus.Timeout.String() != "3s" {
		t.Fatalf("prometheus config = %+v", cfg.Prometheus)
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
	if cfg.Embedding.DashScope.Timeout == 0 || cfg.Milvus.Timeout == 0 || cfg.AIOps.Timeout == 0 || cfg.Prometheus.Timeout == 0 {
		t.Fatalf("timeouts should be parsed: embedding=%s milvus=%s aiops=%s prometheus=%s", cfg.Embedding.DashScope.Timeout, cfg.Milvus.Timeout, cfg.AIOps.Timeout, cfg.Prometheus.Timeout)
	}
	if cfg.AIOps.Mode != "rule" || !cfg.AIOps.FallbackToRule || cfg.AIOps.Agent.Timeout == 0 || cfg.LLM.Provider != "mock" || cfg.LLM.Timeout == 0 {
		t.Fatalf("stage 5 defaults should be parsed: aiops=%+v llm=%+v", cfg.AIOps, cfg.LLM)
	}
}
