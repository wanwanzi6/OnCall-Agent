package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"oncall-agent/internal/infra/trace"
	aiopstools "oncall-agent/internal/tools/aiops"
)

func TestAIOpsAgentToolQueryActiveAlerts(t *testing.T) {
	tools := newTestAgentTools(nil, aiopstools.NewMockAlertProvider(), aiopstools.NewMockLogProvider(), aiopstools.NewMockMetricProvider())
	output, err := tools.ActiveAlerts.InvokableRun(trace.WithTraceID(context.Background(), "trace-tool-alerts"), `{}`)
	if err != nil {
		t.Fatalf("InvokableRun returned error: %v", err)
	}
	var decoded queryActiveAlertsOutput
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(decoded.Alerts) == 0 {
		t.Fatal("expected mock alerts")
	}
	assertRecordedTool(t, tools, "query_active_alerts", stepStatusSuccess)
}

func TestAIOpsAgentToolQueryInternalDocs(t *testing.T) {
	knowledge := newTestKnowledgeService(t)
	path := writeServiceTestFile(t, "agent-sop.md", "# 服务下线\n服务 panic 后需要检查 restart_count。")
	if _, err := knowledge.IndexFile(context.Background(), path); err != nil {
		t.Fatalf("IndexFile returned error: %v", err)
	}
	tools := newTestAgentTools(knowledge, aiopstools.NewMockAlertProvider(), aiopstools.NewMockLogProvider(), aiopstools.NewMockMetricProvider())

	output, err := tools.InternalDocs.InvokableRun(trace.WithTraceID(context.Background(), "trace-tool-docs"), `{"query":"服务下线 panic","top_k":3}`)
	if err != nil {
		t.Fatalf("InvokableRun returned error: %v", err)
	}
	var decoded queryInternalDocsOutput
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(decoded.Results) == 0 || len(decoded.Evidence) == 0 {
		t.Fatalf("expected docs results and evidence: %+v", decoded)
	}
	assertRecordedTool(t, tools, "query_internal_docs", stepStatusSuccess)
}

func TestAIOpsAgentToolQueryLogsAndMetrics(t *testing.T) {
	tools := newTestAgentTools(nil, aiopstools.NewMockAlertProvider(), aiopstools.NewMockLogProvider(), aiopstools.NewMockMetricProvider())
	ctx := trace.WithTraceID(context.Background(), "trace-tool-evidence")
	logsOutput, err := tools.Logs.InvokableRun(ctx, `{"service":"billing-service","keyword":"panic","from":"2026-06-10T08:30:00+08:00","to":"2026-06-10T10:30:00+08:00","limit":20}`)
	if err != nil {
		t.Fatalf("logs InvokableRun returned error: %v", err)
	}
	var logs queryLogsOutput
	if err := json.Unmarshal([]byte(logsOutput), &logs); err != nil {
		t.Fatalf("unmarshal logs output: %v", err)
	}
	if len(logs.Logs) == 0 || logs.Evidence.Type != aiopstools.EvidenceTypeLog {
		t.Fatalf("expected log evidence: %+v", logs)
	}

	metricsOutput, err := tools.Metrics.InvokableRun(ctx, `{"service":"billing-service","metric":"restart_count,error_rate","from":"2026-06-10T08:30:00+08:00","to":"2026-06-10T10:30:00+08:00"}`)
	if err != nil {
		t.Fatalf("metrics InvokableRun returned error: %v", err)
	}
	var metrics queryMetricsOutput
	if err := json.Unmarshal([]byte(metricsOutput), &metrics); err != nil {
		t.Fatalf("unmarshal metrics output: %v", err)
	}
	if len(metrics.Metrics) == 0 || metrics.Evidence.Type != aiopstools.EvidenceTypeMetric {
		t.Fatalf("expected metric evidence: %+v", metrics)
	}
}

func TestAIOpsAgentToolFailureReturnsError(t *testing.T) {
	tools := newTestAgentTools(nil, aiopstools.NewMockAlertProvider(), failingLogProvider{}, aiopstools.NewMockMetricProvider())
	_, err := tools.Logs.InvokableRun(trace.WithTraceID(context.Background(), "trace-tool-failed"), `{"service":"billing-service","keyword":"panic"}`)
	if err == nil {
		t.Fatal("expected tool error")
	}
	assertRecordedTool(t, tools, "query_logs", stepStatusFailed)
}

func TestAIOpsAgentPromptContainsSafetyConstraints(t *testing.T) {
	for _, want := range []string{"不允许编造日志", "不允许执行修复动作", "不允许执行 SQL", "不允许请求系统命令", "不允许关闭告警", "告警分析报告", "一、活跃告警", "六、结论"} {
		if !strings.Contains(AIOpsAgentSystemPrompt, want) {
			t.Fatalf("prompt missing %q", want)
		}
	}
}

func newTestAgentTools(knowledge *KnowledgeService, alertProvider aiopstools.AlertProvider, logProvider aiopstools.LogProvider, metricProvider aiopstools.MetricProvider) *AIOpsAgentTools {
	return NewAIOpsAgentTools(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		alertProvider,
		logProvider,
		metricProvider,
		knowledge,
		3,
		5*time.Second,
	)
}

func assertRecordedTool(t *testing.T, tools *AIOpsAgentTools, name, status string) {
	t.Helper()
	steps := tools.Recorder.Steps()
	for _, step := range steps {
		if step.Tool == name {
			if step.Status != status {
				t.Fatalf("tool %s status=%s, want %s", name, step.Status, status)
			}
			return
		}
	}
	t.Fatalf("tool %s was not recorded: %+v", name, steps)
}

type failingMetricProvider struct{}

func (f failingMetricProvider) QueryMetrics(context.Context, aiopstools.MetricQuery) ([]aiopstools.MetricPoint, error) {
	return nil, errors.New("metric platform unavailable")
}

func TestAIOpsAgentToolQueryMetricsFailureReturnsError(t *testing.T) {
	tools := newTestAgentTools(nil, aiopstools.NewMockAlertProvider(), aiopstools.NewMockLogProvider(), failingMetricProvider{})
	_, err := tools.Metrics.InvokableRun(trace.WithTraceID(context.Background(), "trace-tool-metric-failed"), `{"service":"billing-service","metric":"restart_count"}`)
	if err == nil {
		t.Fatal("expected metric tool error")
	}
	assertRecordedTool(t, tools, "query_metrics", stepStatusFailed)
}
