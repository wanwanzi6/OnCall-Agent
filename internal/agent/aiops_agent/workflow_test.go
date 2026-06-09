package aiops_agent

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"oncall-agent/internal/infra/trace"
	alerttool "oncall-agent/internal/tools/alert"
	docstool "oncall-agent/internal/tools/docs"
	logtool "oncall-agent/internal/tools/log"
)

func TestWorkflowRunSuccessIncludesTraceAndSteps(t *testing.T) {
	ctx := trace.WithTraceID(context.Background(), "trace-workflow")
	workflow := NewWorkflow(testLogger())

	report := workflow.Run(ctx, "服务下线", "payment-api")

	if report.TraceID != "trace-workflow" {
		t.Fatalf("report trace_id = %q", report.TraceID)
	}
	if len(report.Workflow) != 4 {
		t.Fatalf("steps = %d, want 4", len(report.Workflow))
	}
	for _, step := range report.Workflow {
		if step.TraceID != "trace-workflow" {
			t.Fatalf("step %s trace_id = %q", step.Name, step.TraceID)
		}
		if step.Status != "success" {
			t.Fatalf("step %s status = %q", step.Name, step.Status)
		}
	}
}

func TestWorkflowToolFailureDoesNotExitFlow(t *testing.T) {
	ctx := trace.WithTraceID(context.Background(), "trace-failure")
	workflow := NewWorkflowWithTools(
		testLogger(),
		alerttool.NewMockTool(),
		failingTool{name: "failing_docs"},
		logtool.NewMockTool(),
	)

	report := workflow.Run(ctx, "服务下线", "payment-api")

	if len(report.Workflow) != 4 {
		t.Fatalf("steps = %d, want 4", len(report.Workflow))
	}
	if report.Workflow[1].Status != "failed" {
		t.Fatalf("docs step status = %q, want failed", report.Workflow[1].Status)
	}
	if report.Workflow[1].Error == "" {
		t.Fatal("failed tool step should include error")
	}
	if report.Workflow[3].Status != "partial_success" {
		t.Fatalf("report step status = %q, want partial_success", report.Workflow[3].Status)
	}
}

func TestWorkflowWithDefaultMockTools(t *testing.T) {
	ctx := trace.WithTraceID(context.Background(), "trace-default")
	workflow := NewWorkflowWithTools(
		testLogger(),
		alerttool.NewMockTool(),
		docstool.NewMockTool(),
		logtool.NewMockTool(),
	)

	report := workflow.Run(ctx, "", "")
	if report.Alert.Service == "" {
		t.Fatal("default mock alert should include service")
	}
}

type failingTool struct {
	name string
}

func (t failingTool) Name() string {
	return t.name
}

func (t failingTool) Timeout() time.Duration {
	return time.Second
}

func (t failingTool) Execute(context.Context, any) (any, error) {
	return nil, errors.New("mock tool failed")
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
