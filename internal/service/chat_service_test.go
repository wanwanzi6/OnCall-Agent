package service

import (
	"context"
	"io"
	"log/slog"
	"testing"
)

func TestChatServiceReturnsCitationsWhenKnowledgeMatches(t *testing.T) {
	knowledge := newTestKnowledgeService(t)
	path := writeServiceTestFile(t, "runbook.md", "# 服务下线\n服务下线告警通常需要先查询最近 1 小时 panic 日志。")
	if _, err := knowledge.IndexFile(context.Background(), path); err != nil {
		t.Fatalf("IndexFile returned error: %v", err)
	}
	chat := NewChatService(true, slog.New(slog.NewTextHandler(io.Discard, nil)), knowledge)

	result, err := chat.Chat(context.Background(), "服务下线怎么处理")
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if len(result.Citations) == 0 {
		t.Fatal("expected citations")
	}
	if result.Citations[0].Source != "runbook.md" {
		t.Fatalf("source = %q", result.Citations[0].Source)
	}
}

func TestChatServiceReturnsNoKnowledgeMessage(t *testing.T) {
	knowledge := newTestKnowledgeService(t)
	chat := NewChatService(true, slog.New(slog.NewTextHandler(io.Discard, nil)), knowledge)

	result, err := chat.Chat(context.Background(), "服务下线怎么处理")
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if result.Answer != "知识库中没有检索到相关内容，请先上传对应 SOP 文档。" {
		t.Fatalf("answer = %q", result.Answer)
	}
	if len(result.Citations) != 0 {
		t.Fatalf("citations = %d, want 0", len(result.Citations))
	}
}
