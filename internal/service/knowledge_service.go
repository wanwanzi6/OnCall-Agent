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
	"sync"

	"oncall-agent/internal/infra/config"
	"oncall-agent/internal/infra/trace"
	"oncall-agent/internal/infra/upload"
	"oncall-agent/internal/model/domain"
	"oncall-agent/internal/rag"
	"oncall-agent/internal/rag/embedder"
	"oncall-agent/internal/rag/indexer"
	"oncall-agent/internal/rag/loader"
	"oncall-agent/internal/rag/splitter"
)

type KnowledgeService struct {
	mockEnabled         bool
	policy              upload.Policy
	log                 *slog.Logger
	loader              rag.Loader
	splitter            rag.Splitter
	embedder            rag.Embedder
	vectorStore         rag.VectorStore
	defaultTopK         int
	embedderProvider    string
	vectorStoreProvider string
	mu                  sync.RWMutex
	documents           map[string]domain.Document
}

func NewKnowledgeService(mockEnabled bool, cfg config.KnowledgeConfig, ragCfg config.RAGConfig, log *slog.Logger) *KnowledgeService {
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
		log:                 log,
		loader:              loader.NewFileLoader(),
		splitter:            splitter.NewTextSplitter(ragCfg.ChunkSize, ragCfg.ChunkOverlap),
		embedder:            embedder.NewMockEmbedder(ragCfg.EmbeddingDim),
		vectorStore:         indexer.NewMemoryVectorStore(ragCfg.DefaultTopK),
		defaultTopK:         ragCfg.DefaultTopK,
		embedderProvider:    rag.EmbedderProviderMock,
		vectorStoreProvider: rag.VectorStoreProviderMemory,
		documents:           make(map[string]domain.Document),
	}
}

func NewKnowledgeServiceFromConfig(ctx context.Context, cfg *config.Config, log *slog.Logger) (*KnowledgeService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if log == nil {
		log = slog.Default()
	}
	embedder, err := rag.NewEmbedder(ctx, *cfg)
	if err != nil {
		return nil, err
	}
	vectorStore, err := rag.NewVectorStore(ctx, *cfg)
	if err != nil {
		return nil, err
	}
	service := NewKnowledgeService(cfg.Mock.Enabled, cfg.Knowledge, cfg.RAG, log)
	service.embedder = embedder
	service.vectorStore = vectorStore
	service.embedderProvider = cfg.RAG.EmbedderProvider
	service.vectorStoreProvider = cfg.RAG.VectorStoreProvider
	log.InfoContext(ctx, "knowledge service initialized",
		"trace_id", trace.FromContext(ctx),
		"service_name", "knowledge",
		"embedder_provider", service.embedderProvider,
		"vector_store_provider", service.vectorStoreProvider,
	)
	return service, nil
}

func (s *KnowledgeService) ProviderStatus() map[string]string {
	return map[string]string{
		"embedder_provider":     s.embedderProvider,
		"vector_store_provider": s.vectorStoreProvider,
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

	indexResult, err := s.IndexFile(ctx, target)
	if err != nil {
		return domain.UploadResult{}, err
	}

	s.log.InfoContext(ctx, "upload saved and indexed",
		"trace_id", trace.FromContext(ctx),
		"service_name", "knowledge",
		"file_name", sanitized,
		"size", written,
		"document_id", indexResult.DocumentID,
		"chunk_count", indexResult.ChunkCount,
	)
	result := s.uploadResult(sanitized, written)
	result.DocID = indexResult.DocumentID
	result.ChunkCount = indexResult.ChunkCount
	return result, nil
}

func (s *KnowledgeService) IndexFile(ctx context.Context, filePath string) (domain.IndexResult, error) {
	doc, err := s.loader.Load(ctx, filePath)
	if err != nil {
		s.log.ErrorContext(ctx, "document load failed",
			"trace_id", trace.FromContext(ctx),
			"service_name", "knowledge",
			"file_path", filePath,
			"error", err.Error(),
		)
		return domain.IndexResult{}, err
	}
	chunks, err := s.splitter.Split(ctx, doc)
	if err != nil {
		return domain.IndexResult{}, err
	}
	texts := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		texts = append(texts, chunk.Content)
	}
	vectors, err := s.embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		return domain.IndexResult{}, err
	}
	if err := s.vectorStore.Upsert(ctx, chunks, vectors); err != nil {
		return domain.IndexResult{}, err
	}

	s.mu.Lock()
	doc.Content = ""
	s.documents[doc.ID] = doc
	s.mu.Unlock()

	result := domain.IndexResult{DocumentID: doc.ID, ChunkCount: len(chunks)}
	s.log.InfoContext(ctx, "document indexed",
		"trace_id", trace.FromContext(ctx),
		"service_name", "knowledge",
		"document_id", result.DocumentID,
		"chunk_count", result.ChunkCount,
	)
	return result, nil
}

func (s *KnowledgeService) Search(ctx context.Context, query string, topK int) ([]domain.SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("query is required")
	}
	if topK <= 0 {
		topK = s.defaultTopK
	}
	vector, err := s.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	results, err := s.vectorStore.Search(ctx, vector, topK)
	if err != nil {
		return nil, err
	}
	filtered := make([]domain.SearchResult, 0, len(results))
	for _, result := range results {
		if result.Score > 0 {
			filtered = append(filtered, result)
		}
	}

	s.log.InfoContext(ctx, "knowledge searched",
		"trace_id", trace.FromContext(ctx),
		"service_name", "knowledge",
		"query", query,
		"top_k", topK,
		"result_count", len(filtered),
	)
	return filtered, nil
}

func (s *KnowledgeService) ListDocuments(ctx context.Context) ([]domain.Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	docs := make([]domain.Document, 0, len(s.documents))
	for _, doc := range s.documents {
		docs = append(docs, doc)
	}
	return docs, nil
}

func (s *KnowledgeService) DeleteDocument(ctx context.Context, documentID string) error {
	if strings.TrimSpace(documentID) == "" {
		return fmt.Errorf("document_id is required")
	}
	if err := s.vectorStore.DeleteByDocumentID(ctx, documentID); err != nil {
		return err
	}
	s.mu.Lock()
	delete(s.documents, documentID)
	s.mu.Unlock()
	s.log.InfoContext(ctx, "document deleted",
		"trace_id", trace.FromContext(ctx),
		"service_name", "knowledge",
		"document_id", documentID,
	)
	return nil
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
		DocID:      "",
		NextSteps:  []string{"loader", "splitter", "embedder", "indexer"},
		Mock:       s.mockEnabled,
	}
}
