package embedder

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"oncall-agent/internal/infra/config"
)

func TestDashScopeEmbedderIntegration(t *testing.T) {
	if os.Getenv("RUN_EMBEDDING_INTEGRATION_TEST") != "1" {
		t.Skip("set RUN_EMBEDDING_INTEGRATION_TEST=1 to run DashScope embedding integration test")
	}
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		t.Fatal("DASHSCOPE_API_KEY is required when RUN_EMBEDDING_INTEGRATION_TEST=1")
	}
	dimensions := 1024
	if raw := os.Getenv("DASHSCOPE_EMBEDDING_DIM"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			t.Fatalf("parse DASHSCOPE_EMBEDDING_DIM: %v", err)
		}
		dimensions = parsed
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	embedder, err := NewDashScopeEmbedder(ctx, config.DashScopeEmbeddingConfig{
		APIKey:     apiKey,
		Model:      envOrDefault("DASHSCOPE_EMBEDDING_MODEL", "text-embedding-v4"),
		Dimensions: dimensions,
		Timeout:    30 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewDashScopeEmbedder returned error: %v", err)
	}
	vector, err := embedder.EmbedQuery(ctx, "服务下线 panic restart_count")
	if err != nil {
		t.Fatalf("EmbedQuery returned error: %v", err)
	}
	if len(vector) != dimensions {
		t.Fatalf("vector dimension = %d, want %d", len(vector), dimensions)
	}
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
