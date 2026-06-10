package aiops

import (
	"context"
	"strings"
	"testing"
)

func TestMockAlertProviderReturnsServiceDownAlert(t *testing.T) {
	alerts, err := NewMockAlertProvider().QueryActiveAlerts(context.Background(), AlertFilter{})
	if err != nil {
		t.Fatalf("QueryActiveAlerts returned error: %v", err)
	}
	if len(alerts) == 0 {
		t.Fatal("expected mock alert")
	}
	if alerts[0].Name != "服务下线" || alerts[0].Service != "billing-service" || alerts[0].Severity != "critical" {
		t.Fatalf("unexpected alert: %+v", alerts[0])
	}
}

func TestMockLogProviderReturnsPanicLogs(t *testing.T) {
	logs, err := NewMockLogProvider().QueryLogs(context.Background(), LogQuery{Service: "billing-service", Keyword: "panic"})
	if err != nil {
		t.Fatalf("QueryLogs returned error: %v", err)
	}
	if len(logs) == 0 {
		t.Fatal("expected mock logs")
	}
	if !strings.Contains(logs[0].Message, "panic") {
		t.Fatalf("first log = %q, want panic log", logs[0].Message)
	}
}

func TestMockMetricProviderReturnsRestartCount(t *testing.T) {
	points, err := NewMockMetricProvider().QueryMetrics(context.Background(), MetricQuery{Service: "billing-service", Metric: "restart_count"})
	if err != nil {
		t.Fatalf("QueryMetrics returned error: %v", err)
	}
	if len(points) == 0 {
		t.Fatal("expected mock metric points")
	}
	if points[0].Metric != "restart_count" {
		t.Fatalf("first metric = %q, want restart_count", points[0].Metric)
	}
}
