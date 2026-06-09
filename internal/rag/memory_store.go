package rag

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"oncall-agent/internal/model/domain"
)

const DefaultTopK = 3

type MemoryVectorStore struct {
	mu      sync.RWMutex
	records map[string]memoryRecord
	topK    int
}

type memoryRecord struct {
	chunk  domain.Chunk
	vector []float32
}

func NewMemoryVectorStore(defaultTopK int) *MemoryVectorStore {
	if defaultTopK <= 0 {
		defaultTopK = DefaultTopK
	}
	return &MemoryVectorStore{
		records: make(map[string]memoryRecord),
		topK:    defaultTopK,
	}
}

func (s *MemoryVectorStore) Upsert(ctx context.Context, chunks []domain.Chunk, vectors [][]float32) error {
	if len(chunks) != len(vectors) {
		return fmt.Errorf("chunks and vectors length mismatch")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, chunk := range chunks {
		vector := append([]float32(nil), vectors[i]...)
		s.records[chunk.ID] = memoryRecord{chunk: chunk, vector: vector}
	}
	return nil
}

func (s *MemoryVectorStore) Search(ctx context.Context, vector []float32, topK int) ([]domain.SearchResult, error) {
	if topK <= 0 {
		topK = s.topK
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make([]domain.SearchResult, 0, len(s.records))
	for _, record := range s.records {
		score := cosine(vector, record.vector)
		results = append(results, domain.SearchResult{
			Chunk:     record.chunk,
			Score:     score,
			Source:    record.chunk.Metadata["source_file"],
			TitlePath: record.chunk.Metadata["title_path"],
		})
	}
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	if len(results) > topK {
		results = results[:topK]
	}
	return results, nil
}

func (s *MemoryVectorStore) DeleteByDocumentID(ctx context.Context, documentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, record := range s.records {
		if record.chunk.DocumentID == documentID {
			delete(s.records, id)
		}
	}
	return nil
}

func cosine(a, b []float32) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}
	var dot float64
	for i := range a {
		dot += float64(a[i] * b[i])
	}
	return dot
}
