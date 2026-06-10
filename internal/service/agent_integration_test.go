package service

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"oncall-agent/internal/infra/config"
	"oncall-agent/internal/infra/trace"
	aiopstools "oncall-agent/internal/tools/aiops"
)

func TestRealLLMAgentIntegration(t *testing.T) {
	if os.Getenv("RUN_AGENT_INTEGRATION_TEST") != "1" {
		t.Skip("set RUN_AGENT_INTEGRATION_TEST=1 to run real LLM agent integration test")
	}
	if strings.TrimSpace(os.Getenv("LLM_API_KEY")) == "" || strings.TrimSpace(os.Getenv("LLM_MODEL")) == "" {
		t.Fatal("LLM_API_KEY and LLM_MODEL are required when RUN_AGENT_INTEGRATION_TEST=1")
	}
	cfg := config.Config{
		AIOps: config.AIOpsConfig{
			AlertProvider:  "mock",
			LogProvider:    "mock",
			MetricProvider: "mock",
			Mode:           AnalyzerModeAgent,
			FallbackToRule: false,
			Agent:          config.AgentConfig{MaxSteps: 12, Timeout: 60 * time.Second},
			Timeout:        10 * time.Second,
			SOPTopK:        3,
		},
		LLM: config.LLMConfig{
			Provider: envOrDefaultString("LLM_PROVIDER", "openai-compatible"),
			APIKey:   os.Getenv("LLM_API_KEY"),
			BaseURL:  os.Getenv("LLM_BASE_URL"),
			Model:    os.Getenv("LLM_MODEL"),
			Timeout:  60 * time.Second,
		},
	}
	svc, err := NewAIOpsServiceWithProvidersFromConfig(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		aiopstools.NewMockAlertProvider(),
		aiopstools.NewMockLogProvider(),
		aiopstools.NewMockMetricProvider(),
		nil,
		cfg,
	)
	if err != nil {
		t.Fatalf("NewAIOpsServiceWithProvidersFromConfig returned error: %v", err)
	}
	result, err := svc.Analyze(trace.WithTraceID(context.Background(), "trace-agent-integration"), "服务下线", "billing-service")
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if result.Mode != AnalyzerModeAgent || result.Report == "" || len(result.Evidence) == 0 {
		t.Fatalf("unexpected agent result: %+v", result)
	}
}

func envOrDefaultString(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
