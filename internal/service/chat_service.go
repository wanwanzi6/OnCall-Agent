package service

import (
	"context"
	"log/slog"
	"strings"

	"oncall-agent/internal/infra/trace"
	"oncall-agent/internal/model/domain"
)

type ChatService struct {
	mockEnabled      bool
	log              *slog.Logger
	knowledgeService *KnowledgeService
}

func NewChatService(mockEnabled bool, log *slog.Logger, knowledgeService *KnowledgeService) *ChatService {
	if log == nil {
		log = slog.Default()
	}
	return &ChatService{mockEnabled: mockEnabled, log: log, knowledgeService: knowledgeService}
}

func (s *ChatService) Chat(ctx context.Context, message string) (domain.ChatResult, error) {
	s.log.InfoContext(ctx, "chat requested", "trace_id", trace.FromContext(ctx), "service_name", "chat")
	if s.knowledgeService == nil {
		return domain.ChatResult{
			Answer:  "知识库中没有检索到相关内容，请先上传对应 SOP 文档。",
			Sources: []string{},
			Mock:    s.mockEnabled,
		}, nil
	}

	results, err := s.knowledgeService.Search(ctx, message, 3)
	if err != nil {
		return domain.ChatResult{}, err
	}
	if len(results) == 0 {
		return domain.ChatResult{
			Answer:    "知识库中没有检索到相关内容，请先上传对应 SOP 文档。",
			Sources:   []string{},
			Citations: []domain.Citation{},
			Mock:      s.mockEnabled,
		}, nil
	}

	citations := make([]domain.Citation, 0, len(results))
	sources := make([]string, 0, len(results))
	for _, result := range results {
		citations = append(citations, domain.Citation{
			ChunkID:    result.Chunk.ID,
			DocumentID: result.Chunk.DocumentID,
			Source:     result.Source,
			Score:      result.Score,
			Content:    result.Chunk.Content,
		})
		if result.Source != "" {
			sources = append(sources, result.Source)
		}
	}

	answer := buildMockRAGAnswer(results[0])
	return domain.ChatResult{
		Answer:    answer,
		Sources:   sources,
		Citations: citations,
		Mock:      s.mockEnabled,
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
