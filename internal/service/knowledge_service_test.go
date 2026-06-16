package service

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"oncall-agent/internal/infra/config"
)

func TestKnowledgeServiceIndexSearchAndDelete(t *testing.T) {
	svc := newTestKnowledgeService(t)
	path := writeServiceTestFile(t, "runbook.md", "# 服务下线\n## 日志排查\n服务下线后先查询最近 1 小时 panic 日志。")

	index, err := svc.IndexFile(context.Background(), path)
	if err != nil {
		t.Fatalf("IndexFile returned error: %v", err)
	}
	if index.DocumentID == "" || index.ChunkCount == 0 {
		t.Fatalf("index result = %+v", index)
	}

	results, err := svc.Search(context.Background(), "服务下线 panic 日志", 3)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected search results")
	}
	if results[0].Chunk.DocumentID != index.DocumentID {
		t.Fatalf("document id = %q, want %q", results[0].Chunk.DocumentID, index.DocumentID)
	}
	traced, err := svc.SearchWithTrace(context.Background(), "服务下线 panic 日志", 3)
	if err != nil {
		t.Fatalf("SearchWithTrace returned error: %v", err)
	}
	if traced.Plan == nil || len(traced.Iterations) == 0 || len(traced.Steps) == 0 || len(traced.Results) == 0 {
		t.Fatalf("expected rag agent trace: %+v", traced)
	}

	if err := svc.DeleteDocument(context.Background(), index.DocumentID); err != nil {
		t.Fatalf("DeleteDocument returned error: %v", err)
	}
	results, err = svc.Search(context.Background(), "服务下线 panic 日志", 3)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	for _, result := range results {
		if result.Chunk.DocumentID == index.DocumentID {
			t.Fatal("deleted document returned")
		}
	}
}

func newTestKnowledgeService(t *testing.T) *KnowledgeService {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewKnowledgeService(true, config.KnowledgeConfig{
		UploadDir:        t.TempDir(),
		MaxFileSizeBytes: 1024 * 1024,
		AllowedExts:      []string{".md", ".markdown", ".txt"},
	}, config.RAGConfig{
		ChunkSize:    80,
		ChunkOverlap: 10,
		EmbeddingDim: 32,
		DefaultTopK:  3,
	}, log)
}

func writeServiceTestFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	return path
}
