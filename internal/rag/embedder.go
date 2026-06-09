package rag

import (
	"context"
	"hash/fnv"
	"math"
	"strings"
	"unicode"
)

const DefaultEmbeddingDim = 64

type MockEmbedder struct {
	dim int
}

func NewMockEmbedder(dim int) *MockEmbedder {
	if dim <= 0 {
		dim = DefaultEmbeddingDim
	}
	return &MockEmbedder{dim: dim}
}

func (e *MockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	vectors := make([][]float32, 0, len(texts))
	for _, text := range texts {
		vectors = append(vectors, e.embed(text))
	}
	return vectors, nil
}

func (e *MockEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return e.embed(text), nil
}

func (e *MockEmbedder) embed(text string) []float32 {
	vector := make([]float32, e.dim)
	for _, token := range tokens(text) {
		h := hashToken(token)
		index := int(h % uint64(e.dim))
		sign := float32(1)
		if h&(1<<63) != 0 {
			sign = -1
		}
		vector[index] += sign
	}
	normalize(vector)
	return vector
}

func tokens(text string) []string {
	text = strings.ToLower(text)
	var result []string
	var word strings.Builder
	flush := func() {
		if word.Len() > 0 {
			result = append(result, word.String())
			word.Reset()
		}
	}

	for _, r := range text {
		switch {
		case unicode.Is(unicode.Han, r):
			flush()
			result = append(result, string(r))
		case unicode.IsLetter(r), unicode.IsDigit(r):
			word.WriteRune(r)
		default:
			flush()
		}
	}
	flush()
	return result
}

func hashToken(token string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(token))
	return h.Sum64()
}

func normalize(vector []float32) {
	var sum float64
	for _, v := range vector {
		sum += float64(v * v)
	}
	if sum == 0 {
		return
	}
	norm := float32(math.Sqrt(sum))
	for i := range vector {
		vector[i] /= norm
	}
}
