package service

import (
	"strings"

	"oncall-agent/internal/model/domain"
)

type ChatService struct {
	mockEnabled bool
}

func NewChatService(mockEnabled bool) *ChatService {
	return &ChatService{mockEnabled: mockEnabled}
}

func (s *ChatService) Chat(message string) domain.ChatResult {
	answer := "这是 Mock 聊天回复。当前阶段暂不接 LLM，会根据输入返回固定结构，后续可在 service 层替换为 RAG + LLM。"
	if strings.Contains(message, "服务下线") {
		answer = "服务下线告警建议先确认服务、实例和地域，再查询健康检查、发布变更和依赖连接日志。当前回复来自 Mock SOP。"
	}
	return domain.ChatResult{
		Answer:  answer,
		Sources: []string{"mock://sop/service-offline"},
		Mock:    s.mockEnabled,
	}
}

func (s *ChatService) StreamChat(message string) []domain.StreamChunk {
	answer := s.Chat(message).Answer
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
	return chunks
}
