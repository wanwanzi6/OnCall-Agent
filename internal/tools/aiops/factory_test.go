package aiops

import (
	"testing"
	"time"

	"oncall-agent/internal/infra/config"
)

func TestNewProvidersDefaultsToMock(t *testing.T) {
	providers, err := NewProviders(config.Config{})
	if err != nil {
		t.Fatalf("NewProviders returned error: %v", err)
	}
	if _, ok := providers.Alert.(*MockAlertProvider); !ok {
		t.Fatalf("alert provider = %T, want *MockAlertProvider", providers.Alert)
	}
	if _, ok := providers.Log.(*MockLogProvider); !ok {
		t.Fatalf("log provider = %T, want *MockLogProvider", providers.Log)
	}
	if _, ok := providers.Metric.(*MockMetricProvider); !ok {
		t.Fatalf("metric provider = %T, want *MockMetricProvider", providers.Metric)
	}
}

func TestNewProvidersRejectsInvalidProvider(t *testing.T) {
	_, err := NewProviders(config.Config{
		AIOps: config.AIOpsConfig{
			AlertProvider:  "invalid",
			LogProvider:    "mock",
			MetricProvider: "mock",
		},
	})
	if err == nil {
		t.Fatal("expected invalid provider error")
	}
}

func TestPrometheusProviderCanBeConstructedWithoutNetwork(t *testing.T) {
	provider, err := NewAlertProvider(config.Config{
		AIOps: config.AIOpsConfig{AlertProvider: "prometheus"},
		Prometheus: config.PrometheusConfig{
			BaseURL: "http://localhost:9090",
			Timeout: 100 * time.Millisecond,
		},
	})
	if err != nil {
		t.Fatalf("NewAlertProvider returned error: %v", err)
	}
	if _, ok := provider.(*PrometheusAlertProvider); !ok {
		t.Fatalf("provider = %T, want *PrometheusAlertProvider", provider)
	}
}
