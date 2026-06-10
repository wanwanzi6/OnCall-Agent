package loader

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"oncall-agent/internal/model/domain"
	"oncall-agent/internal/rag"
)

type FileLoader struct{}

func NewFileLoader() *FileLoader {
	return &FileLoader{}
}

func (l *FileLoader) Load(ctx context.Context, filePath string) (domain.Document, error) {
	if strings.TrimSpace(filePath) == "" {
		return domain.Document{}, rag.ErrMissingDocumentPath
	}
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext != ".md" && ext != ".markdown" && ext != ".txt" {
		return domain.Document{}, fmt.Errorf("%w: %s", rag.ErrUnsupportedDocument, ext)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return domain.Document{}, fmt.Errorf("read document: %w", err)
	}
	content := strings.TrimSpace(string(data))
	if content == "" {
		return domain.Document{}, rag.ErrEmptyDocument
	}

	name := filepath.Base(filePath)
	return domain.Document{
		ID:      documentID(filePath, content),
		Name:    name,
		Path:    filePath,
		Content: content,
		Metadata: map[string]string{
			"source_file": name,
			"file_ext":    ext,
		},
		CreatedAt: time.Now().UTC(),
	}, nil
}

func documentID(filePath, content string) string {
	sum := sha1.Sum([]byte(filePath + "\x00" + content))
	return "doc_" + hex.EncodeToString(sum[:])[:16]
}
