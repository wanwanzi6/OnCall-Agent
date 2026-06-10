package embedder

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/embedding/dashscope"
	einoembedding "github.com/cloudwego/eino/components/embedding"

	"oncall-agent/internal/infra/config"
)

type DashScopeEmbedder struct {
	embedder einoembedding.Embedder
	dim      int
	timeout  time.Duration
}

func NewDashScopeEmbedder(ctx context.Context, cfg config.DashScopeEmbeddingConfig) (*DashScopeEmbedder, error) {
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" || strings.HasPrefix(apiKey, "${") {
		return nil, fmt.Errorf("dashscope api key is required")
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return nil, fmt.Errorf("dashscope embedding model is required")
	}
	if cfg.Dimensions <= 0 {
		return nil, fmt.Errorf("dashscope embedding dimensions must be positive")
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	initCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	dimensions := cfg.Dimensions
	embedder, err := dashscope.NewEmbedder(initCtx, &dashscope.EmbeddingConfig{
		APIKey:     apiKey,
		Model:      cfg.Model,
		Dimensions: &dimensions,
		Timeout:    timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize dashscope embedder: %w", err)
	}
	return &DashScopeEmbedder{embedder: embedder, dim: cfg.Dimensions, timeout: timeout}, nil
}

func (e *DashScopeEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}
	for _, text := range texts {
		if strings.TrimSpace(text) == "" {
			return nil, fmt.Errorf("embedding text is empty")
		}
	}
	return e.embed(ctx, texts)
}

func (e *DashScopeEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("embedding query is empty")
	}
	vectors, err := e.embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vectors) != 1 {
		return nil, fmt.Errorf("dashscope returned %d vectors for one query", len(vectors))
	}
	return vectors[0], nil
}

func (e *DashScopeEmbedder) embed(ctx context.Context, texts []string) ([][]float32, error) {
	callCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	raw, err := e.embedder.EmbedStrings(callCtx, texts)
	if err != nil {
		return nil, fmt.Errorf("dashscope embedding request failed: %w", err)
	}
	if len(raw) != len(texts) {
		return nil, fmt.Errorf("dashscope returned %d vectors for %d texts", len(raw), len(texts))
	}

	vectors := make([][]float32, 0, len(raw))
	for _, vector := range raw {
		if len(vector) != e.dim {
			return nil, fmt.Errorf("embedding dimension mismatch: got %d want %d", len(vector), e.dim)
		}
		converted := make([]float32, 0, len(vector))
		for _, value := range vector {
			converted = append(converted, float32(value))
		}
		vectors = append(vectors, converted)
	}
	return vectors, nil
}
