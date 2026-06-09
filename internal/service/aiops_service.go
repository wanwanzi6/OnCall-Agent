package service

import (
	"context"
	"log/slog"

	"oncall-agent/internal/agent/aiops_agent"
	"oncall-agent/internal/infra/trace"
	"oncall-agent/internal/model/domain"
)

type AIOpsService struct {
	workflow *aiops_agent.Workflow
	log      *slog.Logger
}

func NewAIOpsService(log *slog.Logger) *AIOpsService {
	if log == nil {
		log = slog.Default()
	}
	return &AIOpsService{workflow: aiops_agent.NewWorkflow(log), log: log}
}

func (s *AIOpsService) Analyze(ctx context.Context, alertName, service string) (domain.AnalyzeReport, error) {
	s.log.InfoContext(ctx, "aiops analyze requested",
		"trace_id", trace.FromContext(ctx),
		"service_name", service,
	)
	return s.workflow.Run(ctx, alertName, service), nil
}
