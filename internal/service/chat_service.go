package service

import (
	"context"
	"log/slog"
	"strings"

	"oncall-agent/internal/infra/config"
	"oncall-agent/internal/infra/trace"
	"oncall-agent/internal/llm"
	"oncall-agent/internal/model/domain"
)

type ChatService struct {
	mockEnabled      bool
	log              *slog.Logger
	knowledgeService *KnowledgeService
	agent            *ChatAgent
}

func NewChatService(mockEnabled bool, log *slog.Logger, knowledgeService *KnowledgeService, llmCfg ...config.LLMConfig) *ChatService {
	if log == nil {
		log = slog.Default()
	}
	cfg := config.LLMConfig{Provider: llm.ProviderMock}
	if len(llmCfg) > 0 {
		cfg = llmCfg[0]
	}
	model, err := llm.NewEinoToolCallingModel(cfg)
	if err != nil {
		log.Error("initialize chat agent model failed, fallback to mock", "error", err.Error())
		model, _ = llm.NewEinoToolCallingModel(config.LLMConfig{Provider: llm.ProviderMock})
	}
	return &ChatService{mockEnabled: mockEnabled, log: log, knowledgeService: knowledgeService, agent: NewChatAgent(model, knowledgeService, log)}
}

func (s *ChatService) Chat(ctx context.Context, message string) (domain.ChatResult, error) {
	s.log.InfoContext(ctx, "chat requested", "trace_id", trace.FromContext(ctx), "service_name", "chat")
	if s.agent == nil {
		return domain.ChatResult{}, nil
	}
	return s.agent.Chat(ctx, message, s.mockEnabled)
}

func (s *ChatService) StreamChat(ctx context.Context, message string) ([]domain.StreamChunk, error) {
	result, err := s.Chat(ctx, message)
	if err != nil {
		return nil, err
	}
	answer := result.Answer
	parts := strings.Fields(answer)
	if len(parts) == 0 {
		parts = []string{answer}
	}

	chunks := make([]domain.StreamChunk, 0, len(parts)+1)
	for i, part := range parts {
		chunks = append(chunks, domain.StreamChunk{
			Index: i,
			Delta: part,
			Done:  false,
		})
	}
	chunks = append(chunks, domain.StreamChunk{Index: len(parts), Done: true})
	return chunks, nil
}

func buildMockRAGAnswer(result domain.SearchResult) string {
	content := strings.TrimSpace(result.Chunk.Content)
	if content == "" {
		return "根据知识库内容，当前问题可参考已上传 SOP 中的相关片段处理。"
	}
	content = strings.ReplaceAll(content, "\n", " ")
	if len(content) > 120 {
		content = content[:120]
	}
	return "根据知识库内容，" + content
}
