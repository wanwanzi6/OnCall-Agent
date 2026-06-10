package splitter

import (
	"context"
	"strings"
	"testing"

	"oncall-agent/internal/model/domain"
)

func TestTextSplitterPreservesMarkdownTitlePath(t *testing.T) {
	doc := domain.Document{
		ID:      "doc-1",
		Name:    "runbook.md",
		Content: "# 服务下线\n## 排查\n### 日志\n查询 panic 日志",
		Metadata: map[string]string{
			"source_file": "runbook.md",
			"file_ext":    ".md",
		},
	}

	chunks, err := NewTextSplitter(800, 100).Split(context.Background(), doc)
	if err != nil {
		t.Fatalf("Split returned error: %v", err)
	}
	if len(chunks) == 0 {
		t.Fatal("expected chunks")
	}
	got := chunks[len(chunks)-1].Metadata["title_path"]
	want := "服务下线 > 排查 > 日志"
	if got != want {
		t.Fatalf("title_path = %q, want %q", got, want)
	}
}

func TestTextSplitterSplitsLongText(t *testing.T) {
	doc := domain.Document{
		ID:      "doc-1",
		Name:    "note.txt",
		Content: strings.Repeat("服务下线需要排查日志。", 30),
		Metadata: map[string]string{
			"source_file": "note.txt",
			"file_ext":    ".txt",
		},
	}

	chunks, err := NewTextSplitter(50, 10).Split(context.Background(), doc)
	if err != nil {
		t.Fatalf("Split returned error: %v", err)
	}
	if len(chunks) < 2 {
		t.Fatalf("chunk count = %d, want multiple", len(chunks))
	}
	for i, chunk := range chunks {
		if chunk.Index != i {
			t.Fatalf("chunk index = %d, want %d", chunk.Index, i)
		}
	}
}
