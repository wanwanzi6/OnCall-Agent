package aiops_agent

import (
	"oncall-agent/internal/model/domain"
	alerttool "oncall-agent/internal/tools/alert"
	docstool "oncall-agent/internal/tools/docs"
	logtool "oncall-agent/internal/tools/log"
)

type Workflow struct {
	alerts *alerttool.MockTool
	docs   *docstool.MockTool
	logs   *logtool.MockTool
}

func NewWorkflow() *Workflow {
	return &Workflow{
		alerts: alerttool.NewMockTool(),
		docs:   docstool.NewMockTool(),
		logs:   logtool.NewMockTool(),
	}
}

func (w *Workflow) Run(alertName, service string) domain.AnalyzeReport {
	steps := make([]domain.WorkflowStep, 0, 4)

	alert := w.alerts.ActiveAlert(alertName, service)
	steps = append(steps, domain.WorkflowStep{
		Name:    "AlertCollector",
		Status:  "success",
		Summary: "读取 Mock 活跃告警",
		Output:  alert,
	})

	sop := w.docs.QuerySOP(alert)
	steps = append(steps, domain.WorkflowStep{
		Name:    "SOPRetriever",
		Status:  "success",
		Summary: "按告警名称匹配 Mock SOP",
		Output:  sop,
	})

	evidence := w.logs.QueryLogs(alert)
	steps = append(steps, domain.WorkflowStep{
		Name:    "EvidenceCollector",
		Status:  "success",
		Summary: "按服务、地域和实例收集 Mock 日志证据",
		Output:  evidence,
	})

	report := domain.AnalyzeReport{
		Alert:          alert,
		SOP:            sop,
		Evidence:       evidence,
		PossibleCause:  "依赖服务连接超时导致健康检查失败，实例被服务发现摘除",
		Confidence:     "medium",
		Recommendation: []string{"检查依赖服务状态", "联系 " + alert.Service + " 发布负责人确认近期变更", "持续观察健康检查通过率和依赖错误日志"},
		RiskNotice:     "当前阶段只生成分析报告，不自动重启服务或修改线上配置",
		Conclusion:     "本次告警更可能由依赖连接异常触发，建议人工确认依赖状态后再执行处置。",
		Workflow:       steps,
		Mock:           true,
	}

	report.Workflow = append(report.Workflow, domain.WorkflowStep{
		Name:    "ReportGenerator",
		Status:  "success",
		Summary: "基于告警、SOP 和证据生成确定性分析报告",
	})

	return report
}
