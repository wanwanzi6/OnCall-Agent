package indexer

import (
	"context"
	"os"
	"testing"
	"time"

	"oncall-agent/internal/infra/config"
	"oncall-agent/internal/model/domain"
)

func TestMilvusVectorStoreIntegration(t *testing.T) {
	if os.Getenv("RUN_MILVUS_INTEGRATION_TEST") != "1" {
		t.Skip("set RUN_MILVUS_INTEGRATION_TEST=1 to run Milvus integration test")
	}

	collection := os.Getenv("MILVUS_COLLECTION")
	if collection == "" {
		collection = "oncall_knowledge_test"
	}
	store, err := NewMilvusVectorStore(context.Background(), config.MilvusConfig{
		Address:     envOrDefault("MILVUS_ADDRESS", "localhost:19530"),
		Database:    envOrDefault("MILVUS_DATABASE", "agent"),
		Collection:  collection,
		VectorField: envOrDefault("MILVUS_VECTOR_FIELD", "vector"),
		Timeout:     10 * time.Second,
	}, 3, 2)
	if err != nil {
		t.Fatalf("NewMilvusVectorStore returned error: %v", err)
	}

	chunk := domain.Chunk{
		ID:         "chk_integration_1",
		DocumentID: "doc_integration_1",
		Content:    "服务下线 panic 日志",
		Index:      0,
		Metadata: map[string]string{
			"source_file": "integration.md",
			"title_path":  "服务下线",
		},
	}
	if err := store.Upsert(context.Background(), []domain.Chunk{chunk}, [][]float32{{1, 0}}); err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}
	results, err := store.Search(context.Background(), []float32{1, 0}, 1)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected search result")
	}
	if err := store.DeleteByDocumentID(context.Background(), "doc_integration_1"); err != nil {
		t.Fatalf("DeleteByDocumentID returned error: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
