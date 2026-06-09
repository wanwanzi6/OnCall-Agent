package service

import (
	"path/filepath"
	"strings"

	"oncall-agent/internal/model/domain"
)

type KnowledgeService struct {
	mockEnabled bool
}

func NewKnowledgeService(mockEnabled bool) *KnowledgeService {
	return &KnowledgeService{mockEnabled: mockEnabled}
}

func (s *KnowledgeService) Upload(fileName string, size int64) domain.UploadResult {
	if fileName == "" {
		fileName = "mock-sop.md"
	}
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(fileName)), ".")
	if ext == "" {
		ext = "md"
	}

	chunkCount := 3
	if size > 0 {
		chunkCount = int(size/1024) + 1
	}

	return domain.UploadResult{
		FileName:   fileName,
		FileType:   ext,
		ChunkCount: chunkCount,
		DocID:      "mock-doc-service-offline",
		NextSteps:  []string{"loader", "splitter", "embedder", "indexer"},
		Mock:       s.mockEnabled,
	}
}
