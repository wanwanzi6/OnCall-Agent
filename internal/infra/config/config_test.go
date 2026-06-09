package config

import "testing"

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
}
