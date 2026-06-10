package indexer

import (
	"context"
	"sync"
	"testing"

	"oncall-agent/internal/model/domain"
)

func TestMemoryVectorStoreSearchAndTopK(t *testing.T) {
	store := NewMemoryVectorStore(3)
	chunks := []domain.Chunk{
		testChunk("c1", "doc1", 0),
		testChunk("c2", "doc1", 1),
	}
	vectors := [][]float32{
		{1, 0},
		{0, 1},
	}

	if err := store.Upsert(context.Background(), chunks, vectors); err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}
	results, err := store.Search(context.Background(), []float32{1, 0}, 1)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("result count = %d, want 1", len(results))
	}
	if results[0].Chunk.ID != "c1" {
		t.Fatalf("top chunk = %q", results[0].Chunk.ID)
	}
}

func TestMemoryVectorStoreDeleteByDocumentID(t *testing.T) {
	store := NewMemoryVectorStore(3)
	if err := store.Upsert(context.Background(), []domain.Chunk{
		testChunk("c1", "doc1", 0),
		testChunk("c2", "doc2", 0),
	}, [][]float32{{1, 0}, {0, 1}}); err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}

	if err := store.DeleteByDocumentID(context.Background(), "doc1"); err != nil {
		t.Fatalf("DeleteByDocumentID returned error: %v", err)
	}
	results, err := store.Search(context.Background(), []float32{1, 0}, 10)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	for _, result := range results {
		if result.Chunk.DocumentID == "doc1" {
			t.Fatal("deleted document returned")
		}
	}
}

func TestMemoryVectorStoreConcurrentAccess(t *testing.T) {
	store := NewMemoryVectorStore(3)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			chunk := testChunk("c"+string(rune('a'+i)), "doc", i)
			_ = store.Upsert(context.Background(), []domain.Chunk{chunk}, [][]float32{{1, 0}})
			_, _ = store.Search(context.Background(), []float32{1, 0}, 3)
		}(i)
	}
	wg.Wait()
}

func testChunk(id, docID string, index int) domain.Chunk {
	return domain.Chunk{
		ID:         id,
		DocumentID: docID,
		Content:    "content",
		Index:      index,
		Metadata: map[string]string{
			"source_file": "runbook.md",
			"title_path":  "服务下线",
		},
	}
}
