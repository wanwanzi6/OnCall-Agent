package aiops

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"oncall-agent/internal/model/domain"
)

type PrometheusAlertProvider struct {
	baseURL string
	timeout time.Duration
	client  *http.Client
}

func NewPrometheusAlertProvider(baseURL string, timeout time.Duration) (*PrometheusAlertProvider, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("prometheus base_url is required")
	}
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return nil, fmt.Errorf("invalid prometheus base_url: %w", err)
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &PrometheusAlertProvider{
		baseURL: baseURL,
		timeout: timeout,
		client:  &http.Client{Timeout: timeout},
	}, nil
}

func (p *PrometheusAlertProvider) QueryActiveAlerts(ctx context.Context, filter AlertFilter) ([]domain.Alert, error) {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/api/v1/alerts", nil)
	if err != nil {
		return nil, fmt.Errorf("create prometheus alerts request: %w", err)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query prometheus alerts: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("query prometheus alerts: status %d", resp.StatusCode)
	}

	var body prometheusAlertsResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode prometheus alerts response: %w", err)
	}
	if body.Status != "success" {
		return nil, fmt.Errorf("prometheus alerts response status: %s", body.Status)
	}

	alerts := make([]domain.Alert, 0, len(body.Data.Alerts))
	for _, item := range body.Data.Alerts {
		if item.State != "firing" {
			continue
		}
		alert := mapPrometheusAlert(item)
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

type prometheusAlertsResponse struct {
	Status string `json:"status"`
	Data   struct {
		Alerts []prometheusAlert `json:"alerts"`
	} `json:"data"`
}

type prometheusAlert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	State       string            `json:"state"`
	ActiveAt    time.Time         `json:"activeAt"`
	Value       string            `json:"value"`
}

func mapPrometheusAlert(item prometheusAlert) domain.Alert {
	name := firstNonEmpty(item.Labels["alertname"], item.Labels["alert_name"], "unknown")
	service := firstNonEmpty(item.Labels["service"], item.Labels["job"], "unknown")
	severity := firstNonEmpty(item.Labels["severity"], "unknown")
	description := firstNonEmpty(item.Annotations["description"], item.Annotations["summary"], name)
	region := firstNonEmpty(item.Labels["region"], item.Labels["zone"])
	return domain.Alert{
		ID:          firstNonEmpty(item.Labels["id"], name+"-"+service),
		Name:        name,
		AlertName:   name,
		Service:     service,
		Severity:    severity,
		Status:      item.State,
		Description: description,
		Region:      region,
		Labels:      item.Labels,
		StartsAt:    item.ActiveAt,
		TriggeredAt: item.ActiveAt.Format(time.RFC3339),
		Instance:    item.Labels["instance"],
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
