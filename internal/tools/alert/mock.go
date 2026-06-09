package alert

import (
	"context"
	"fmt"
	"time"

	"oncall-agent/internal/model/domain"
)

type ActiveAlertInput struct {
	AlertName string
	Service   string
}

type MockTool struct{}

func NewMockTool() *MockTool {
	return &MockTool{}
}

func (t *MockTool) Name() string {
	return "mock_alert"
}

func (t *MockTool) Timeout() time.Duration {
	return time.Second
}

func (t *MockTool) Execute(ctx context.Context, input any) (any, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	req, ok := input.(ActiveAlertInput)
	if !ok {
		return nil, fmt.Errorf("invalid input for %s", t.Name())
	}
	return t.ActiveAlert(req.AlertName, req.Service), nil
}

func (t *MockTool) ActiveAlert(alertName, service string) domain.Alert {
	if alertName == "" {
		alertName = "服务下线"
	}
	if service == "" {
		service = "payment-api"
	}
	return domain.Alert{
		AlertName:   alertName,
		Service:     service,
		Severity:    "critical",
		Region:      "ap-shanghai",
		Instance:    service + "-03",
		TriggeredAt: "2026-06-08T10:15:00+08:00",
		Description: service + "-03 健康检查连续失败，实例从服务发现中摘除",
		Status:      "active",
	}
}
