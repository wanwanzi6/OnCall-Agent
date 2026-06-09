package aiops_agent

import (
	"context"
	"fmt"
	"log/slog"

	"oncall-agent/internal/infra/trace"
	"oncall-agent/internal/model/domain"
	"oncall-agent/internal/tools"
	alerttool "oncall-agent/internal/tools/alert"
	docstool "oncall-agent/internal/tools/docs"
	logtool "oncall-agent/internal/tools/log"
)

type Workflow struct {
	alerts tools.Tool
	docs   tools.Tool
	logs   tools.Tool
	log    *slog.Logger
}

func NewWorkflow(log *slog.Logger) *Workflow {
	if log == nil {
		log = slog.Default()
	}
	return &Workflow{
		alerts: alerttool.NewMockTool(),
		docs:   docstool.NewMockTool(),
		logs:   logtool.NewMockTool(),
		log:    log,
	}
}

func NewWorkflowWithTools(log *slog.Logger, alerts, docs, logs tools.Tool) *Workflow {
	if log == nil {
		log = slog.Default()
	}
	return &Workflow{alerts: alerts, docs: docs, logs: logs, log: log}
}

func (w *Workflow) Run(ctx context.Context, alertName, service string) domain.AnalyzeReport {
	steps := make([]domain.WorkflowStep, 0, 4)
	traceID := trace.FromContext(ctx)

	alertOutput, alertStep := w.execute(ctx, "AlertCollector", w.alerts, alerttool.ActiveAlertInput{
		AlertName: alertName,
		Service:   service,
	}, "读取 Mock 活跃告警")
	steps = append(steps, alertStep)
	alert, _ := alertOutput.(domain.Alert)
	if alertStep.Status != "success" {
		return domain.AnalyzeReport{
			Alert:      alert,
			RiskNotice: "当前阶段只生成分析报告，不自动重启服务或修改线上配置",
			Conclusion: "告警采集失败，无法继续生成完整分析报告。",
			Workflow:   append(steps, reportStep(traceID, "failed", "前置步骤失败，跳过报告生成")),
			Mock:       true,
			TraceID:    traceID,
		}
	}

	sopOutput, sopStep := w.execute(ctx, "SOPRetriever", w.docs, docstool.QuerySOPInput{Alert: alert}, "按告警名称匹配 Mock SOP")
	steps = append(steps, sopStep)
	sop, _ := sopOutput.(domain.SOP)

	evidenceOutput, evidenceStep := w.execute(ctx, "EvidenceCollector", w.logs, logtool.QueryLogsInput{Alert: alert}, "按服务、地域和实例收集 Mock 日志证据")
	steps = append(steps, evidenceStep)
	evidence, _ := evidenceOutput.(domain.Evidence)

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
		TraceID:        traceID,
	}

	status := "success"
	summary := "基于告警、SOP 和证据生成确定性分析报告"
	if sopStep.Status != "success" || evidenceStep.Status != "success" {
		status = "partial_success"
		summary = "部分工具执行失败，基于已获取信息生成降级分析报告"
	}
	report.Workflow = append(report.Workflow, reportStep(traceID, status, summary))

	return report
}

func (w *Workflow) execute(ctx context.Context, stepName string, tool tools.Tool, input any, summary string) (any, domain.WorkflowStep) {
	step := domain.WorkflowStep{
		Name:    stepName,
		Tool:    tool.Name(),
		Status:  "success",
		Summary: summary,
		TraceID: trace.FromContext(ctx),
	}

	toolCtx, cancel := context.WithTimeout(ctx, tool.Timeout())
	defer cancel()

	output, err := tool.Execute(toolCtx, input)
	if err != nil {
		step.Status = "failed"
		step.Error = err.Error()
		step.Summary = fmt.Sprintf("%s失败", summary)
		w.log.ErrorContext(ctx, "tool execution failed",
			"trace_id", step.TraceID,
			"agent_step_name", stepName,
			"tool_name", tool.Name(),
			"error", err.Error(),
		)
		return nil, step
	}

	step.Output = output
	w.log.InfoContext(ctx, "tool execution completed",
		"trace_id", step.TraceID,
		"agent_step_name", stepName,
		"tool_name", tool.Name(),
	)
	return output, step
}

func reportStep(traceID, status, summary string) domain.WorkflowStep {
	return domain.WorkflowStep{
		Name:    "ReportGenerator",
		Status:  status,
		Summary: summary,
		TraceID: traceID,
	}
}
