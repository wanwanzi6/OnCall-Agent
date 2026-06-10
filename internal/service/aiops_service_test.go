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
