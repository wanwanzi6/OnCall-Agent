package rag

import (
	"context"
	"math"
	"testing"
)

func TestMockEmbedderIsStable(t *testing.T) {
	embedder := NewMockEmbedder(64)

	a, err := embedder.EmbedQuery(context.Background(), "服务下线怎么处理")
	if err != nil {
		t.Fatalf("EmbedQuery returned error: %v", err)
	}
	b, err := embedder.EmbedQuery(context.Background(), "服务下线怎么处理")
	if err != nil {
		t.Fatalf("EmbedQuery returned error: %v", err)
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("vector differs at %d", i)
		}
	}
}

func TestMockEmbedderDimensionAndNorm(t *testing.T) {
	embedder := NewMockEmbedder(32)

	vector, err := embedder.EmbedQuery(context.Background(), "服务下线 panic 日志")
	if err != nil {
		t.Fatalf("EmbedQuery returned error: %v", err)
	}
	if len(vector) != 32 {
		t.Fatalf("dimension = %d, want 32", len(vector))
	}
	var sum float64
	for _, v := range vector {
		sum += float64(v * v)
	}
	if math.Abs(math.Sqrt(sum)-1) > 0.0001 {
		t.Fatalf("norm = %f, want 1", math.Sqrt(sum))
	}
}
