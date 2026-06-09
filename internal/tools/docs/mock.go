package docs

import (
	"context"
	"fmt"
	"time"

	"oncall-agent/internal/model/domain"
)

type QuerySOPInput struct {
	Alert domain.Alert
}

type MockTool struct{}

func NewMockTool() *MockTool {
	return &MockTool{}
}

func (t *MockTool) Name() string {
	return "mock_docs"
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
	req, ok := input.(QuerySOPInput)
	if !ok {
		return nil, fmt.Errorf("invalid input for %s", t.Name())
	}
	return t.QuerySOP(req.Alert), nil
}

func (t *MockTool) QuerySOP(alert domain.Alert) domain.SOP {
	return domain.SOP{
		Title: alert.AlertName + "告警处理手册",
		Steps: []string{
			"确认告警服务名、实例和地域",
			"查询最近 30 分钟服务启动、停止和健康检查日志",
			"检查是否存在部署、重启、依赖连接失败或健康检查超时",
			"如果发现实例主动退出，优先查看退出前错误日志",
		},
	}
}
