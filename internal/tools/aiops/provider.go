package aiops

import (
	"context"
	"time"

	"oncall-agent/internal/model/domain"
)

const (
	ProviderMock       = "mock"
	ProviderPrometheus = "prometheus"

	EvidenceTypeLog    = "log"
	EvidenceTypeMetric = "metric"
	EvidenceTypeSOP    = "sop"
)

type AlertProvider interface {
	QueryActiveAlerts(ctx context.Context, filter AlertFilter) ([]domain.Alert, error)
}

type LogProvider interface {
	QueryLogs(ctx context.Context, query LogQuery) ([]LogEntry, error)
}

type MetricProvider interface {
	QueryMetrics(ctx context.Context, query MetricQuery) ([]MetricPoint, error)
}

type AlertFilter struct {
	AlertName string
	Service   string
	Region    string
	Labels    map[string]string
}

type LogQuery struct {
	Service string
	Keyword string
	Region  string
	From    time.Time
	To      time.Time
	Limit   int
}

type LogEntry struct {
	Timestamp time.Time
	Service   string
	Level     string
	Message   string
	Metadata  map[string]string
}

type MetricQuery struct {
	Service string
	Metric  string
	Region  string
	From    time.Time
	To      time.Time
}

type MetricPoint struct {
	Timestamp time.Time
	Metric    string
	Value     float64
	Labels    map[string]string
}
