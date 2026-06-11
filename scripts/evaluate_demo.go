package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"oncall-agent/internal/infra/config"
	"oncall-agent/internal/infra/trace"
	"oncall-agent/internal/service"
	aiopstools "oncall-agent/internal/tools/aiops"
)

type ragCase struct {
	Name     string
	Query    string
	Expected []string
}

type ragMetrics struct {
	Total         int
	RecallAt1     int
	RecallAt3     int
	ReciprocalSum float64
}

func main() {
	ctx := trace.WithTraceID(context.Background(), "eval-demo")
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg, err := config.Load("")
	must(err)

	knowledge := service.NewKnowledgeService(cfg.Mock.Enabled, cfg.Knowledge, cfg.RAG, log)
	indexResult, err := knowledge.IndexFile(ctx, "docs/demo/告警处理手册.md")
	must(err)

	ragCases := []ragCase{
		{Name: "service-down-cause", Query: "服务下线告警常见原因是什么", Expected: []string{"应用 panic", "容器重启", "依赖超时"}},
		{Name: "panic-log", Query: "服务下线时应该优先搜索哪些 panic 日志关键字", Expected: []string{"panic", "runtime error", "nil pointer"}},
		{Name: "restart-metric", Query: "如何确认 pod 是否发生重启", Expected: []string{"restart_count", "重启"}},
		{Name: "root-cause", Query: "panic 和 restart_count 同时出现说明什么", Expected: []string{"应用 panic", "实例重启", "服务下线"}},
		{Name: "rollback", Query: "怀疑新版本导致服务下线时该怎么处理", Expected: []string{"回滚", "最近发布"}},
		{Name: "safety", Query: "系统会不会自动修复或关闭告警", Expected: []string{"不自动修复", "不执行 SQL", "不关闭告警"}},
		{Name: "severity", Query: "billing-service 服务下线告警级别和地域", Expected: []string{"critical", "ap-guangzhou"}},
		{Name: "health", Query: "服务实例健康状态需要观察什么指标", Expected: []string{"restart_count", "error_rate", "服务实例健康状态"}},
	}

	rag := evaluateRAG(ctx, knowledge, ragCases)

	aiops := service.NewAIOpsServiceWithProviders(
		log,
		aiopstools.NewMockAlertProvider(),
		aiopstools.NewMockLogProvider(),
		aiopstools.NewMockMetricProvider(),
		knowledge,
		config.AIOpsConfig{Timeout: 5 * time.Second, SOPTopK: 3},
	)
	start := time.Now()
	result, err := aiops.Analyze(ctx, "服务下线", "billing-service")
	must(err)
	latency := time.Since(start)

	successSteps := 0
	for _, step := range result.Steps {
		if step.Status == "success" {
			successSteps++
		}
	}
	reportOK := strings.Contains(result.Report, "panic") &&
		strings.Contains(result.Report, "restart_count") &&
		strings.Contains(result.Report, "服务下线")

	fmt.Println("# Demo Evaluation")
	fmt.Println()
	fmt.Printf("- Indexed documents: 1\n")
	fmt.Printf("- Indexed chunks: %d\n", indexResult.ChunkCount)
	fmt.Printf("- RAG eval cases: %d\n", rag.Total)
	fmt.Printf("- RAG Recall@1: %.1f%% (%d/%d)\n", pct(rag.RecallAt1, rag.Total), rag.RecallAt1, rag.Total)
	fmt.Printf("- RAG Recall@3: %.1f%% (%d/%d)\n", pct(rag.RecallAt3, rag.Total), rag.RecallAt3, rag.Total)
	fmt.Printf("- RAG MRR@3: %.3f\n", rag.ReciprocalSum/float64(rag.Total))
	fmt.Printf("- AIOps workflow success steps: %d/%d\n", successSteps, len(result.Steps))
	fmt.Printf("- AIOps evidence count: %d\n", len(result.Evidence))
	fmt.Printf("- AIOps citation count: %d\n", len(result.Citations))
	fmt.Printf("- AIOps report contains expected root-cause signals: %t\n", reportOK)
	fmt.Printf("- AIOps demo latency: %s\n", latency.Round(time.Millisecond))
}

func evaluateRAG(ctx context.Context, knowledge *service.KnowledgeService, cases []ragCase) ragMetrics {
	metrics := ragMetrics{Total: len(cases)}
	for _, item := range cases {
		results, err := knowledge.Search(ctx, item.Query, 3)
		must(err)
		rank := 0
		for i, result := range results {
			if containsAll(result.Chunk.Content, item.Expected) {
				rank = i + 1
				break
			}
		}
		if rank == 1 {
			metrics.RecallAt1++
		}
		if rank > 0 && rank <= 3 {
			metrics.RecallAt3++
			metrics.ReciprocalSum += 1.0 / float64(rank)
		}
	}
	return metrics
}

func containsAll(text string, expected []string) bool {
	for _, item := range expected {
		if !strings.Contains(text, item) {
			return false
		}
	}
	return true
}

func pct(value, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(value) * 100 / float64(total)
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
