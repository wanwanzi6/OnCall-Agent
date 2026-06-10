package llm

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
)

const (
	ProviderMock             = "mock"
	ProviderOpenAICompatible = "openai-compatible"
)

type ChatModel interface {
	Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

func NewChatModel(cfg config.LLMConfig) (ChatModel, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if provider == "" {
		provider = ProviderMock
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	switch provider {
	case ProviderMock:
		return MockChatModel{}, nil
	case ProviderOpenAICompatible:
		if strings.TrimSpace(cfg.APIKey) == "" {
			return nil, fmt.Errorf("llm api_key is required when provider is %q", ProviderOpenAICompatible)
		}
		if strings.TrimSpace(cfg.Model) == "" {
			return nil, fmt.Errorf("llm model is required when provider is %q", ProviderOpenAICompatible)
		}
		baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		return &OpenAICompatibleChatModel{
			apiKey:  cfg.APIKey,
			baseURL: baseURL,
			model:   cfg.Model,
			client:  &http.Client{Timeout: cfg.Timeout},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported llm provider %q", cfg.Provider)
	}
}

type MockChatModel struct{}

func (MockChatModel) Generate(_ context.Context, _ string, userPrompt string) (string, error) {
	if strings.TrimSpace(userPrompt) == "" {
		return "", fmt.Errorf("user prompt is required")
	}
	return userPrompt, nil
}

type OpenAICompatibleChatModel struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

func (m *OpenAICompatibleChatModel) Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if m == nil || m.client == nil {
		return "", fmt.Errorf("llm client is not configured")
	}
	body := openAIChatRequest{
		Model: m.model,
		Messages: []openAIMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal llm request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create llm request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.apiKey)

	resp, err := m.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("call llm: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return "", fmt.Errorf("read llm response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("llm returned status %d", resp.StatusCode)
	}
	var decoded openAIChatResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		return "", fmt.Errorf("parse llm response: %w", err)
	}
	if len(decoded.Choices) == 0 || strings.TrimSpace(decoded.Choices[0].Message.Content) == "" {
		return "", fmt.Errorf("llm returned empty response")
	}
	return decoded.Choices[0].Message.Content, nil
}

type openAIChatRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
}
