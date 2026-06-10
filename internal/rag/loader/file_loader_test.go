package loader

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileLoaderLoadsMarkdown(t *testing.T) {
	path := writeTestFile(t, "runbook.md", "# 服务下线\n处理步骤")

	doc, err := NewFileLoader().Load(context.Background(), path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if doc.Name != "runbook.md" {
		t.Fatalf("name = %q", doc.Name)
	}
	if doc.Metadata["source_file"] != "runbook.md" || doc.Metadata["file_ext"] != ".md" {
		t.Fatalf("metadata = %#v", doc.Metadata)
	}
}

func TestFileLoaderLoadsTXT(t *testing.T) {
	path := writeTestFile(t, "note.txt", "服务下线处理")

	doc, err := NewFileLoader().Load(context.Background(), path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if doc.Content != "服务下线处理" {
		t.Fatalf("content = %q", doc.Content)
	}
}

func TestFileLoaderRejectsEmptyFile(t *testing.T) {
	path := writeTestFile(t, "empty.md", " \n")

	_, err := NewFileLoader().Load(context.Background(), path)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFileLoaderRejectsUnsupportedFile(t *testing.T) {
	path := writeTestFile(t, "runbook.pdf", "content")

	_, err := NewFileLoader().Load(context.Background(), path)
	if err == nil {
		t.Fatal("expected error")
	}
}

func writeTestFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	return path
}
