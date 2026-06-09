package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"oncall-agent/internal/infra/config"
	"oncall-agent/internal/infra/trace"
	"oncall-agent/internal/infra/upload"
	"oncall-agent/internal/model/domain"
)

type KnowledgeService struct {
	mockEnabled bool
	policy      upload.Policy
	log         *slog.Logger
}

func NewKnowledgeService(mockEnabled bool, cfg config.KnowledgeConfig, log *slog.Logger) *KnowledgeService {
	if log == nil {
		log = slog.Default()
	}
	return &KnowledgeService{
		mockEnabled: mockEnabled,
		policy: upload.Policy{
			UploadDir:        cfg.UploadDir,
			MaxFileSizeBytes: cfg.MaxFileSizeBytes,
			AllowedExts:      cfg.AllowedExts,
		},
		log: log,
	}
}

func (s *KnowledgeService) UploadMetadata(ctx context.Context, fileName string, size int64) (domain.UploadResult, error) {
	sanitized, _, err := upload.Validate(s.policy, fileName, size)
	if err != nil {
		s.log.ErrorContext(ctx, "upload metadata validation failed",
			"trace_id", trace.FromContext(ctx),
			"service_name", "knowledge",
			"error", err.Error(),
		)
		return domain.UploadResult{}, err
	}
	return s.uploadResult(sanitized, size), nil
}

func (s *KnowledgeService) SaveUpload(ctx context.Context, header *multipart.FileHeader) (domain.UploadResult, error) {
	if header == nil {
		return domain.UploadResult{}, fmt.Errorf("file is required")
	}

	sanitized, target, err := upload.Validate(s.policy, header.Filename, header.Size)
	if err != nil {
		s.log.ErrorContext(ctx, "upload validation failed",
			"trace_id", trace.FromContext(ctx),
			"service_name", "knowledge",
			"error", err.Error(),
		)
		return domain.UploadResult{}, err
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return domain.UploadResult{}, fmt.Errorf("create upload directory: %w", err)
	}

	src, err := header.Open()
	if err != nil {
		return domain.UploadResult{}, fmt.Errorf("open uploaded file: %w", err)
	}
	defer src.Close()

	dst, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return domain.UploadResult{}, fmt.Errorf("create upload file: %w", err)
	}
	defer dst.Close()

	written, err := io.Copy(dst, io.LimitReader(src, s.policy.MaxFileSizeBytes+1))
	if err != nil {
		return domain.UploadResult{}, fmt.Errorf("save upload file: %w", err)
	}
	if err := upload.ValidateSize(written, s.policy.MaxFileSizeBytes); err != nil {
		return domain.UploadResult{}, err
	}

	s.log.InfoContext(ctx, "upload saved",
		"trace_id", trace.FromContext(ctx),
		"service_name", "knowledge",
		"file_name", sanitized,
		"size", written,
	)
	return s.uploadResult(sanitized, written), nil
}

func (s *KnowledgeService) uploadResult(fileName string, size int64) domain.UploadResult {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(fileName)), ".")
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
