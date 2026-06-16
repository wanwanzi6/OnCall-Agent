package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"oncall-agent/internal/infra/config"
	"oncall-agent/internal/infra/trace"
	"oncall-agent/internal/model/domain"
	"oncall-agent/internal/model/request"
	aiopstools "oncall-agent/internal/tools/aiops"
)

func TestAIOpsAnalyzeWithoutKnowledgeStillGeneratesReport(t *testing.T) {
	svc := newTestAIOpsService(nil, aiopstools.NewMockAlertProvider(), aiopstools.NewMockLogProvider(), aiopstools.NewMockMetricProvider())
	ctx := trace.WithTraceID(context.Background(), "trace-aiops")

	result, err := svc.Analyze(ctx, "", "")
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if result.TraceID != "trace-aiops" {
		t.Fatalf("trace_id = %q", result.TraceID)
	}
	assertStepNames(t, result.Steps)
	if len(result.Alerts) == 0 {
		t.Fatal("expected alerts")
	}
	if len(result.Citations) != 0 {
		t.Fatalf("citations = %d, want 0", len(result.Citations))
	}
	if !containsEvidenceType(result.Evidence, aiopstools.EvidenceTypeLog) || !containsEvidenceType(result.Evidence, aiopstools.EvidenceTypeMetric) {
		t.Fatalf("expected log and metric evidence: %+v", result.Evidence)
	}
	if !strings.Contains(result.Report, "告警分析报告") || !strings.Contains(result.Report, "panic") || !strings.Contains(result.Report, "restart_count") {
		t.Fatalf("report missing expected content:\n%s", result.Report)
	}
}

func TestAIOpsAnalyzeWithSOPReturnsCitations(t *testing.T) {
	knowledge := newTestKnowledgeService(t)
	path := writeServiceTestFile(t, "service-down-sop.md", "# 服务下线\n告警解释：服务下线可能因为服务 panic，导致 pod 重启造成的。\n解决方案：检查 restart_count 是否增加。")
	if _, err := knowledge.IndexFile(context.Background(), path); err != nil {
		t.Fatalf("IndexFile returned error: %v", err)
	}
	svc := newTestAIOpsService(knowledge, aiopstools.NewMockAlertProvider(), aiopstools.NewMockLogProvider(), aiopstools.NewMockMetricProvider())

	result, err := svc.Analyze(trace.WithTraceID(context.Background(), "trace-sop"), "", "")
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(result.Citations) == 0 {
		t.Fatal("expected citations")
	}
	if !containsEvidenceType(result.Evidence, aiopstools.EvidenceTypeSOP) {
		t.Fatalf("expected sop evidence: %+v", result.Evidence)
	}
}

func TestAIOpsAnalyzeProviderFailureDoesNotCrash(t *testing.T) {
	svc := newTestAIOpsService(nil, aiopstools.NewMockAlertProvider(), failingLogProvider{}, aiopstools.NewMockMetricProvider())

	result, err := svc.Analyze(trace.WithTraceID(context.Background(), "trace-failure"), "", "")
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	assertStepNames(t, result.Steps)
	step := findStep(result.Steps, "EvidenceCollector")
	if step.Status != "failed" {
		t.Fatalf("EvidenceCollector status = %q, want failed", step.Status)
	}
	if result.Report == "" {
		t.Fatal("report should still be generated")
	}
}

func TestAIOpsAnalyzeNoActiveAlerts(t *testing.T) {
	svc := newTestAIOpsService(nil, emptyAlertProvider{}, aiopstools.NewMockLogProvider(), aiopstools.NewMockMetricProvider())

	result, err := svc.Analyze(trace.WithTraceID(context.Background(), "trace-empty"), "", "")
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(result.Alerts) != 0 {
		t.Fatalf("alerts = %d, want 0", len(result.Alerts))
	}
	assertStepNames(t, result.Steps)
	if !strings.Contains(result.Report, "当前无活跃告警") {
		t.Fatalf("report = %q, want no active alert message", result.Report)
	}
}

func TestAIOpsAnalyzerSelectionDefaultsToRule(t *testing.T) {
	cfg := testAIOpsConfig()
	cfg.AIOps.Mode = ""
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
	result, err := svc.Analyze(trace.WithTraceID(context.Background(), "trace-rule-mode"), "", "")
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if result.Mode != AnalyzerModeRule || result.FallbackUsed {
		t.Fatalf("mode=%q fallback=%v, want rule false", result.Mode, result.FallbackUsed)
	}
}

func TestAIOpsAnalyzerSelectionAgentMode(t *testing.T) {
	cfg := testAIOpsConfig()
	cfg.AIOps.Mode = AnalyzerModeAgent
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
	result, err := svc.Analyze(trace.WithTraceID(context.Background(), "trace-agent-mode"), "", "")
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if result.Mode != AnalyzerModeAgent || result.FallbackUsed {
		t.Fatalf("mode=%q fallback=%v, want agent false", result.Mode, result.FallbackUsed)
	}
	if len(result.Steps) == 0 || !strings.HasPrefix(result.Steps[0].Name, "AgentTool:") {
		t.Fatalf("expected agent tool steps: %+v", result.Steps)
	}
	if result.Report == "" || len(result.Alerts) == 0 {
		t.Fatalf("expected agent report and alerts: %+v", result)
	}
	if result.Plan == nil || len(result.Iterations) == 0 || result.ReplanReason == "" {
		t.Fatalf("expected aiops agent plan-execute-replan trace: %+v", result)
	}
}

func TestAIOpsAnalyzerSelectionInvalidMode(t *testing.T) {
	cfg := testAIOpsConfig()
	cfg.AIOps.Mode = "invalid"
	_, err := NewAIOpsServiceWithProvidersFromConfig(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		aiopstools.NewMockAlertProvider(),
		aiopstools.NewMockLogProvider(),
		aiopstools.NewMockMetricProvider(),
		nil,
		cfg,
	)
	if err == nil {
		t.Fatal("expected invalid mode error")
	}
}

func TestAIOpsAgentFailureFallbackToRule(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	rule := NewRuleBasedAnalyzer(log, aiopstools.NewMockAlertProvider(), aiopstools.NewMockLogProvider(), aiopstools.NewMockMetricProvider(), nil, config.AIOpsConfig{Timeout: 5 * time.Second, SOPTopK: 3})
	svc := &AIOpsService{analyzer: failingAnalyzer{}, ruleAnalyzer: rule, mode: AnalyzerModeAgent, fallbackToRule: true, log: log}

	result, err := svc.Analyze(trace.WithTraceID(context.Background(), "trace-fallback"), "", "")
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if !result.FallbackUsed || result.Mode != AnalyzerModeRule {
		t.Fatalf("mode=%q fallback=%v, want rule true", result.Mode, result.FallbackUsed)
	}
	if len(result.Steps) == 0 || result.Steps[0].Name != "AgentAnalyzer" || result.Steps[0].Error == "" {
		t.Fatalf("fallback step missing original error: %+v", result.Steps)
	}
}

func TestAIOpsAgentFailureWithoutFallbackReturnsError(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	rule := NewRuleBasedAnalyzer(log, aiopstools.NewMockAlertProvider(), aiopstools.NewMockLogProvider(), aiopstools.NewMockMetricProvider(), nil, config.AIOpsConfig{Timeout: 5 * time.Second, SOPTopK: 3})
	svc := &AIOpsService{analyzer: failingAnalyzer{}, ruleAnalyzer: rule, mode: AnalyzerModeAgent, fallbackToRule: false, log: log}

	_, err := svc.Analyze(trace.WithTraceID(context.Background(), "trace-no-fallback"), "", "")
	if err == nil {
		t.Fatal("expected agent error without fallback")
	}
}

func newTestAIOpsService(knowledge *KnowledgeService, alertProvider aiopstools.AlertProvider, logProvider aiopstools.LogProvider, metricProvider aiopstools.MetricProvider) *AIOpsService {
	return NewAIOpsServiceWithProviders(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		alertProvider,
		logProvider,
		metricProvider,
		knowledge,
		config.AIOpsConfig{Timeout: 5 * time.Second, SOPTopK: 3},
	)
}

func testAIOpsConfig() config.Config {
	return config.Config{
		AIOps: config.AIOpsConfig{
			AlertProvider:  "mock",
			LogProvider:    "mock",
			MetricProvider: "mock",
			FallbackToRule: true,
			Agent:          config.AgentConfig{MaxSteps: 12, Timeout: 30 * time.Second},
			Timeout:        5 * time.Second,
			SOPTopK:        3,
		},
		LLM: config.LLMConfig{Provider: "mock", Timeout: 5 * time.Second},
	}
}

func assertStepNames(t *testing.T, steps []domain.WorkflowStep) {
	t.Helper()
	want := []string{"AlertCollector", "SOPRetriever", "EvidencePlanner", "EvidenceCollector", "RootCauseAnalyzer", "ReportGenerator"}
	if len(steps) != len(want) {
		t.Fatalf("steps = %d, want %d: %+v", len(steps), len(want), steps)
	}
	for i, name := range want {
		if steps[i].Name != name {
			t.Fatalf("step[%d] = %q, want %q", i, steps[i].Name, name)
		}
	}
}

func containsEvidenceType(evidence []domain.Evidence, evidenceType string) bool {
	for _, item := range evidence {
		if item.Type == evidenceType {
			return true
		}
	}
	return false
}

func findStep(steps []domain.WorkflowStep, name string) domain.WorkflowStep {
	for _, step := range steps {
		if step.Name == name {
			return step
		}
	}
	return domain.WorkflowStep{}
}

type failingLogProvider struct{}

func (f failingLogProvider) QueryLogs(context.Context, aiopstools.LogQuery) ([]aiopstools.LogEntry, error) {
	return nil, errors.New("log platform unavailable")
}

type emptyAlertProvider struct{}

func (e emptyAlertProvider) QueryActiveAlerts(context.Context, aiopstools.AlertFilter) ([]domain.Alert, error) {
	return nil, nil
}

type failingAnalyzer struct{}

func (f failingAnalyzer) Analyze(context.Context, request.AnalyzeRequest) (domain.AIOpsAnalyzeResult, error) {
	return domain.AIOpsAnalyzeResult{}, errors.New("agent runner failed")
}
