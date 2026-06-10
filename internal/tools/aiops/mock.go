package aiops

import (
	"context"
	"strings"
	"time"

	"oncall-agent/internal/model/domain"
)

type MockAlertProvider struct {
	Alerts []domain.Alert
}

func NewMockAlertProvider() *MockAlertProvider {
	startsAt := time.Date(2026, 6, 10, 9, 30, 0, 0, time.FixedZone("CST", 8*60*60))
	return &MockAlertProvider{
		Alerts: []domain.Alert{
			{
				ID:          "alert-billing-service-down",
				Name:        "服务下线",
				AlertName:   "服务下线",
				Service:     "billing-service",
				Severity:    "critical",
				Status:      "firing",
				Description: "服务实例下线，可能由 panic 或 pod 重启导致",
				Region:      "ap-guangzhou",
				Labels: map[string]string{
					"instance":  "billing-service-0",
					"namespace": "prod",
				},
				StartsAt:    startsAt,
				TriggeredAt: startsAt.Format(time.RFC3339),
				Instance:    "billing-service-0",
			},
		},
	}
}

func (p *MockAlertProvider) QueryActiveAlerts(ctx context.Context, filter AlertFilter) ([]domain.Alert, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	alerts := make([]domain.Alert, 0, len(p.Alerts))
	for _, alert := range p.Alerts {
		if filter.AlertName != "" && !strings.Contains(alert.Name, filter.AlertName) {
			continue
		}
		if filter.Service != "" && alert.Service != filter.Service {
			continue
		}
		if filter.Region != "" && alert.Region != filter.Region {
			continue
		}
		alerts = append(alerts, alert)
	}
	return alerts, nil
}

type MockLogProvider struct{}

func NewMockLogProvider() *MockLogProvider {
	return &MockLogProvider{}
}

func (p *MockLogProvider) QueryLogs(ctx context.Context, query LogQuery) ([]LogEntry, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	keyword := strings.ToLower(query.Keyword)
	service := query.Service
	if service == "" {
		service = "billing-service"
	}
	if keyword != "" && !strings.Contains(keyword, "panic") && !strings.Contains(strings.ToLower(service), keyword) {
		return nil, nil
	}

	base := query.From
	if base.IsZero() {
		base = time.Date(2026, 6, 10, 9, 28, 0, 0, time.FixedZone("CST", 8*60*60))
	}
	return []LogEntry{
		{
			Timestamp: base.Add(12 * time.Minute),
			Service:   service,
			Level:     "ERROR",
			Message:   "panic: runtime error: invalid memory address or nil pointer dereference",
			Metadata:  map[string]string{"pod": service + "-0", "region": query.Region},
		},
		{
			Timestamp: base.Add(13 * time.Minute),
			Service:   service,
			Level:     "WARN",
			Message:   "pod restarted due to application panic",
			Metadata:  map[string]string{"pod": service + "-0", "region": query.Region},
		},
	}, nil
}

type MockMetricProvider struct{}

func NewMockMetricProvider() *MockMetricProvider {
	return &MockMetricProvider{}
}

func (p *MockMetricProvider) QueryMetrics(ctx context.Context, query MetricQuery) ([]MetricPoint, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	base := query.From
	if base.IsZero() {
		base = time.Date(2026, 6, 10, 9, 28, 0, 0, time.FixedZone("CST", 8*60*60))
	}
	service := query.Service
	if service == "" {
		service = "billing-service"
	}
	return []MetricPoint{
		{
			Timestamp: base.Add(10 * time.Minute),
			Metric:    "restart_count",
			Value:     3,
			Labels:    map[string]string{"service": service, "region": query.Region},
		},
		{
			Timestamp: base.Add(15 * time.Minute),
			Metric:    "error_rate",
			Value:     0.18,
			Labels:    map[string]string{"service": service, "region": query.Region},
		},
	}, nil
}
