package alert

import "oncall-agent/internal/model/domain"

type MockTool struct{}

func NewMockTool() *MockTool {
	return &MockTool{}
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
