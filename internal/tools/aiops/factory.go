package aiops

import (
	"fmt"
	"strings"

	"oncall-agent/internal/infra/config"
)

type Providers struct {
	Alert  AlertProvider
	Log    LogProvider
	Metric MetricProvider
}

func NewProviders(cfg config.Config) (Providers, error) {
	alertProvider, err := NewAlertProvider(cfg)
	if err != nil {
		return Providers{}, err
	}
	logProvider, err := NewLogProvider(cfg)
	if err != nil {
		return Providers{}, err
	}
	metricProvider, err := NewMetricProvider(cfg)
	if err != nil {
		return Providers{}, err
	}
	return Providers{Alert: alertProvider, Log: logProvider, Metric: metricProvider}, nil
}

func NewAlertProvider(cfg config.Config) (AlertProvider, error) {
	switch normalizeProvider(cfg.AIOps.AlertProvider) {
	case "", ProviderMock:
		return NewMockAlertProvider(), nil
	case ProviderPrometheus:
		return NewPrometheusAlertProvider(cfg.Prometheus.BaseURL, cfg.Prometheus.Timeout)
	default:
		return nil, fmt.Errorf("unsupported aiops alert_provider %q", cfg.AIOps.AlertProvider)
	}
}

func NewLogProvider(cfg config.Config) (LogProvider, error) {
	switch normalizeProvider(cfg.AIOps.LogProvider) {
	case "", ProviderMock:
		return NewMockLogProvider(), nil
	default:
		return nil, fmt.Errorf("unsupported aiops log_provider %q", cfg.AIOps.LogProvider)
	}
}

func NewMetricProvider(cfg config.Config) (MetricProvider, error) {
	switch normalizeProvider(cfg.AIOps.MetricProvider) {
	case "", ProviderMock:
		return NewMockMetricProvider(), nil
	default:
		return nil, fmt.Errorf("unsupported aiops metric_provider %q", cfg.AIOps.MetricProvider)
	}
}

func normalizeProvider(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}
