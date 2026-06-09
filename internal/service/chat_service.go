package service

import (
	"context"
	"log/slog"
	"strings"

	"oncall-agent/internal/infra/trace"
	"oncall-agent/internal/model/domain"
)

type ChatService struct {
	mockEnabled bool
	log         *slog.Logger
}

func NewChatService(mockEnabled bool, log *slog.Logger) *ChatService {
	if log == nil {
		log = slog.Default()
	}
	return &ChatService{mockEnabled: mockEnabled, log: log}
}

func (s *ChatService) Chat(ctx context.Context, message string) (domain.ChatResult, error) {
	s.log.InfoContext(ctx, "chat requested", "trace_id", trace.FromContext(ctx), "service_name", "chat")
	answer := "这是 Mock 聊天回复。当前阶段暂不接 LLM，会根据输入返回固定结构，后续可在 service 层替换为 RAG + LLM。"
	if strings.Contains(message, "服务下线") {
		answer = "服务下线告警建议先确认服务、实例和地域，再查询健康检查、发布变更和依赖连接日志。当前回复来自 Mock SOP。"
	}
	return domain.ChatResult{
		Answer:  answer,
		Sources: []string{"mock://sop/service-offline"},
		Mock:    s.mockEnabled,
	}, nil
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
