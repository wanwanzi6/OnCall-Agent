package service

import (
	"oncall-agent/internal/agent/aiops_agent"
	"oncall-agent/internal/model/domain"
)

type AIOpsService struct {
	workflow *aiops_agent.Workflow
}

func NewAIOpsService() *AIOpsService {
	return &AIOpsService{workflow: aiops_agent.NewWorkflow()}
}

func (s *AIOpsService) Analyze(alertName, service string) domain.AnalyzeReport {
	return s.workflow.Run(alertName, service)
}
