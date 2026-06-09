package log

import "oncall-agent/internal/model/domain"

type MockTool struct{}

func NewMockTool() *MockTool {
	return &MockTool{}
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
