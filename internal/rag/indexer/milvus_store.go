package indexer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"oncall-agent/internal/infra/config"
	"oncall-agent/internal/model/domain"
)

type MilvusVectorStore struct {
	baseURL     string
	database    string
	collection  string
	vectorField string
	dim         int
	topK        int
	timeout     time.Duration
	client      *http.Client
}

func NewMilvusVectorStore(ctx context.Context, cfg config.MilvusConfig, defaultTopK, vectorDim int) (*MilvusVectorStore, error) {
	if strings.TrimSpace(cfg.Address) == "" {
		return nil, fmt.Errorf("milvus address is required")
	}
	if strings.TrimSpace(cfg.Collection) == "" {
		return nil, fmt.Errorf("milvus collection is required")
	}
	if strings.TrimSpace(cfg.VectorField) == "" {
		return nil, fmt.Errorf("milvus vector field is required")
	}
	if vectorDim <= 0 {
		return nil, fmt.Errorf("milvus vector dimension must be positive")
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	if defaultTopK <= 0 {
		defaultTopK = DefaultTopK
	}

	store := &MilvusVectorStore{
		baseURL:     normalizeMilvusAddress(cfg.Address),
		database:    strings.TrimSpace(cfg.Database),
		collection:  strings.TrimSpace(cfg.Collection),
		vectorField: strings.TrimSpace(cfg.VectorField),
		dim:         vectorDim,
		topK:        defaultTopK,
		timeout:     timeout,
		client:      &http.Client{Timeout: timeout},
	}
	if err := store.ensureCollection(ctx); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *MilvusVectorStore) Upsert(ctx context.Context, chunks []domain.Chunk, vectors [][]float32) error {
	if len(chunks) != len(vectors) {
		return fmt.Errorf("chunks and vectors length mismatch")
	}
	rows := make([]map[string]any, 0, len(chunks))
	for i, chunk := range chunks {
		if len(vectors[i]) != s.dim {
			return fmt.Errorf("embedding dimension mismatch: got %d want %d", len(vectors[i]), s.dim)
		}
		rows = append(rows, map[string]any{
			"id":          chunk.ID,
			"document_id": chunk.DocumentID,
			"content":     chunk.Content,
			"source":      chunk.Metadata["source_file"],
			"title_path":  chunk.Metadata["title_path"],
			"chunk_index": int64(chunk.Index),
			"metadata":    chunk.Metadata,
			s.vectorField: vectors[i],
		})
	}
	if len(rows) == 0 {
		return nil
	}
	_, err := s.post(ctx, "/v2/vectordb/entities/upsert", map[string]any{
		"collectionName": s.collection,
		"data":           rows,
	})
	if err != nil {
		return fmt.Errorf("milvus upsert failed: %w", err)
	}
	return nil
}

func (s *MilvusVectorStore) Search(ctx context.Context, vector []float32, topK int) ([]domain.SearchResult, error) {
	if len(vector) != s.dim {
		return nil, fmt.Errorf("embedding dimension mismatch: got %d want %d", len(vector), s.dim)
	}
	if topK <= 0 {
		topK = s.topK
	}
	body, err := s.post(ctx, "/v2/vectordb/entities/search", map[string]any{
		"collectionName": s.collection,
		"data":           [][]float32{vector},
		"annsField":      s.vectorField,
		"limit":          topK,
		"outputFields":   []string{"id", "document_id", "content", "source", "title_path", "chunk_index", "metadata"},
	})
	if err != nil {
		return nil, fmt.Errorf("milvus search failed: %w", err)
	}

	var resp struct {
		Data [][]milvusSearchRow `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse milvus search response: %w", err)
	}
	if len(resp.Data) == 0 {
		return []domain.SearchResult{}, nil
	}

	results := make([]domain.SearchResult, 0, len(resp.Data[0]))
	for _, row := range resp.Data[0] {
		metadata := row.Metadata
		if metadata == nil {
			metadata = map[string]string{}
		}
		metadata["source_file"] = row.Source
		metadata["title_path"] = row.TitlePath
		results = append(results, domain.SearchResult{
			Chunk: domain.Chunk{
				ID:         row.ID,
				DocumentID: row.DocumentID,
				Content:    row.Content,
				Metadata:   metadata,
				Index:      int(row.ChunkIndex),
			},
			Score:     row.Distance,
			Source:    row.Source,
			TitlePath: row.TitlePath,
		})
	}
	return results, nil
}

func (s *MilvusVectorStore) DeleteByDocumentID(ctx context.Context, documentID string) error {
	if strings.TrimSpace(documentID) == "" {
		return fmt.Errorf("document_id is required")
	}
	filter := fmt.Sprintf("document_id == %q", documentID)
	_, err := s.post(ctx, "/v2/vectordb/entities/delete", map[string]any{
		"collectionName": s.collection,
		"filter":         filter,
	})
	if err != nil {
		return fmt.Errorf("milvus delete failed: %w", err)
	}
	return nil
}

type milvusSearchRow struct {
	ID         string            `json:"id"`
	DocumentID string            `json:"document_id"`
	Content    string            `json:"content"`
	Source     string            `json:"source"`
	TitlePath  string            `json:"title_path"`
	ChunkIndex int64             `json:"chunk_index"`
	Metadata   map[string]string `json:"metadata"`
	Distance   float64           `json:"distance"`
}

func (s *MilvusVectorStore) ensureCollection(ctx context.Context) error {
	body, err := s.post(ctx, "/v2/vectordb/collections/has", map[string]any{
		"collectionName": s.collection,
	})
	if err != nil {
		return fmt.Errorf("check milvus collection failed: %w", err)
	}
	var hasResp struct {
		Data bool `json:"data"`
	}
	if err := json.Unmarshal(body, &hasResp); err != nil {
		return fmt.Errorf("parse milvus collection check response: %w", err)
	}
	if !hasResp.Data {
		if _, err := s.post(ctx, "/v2/vectordb/collections/create", s.createCollectionRequest()); err != nil {
			return fmt.Errorf("create milvus collection failed: %w", err)
		}
	}
	if _, err := s.post(ctx, "/v2/vectordb/collections/load", map[string]any{
		"collectionName": s.collection,
	}); err != nil {
		return fmt.Errorf("load milvus collection failed: %w", err)
	}
	return nil
}

func (s *MilvusVectorStore) createCollectionRequest() map[string]any {
	return map[string]any{
		"collectionName": s.collection,
		"schema": map[string]any{
			"autoID": false,
			"fields": []map[string]any{
				{"fieldName": "id", "dataType": "VarChar", "isPrimary": true, "elementTypeParams": map[string]any{"max_length": "128"}},
				{"fieldName": "document_id", "dataType": "VarChar", "elementTypeParams": map[string]any{"max_length": "128"}},
				{"fieldName": "content", "dataType": "VarChar", "elementTypeParams": map[string]any{"max_length": "8192"}},
				{"fieldName": "source", "dataType": "VarChar", "elementTypeParams": map[string]any{"max_length": "512"}},
				{"fieldName": "title_path", "dataType": "VarChar", "elementTypeParams": map[string]any{"max_length": "1024"}},
				{"fieldName": "chunk_index", "dataType": "Int64"},
				{"fieldName": "metadata", "dataType": "JSON"},
				{"fieldName": s.vectorField, "dataType": "FloatVector", "elementTypeParams": map[string]any{"dim": fmt.Sprintf("%d", s.dim)}},
			},
		},
		"indexParams": []map[string]any{
			{
				"fieldName":  s.vectorField,
				"indexName":  s.vectorField + "_idx",
				"metricType": "COSINE",
				"indexType":  "AUTOINDEX",
			},
		},
	}
}

func (s *MilvusVectorStore) post(ctx context.Context, path string, payload map[string]any) ([]byte, error) {
	callCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if s.database != "" {
		payload["dbName"] = s.database
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(callCtx, http.MethodPost, s.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http status %d: %s", resp.StatusCode, string(body))
	}
	var base struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &base); err == nil && base.Code != 0 {
		if base.Message == "" {
			base.Message = "unknown milvus error"
		}
		return nil, fmt.Errorf("%s", base.Message)
	}
	return body, nil
}

func normalizeMilvusAddress(address string) string {
	address = strings.TrimSpace(address)
	if strings.HasPrefix(address, "http://") || strings.HasPrefix(address, "https://") {
		return strings.TrimRight(address, "/")
	}
	return "http://" + strings.TrimRight(address, "/")
}
