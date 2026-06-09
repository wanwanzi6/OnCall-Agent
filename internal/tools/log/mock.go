package log

import (
	"context"
	"fmt"
	"time"

	"oncall-agent/internal/model/domain"
)

type QueryLogsInput struct {
	Alert domain.Alert
}

type MockTool struct{}

func NewMockTool() *MockTool {
	return &MockTool{}
}

func (t *MockTool) Name() string {
	return "mock_log"
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
	req, ok := input.(QueryLogsInput)
	if !ok {
		return nil, fmt.Errorf("invalid input for %s", t.Name())
	}
	return t.QueryLogs(req.Alert), nil
}

func (t *MockTool) QueryLogs(alert domain.Alert) domain.Evidence {
	query := alert.Service + " " + alert.Region + " " + alert.Instance + " +/-30m"
	return domain.Evidence{
		Query:   query,
		Summary: "实例多次健康检查失败，并出现依赖连接超时日志",
		Logs: []string{
			"2026-06-08T10:03:18+08:00 WARN healthcheck timeout path=/healthz",
			"2026-06-08T10:08:44+08:00 ERROR dependency redis connect timeout",
			"2026-06-08T10:14:57+08:00 WARN service discovery deregister instance=" + alert.Instance,
		},
	}
}
