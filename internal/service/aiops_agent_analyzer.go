package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	einomodel "github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"oncall-agent/internal/infra/config"
	"oncall-agent/internal/infra/trace"
	"oncall-agent/internal/llm"
	"oncall-agent/internal/model/domain"
	"oncall-agent/internal/model/request"
	aiopstools "oncall-agent/internal/tools/aiops"
)

type EinoAgentAnalyzer struct {
	runner      AgentRunner
	model       einomodel.BaseChatModel
	toolFactory func() *AIOpsAgentTools
	timeout     time.Duration
	maxStep     int
	log         *slog.Logger
}

type AgentRunner interface {
	Run(ctx context.Context, input AgentRunInput) (AgentRunOutput, error)
}

type AgentRunInput struct {
	Request      request.AnalyzeRequest
	SystemPrompt string
	Tools        map[string]einotool.InvokableTool
	Model        einomodel.BaseChatModel
	MaxSteps     int
}

type AgentRunOutput struct {
	Report       string
	Alerts       []domain.Alert
	Evidence     []domain.Evidence
	Citations    []domain.Citation
	Plan         *domain.AgentPlan
	Iterations   []domain.AgentIteration
	ReplanReason string
}

func NewEinoAgentAnalyzerFromConfig(log *slog.Logger, alertProvider aiopstools.AlertProvider, logProvider aiopstools.LogProvider, metricProvider aiopstools.MetricProvider, knowledge *KnowledgeService, cfg config.Config) (*EinoAgentAnalyzer, error) {
	model, err := llm.NewEinoToolCallingModel(cfg.LLM)
	if err != nil {
		return nil, err
	}
	timeout := cfg.AIOps.Agent.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	maxSteps := cfg.AIOps.Agent.MaxSteps
	if maxSteps <= 0 {
		maxSteps = 12
	}
	toolTimeout := cfg.AIOps.Timeout
	if toolTimeout <= 0 {
		toolTimeout = 10 * time.Second
	}
	return NewEinoAgentAnalyzer(log, model, NewEinoPlanExecuteReplanRunner(), alertProvider, logProvider, metricProvider, knowledge, cfg.AIOps.SOPTopK, toolTimeout, timeout, maxSteps), nil
}

func NewEinoAgentAnalyzer(log *slog.Logger, model einomodel.BaseChatModel, runner AgentRunner, alertProvider aiopstools.AlertProvider, logProvider aiopstools.LogProvider, metricProvider aiopstools.MetricProvider, knowledge *KnowledgeService, sopTopK int, toolTimeout, timeout time.Duration, maxSteps int) *EinoAgentAnalyzer {
	if log == nil {
		log = slog.Default()
	}
	if model == nil {
		model = llm.MockEinoChatModel{}
	}
	if runner == nil {
		runner = NewEinoPlanExecuteReplanRunner()
	}
	if alertProvider == nil {
		alertProvider = aiopstools.NewMockAlertProvider()
	}
	if logProvider == nil {
		logProvider = aiopstools.NewMockLogProvider()
	}
	if metricProvider == nil {
		metricProvider = aiopstools.NewMockMetricProvider()
	}
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	if maxSteps <= 0 {
		maxSteps = 12
	}
	return &EinoAgentAnalyzer{
		runner: runner,
		model:  model,
		toolFactory: func() *AIOpsAgentTools {
			return NewAIOpsAgentTools(log, alertProvider, logProvider, metricProvider, knowledge, sopTopK, toolTimeout)
		},
		timeout: timeout,
		maxStep: maxSteps,
		log:     log,
	}
}

func (a *EinoAgentAnalyzer) Analyze(ctx context.Context, req request.AnalyzeRequest) (domain.AIOpsAnalyzeResult, error) {
	traceID := trace.FromContext(ctx)
	if traceID == "" {
		traceID = trace.NewID()
		ctx = trace.WithTraceID(ctx, traceID)
	}
	agentCtx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	if a.toolFactory == nil {
		return domain.AIOpsAnalyzeResult{}, fmt.Errorf("agent tool factory is not configured")
	}
	tools := a.toolFactory()
	output, err := a.runner.Run(agentCtx, AgentRunInput{
		Request:      req,
		SystemPrompt: AIOpsAgentSystemPrompt,
		Tools:        tools.ByName(),
		Model:        a.model,
		MaxSteps:     a.maxStep,
	})
	if err != nil {
		return domain.AIOpsAnalyzeResult{}, err
	}
	result := domain.AIOpsAnalyzeResult{
		TraceID:      traceID,
		Mode:         AnalyzerModeAgent,
		Report:       output.Report,
		Alerts:       output.Alerts,
		Steps:        tools.Recorder.Steps(),
		Evidence:     output.Evidence,
		Citations:    output.Citations,
		Plan:         output.Plan,
		Iterations:   output.Iterations,
		ReplanReason: output.ReplanReason,
	}
	if strings.TrimSpace(result.Report) == "" {
		result.Report = "告警分析报告\n\n证据不足，agent 未生成有效报告。"
	}
	a.log.InfoContext(ctx, "aiops agent analyzer completed",
		"trace_id", traceID,
		"alert_count", len(result.Alerts),
		"evidence_count", len(result.Evidence),
		"citation_count", len(result.Citations),
	)
	return result, nil
}

type EinoPlanExecuteReplanRunner struct{}

type aiopsPERState struct {
	Input        AgentRunInput
	Output       AgentRunOutput
	Plan         *domain.AgentPlan
	Iterations   []domain.AgentIteration
	ReplanReason string
}

func NewEinoPlanExecuteReplanRunner() *EinoPlanExecuteReplanRunner {
	return &EinoPlanExecuteReplanRunner{}
}

func (r *EinoPlanExecuteReplanRunner) Run(ctx context.Context, input AgentRunInput) (AgentRunOutput, error) {
	if input.Model == nil {
		return AgentRunOutput{}, fmt.Errorf("chat model is required")
	}
	if len(input.Tools) == 0 {
		return AgentRunOutput{}, fmt.Errorf("agent tools are required")
	}
	if input.MaxSteps > 0 && input.MaxSteps < 4 {
		return AgentRunOutput{}, fmt.Errorf("agent max_steps %d is too small for required analysis flow", input.MaxSteps)
	}
	graph := compose.NewGraph[aiopsPERState, aiopsPERState]()
	if err := graph.AddLambdaNode("Planner", compose.InvokableLambda(r.plan)); err != nil {
		return AgentRunOutput{}, err
	}
	if err := graph.AddLambdaNode("Executor", compose.InvokableLambda(r.execute)); err != nil {
		return AgentRunOutput{}, err
	}
	if err := graph.AddLambdaNode("Replanner", compose.InvokableLambda(r.replan)); err != nil {
		return AgentRunOutput{}, err
	}
	if err := graph.AddLambdaNode("ReportGenerator", compose.InvokableLambda(r.report)); err != nil {
		return AgentRunOutput{}, err
	}
	for _, edge := range [][2]string{
		{compose.START, "Planner"},
		{"Planner", "Executor"},
		{"Executor", "Replanner"},
		{"Replanner", "ReportGenerator"},
		{"ReportGenerator", compose.END},
	} {
		if err := graph.AddEdge(edge[0], edge[1]); err != nil {
			return AgentRunOutput{}, err
		}
	}
	runnable, err := graph.Compile(ctx, compose.WithGraphName("AIOpsPlanExecuteReplan"), compose.WithMaxRunSteps(input.MaxSteps))
	if err != nil {
		return AgentRunOutput{}, err
	}
	out, err := runnable.Invoke(ctx, aiopsPERState{Input: input})
	if err != nil {
		return AgentRunOutput{}, err
	}
	out.Output.Plan = out.Plan
	out.Output.Iterations = out.Iterations
	out.Output.ReplanReason = out.ReplanReason
	return out.Output, nil
}

func (r *EinoPlanExecuteReplanRunner) plan(ctx context.Context, in aiopsPERState) (aiopsPERState, error) {
	in.Plan = newAgentPlan("AIOps 告警分析 Plan-Execute-Replan", []domain.AgentPlanStep{
		{ID: "alerts", Name: "QueryActiveAlerts", Tool: "query_active_alerts", Status: stepStatusSkipped, Rationale: "先确认活跃告警范围"},
		{ID: "time", Name: "GetCurrentTime", Tool: "get_current_time", Status: stepStatusSkipped, Rationale: "为缺失触发时间的告警提供时间窗口"},
		{ID: "docs", Name: "QueryInternalDocs", Tool: "query_internal_docs", Status: stepStatusSkipped, Rationale: "检索 SOP 和内部知识"},
		{ID: "logs", Name: "QueryLogs", Tool: "query_logs", Status: stepStatusSkipped, Rationale: "收集日志证据"},
		{ID: "metrics", Name: "QueryMetrics", Tool: "query_metrics", Status: stepStatusSkipped, Rationale: "收集指标证据"},
		{ID: "report", Name: "ReportGenerator", Tool: "chat_model", Status: stepStatusSkipped, Rationale: "基于证据生成报告"},
	})
	in.Iterations = append(in.Iterations, domain.AgentIteration{Index: 1, Phase: "Planner", StepID: "planner", Observation: "生成 AIOps 排障计划", StartedAt: time.Now(), EndedAt: time.Now()})
	return in, nil
}

func (r *EinoPlanExecuteReplanRunner) execute(ctx context.Context, in aiopsPERState) (aiopsPERState, error) {
	toolsNode, err := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{Tools: invokableToolsAsBase(in.Input.Tools), ExecuteSequentially: true})
	if err != nil {
		return in, err
	}
	alertOutput, err := invokeAIOpsToolNode(ctx, toolsNode, "query_active_alerts", queryActiveAlertsInput{AlertName: in.Input.Request.AlertName, Service: in.Input.Request.Service})
	if err != nil {
		return in, err
	}
	var activeAlerts queryActiveAlertsOutput
	if err := json.Unmarshal([]byte(alertOutput), &activeAlerts); err != nil {
		return in, fmt.Errorf("parse query_active_alerts output: %w", err)
	}
	in.Output.Alerts = activeAlerts.Alerts
	markPlanStep(in.Plan, "QueryActiveAlerts", stepStatusSuccess, "")
	in.Iterations = append(in.Iterations, domain.AgentIteration{Index: len(in.Iterations) + 1, Phase: "Executor", StepID: "alerts", Tool: "query_active_alerts", Observation: fmt.Sprintf("查询到 %d 条活跃告警", len(activeAlerts.Alerts)), StartedAt: time.Now(), EndedAt: time.Now()})

	if _, err := invokeAIOpsToolNode(ctx, toolsNode, "get_current_time", nil); err != nil {
		return in, err
	}
	markPlanStep(in.Plan, "GetCurrentTime", stepStatusSuccess, "")

	for _, alert := range activeAlerts.Alerts {
		query := strings.TrimSpace(alert.Name + " " + alert.Description)
		docsOutput, err := invokeAIOpsToolNode(ctx, toolsNode, "query_internal_docs", queryInternalDocsInput{Query: query})
		if err != nil {
			return in, err
		}
		var docs queryInternalDocsOutput
		if err := json.Unmarshal([]byte(docsOutput), &docs); err != nil {
			return in, fmt.Errorf("parse query_internal_docs output: %w", err)
		}
		in.Output.Citations = append(in.Output.Citations, docs.Results...)
		in.Output.Evidence = append(in.Output.Evidence, docs.Evidence...)
		markPlanStep(in.Plan, "QueryInternalDocs", stepStatusSuccess, "")

		from, to := alertTimeWindow(alert)
		keyword := alert.Service
		if strings.Contains(alert.Name, "服务下线") || strings.Contains(strings.ToLower(alert.Description), "panic") {
			keyword = "panic"
		}
		logsOutput, err := invokeAIOpsToolNode(ctx, toolsNode, "query_logs", queryLogsInput{Service: alert.Service, Keyword: keyword, Region: alert.Region, From: from.Format(time.RFC3339), To: to.Format(time.RFC3339), Limit: 20})
		if err != nil {
			return in, err
		}
		var logs queryLogsOutput
		if err := json.Unmarshal([]byte(logsOutput), &logs); err != nil {
			return in, fmt.Errorf("parse query_logs output: %w", err)
		}
		in.Output.Evidence = append(in.Output.Evidence, logs.Evidence)
		markPlanStep(in.Plan, "QueryLogs", stepStatusSuccess, "")

		metricsOutput, err := invokeAIOpsToolNode(ctx, toolsNode, "query_metrics", queryMetricsInput{Service: alert.Service, Metric: "restart_count,error_rate", Region: alert.Region, From: from.Format(time.RFC3339), To: to.Format(time.RFC3339)})
		if err != nil {
			return in, err
		}
		var metrics queryMetricsOutput
		if err := json.Unmarshal([]byte(metricsOutput), &metrics); err != nil {
			return in, fmt.Errorf("parse query_metrics output: %w", err)
		}
		in.Output.Evidence = append(in.Output.Evidence, metrics.Evidence)
		markPlanStep(in.Plan, "QueryMetrics", stepStatusSuccess, "")
	}
	in.Iterations = append(in.Iterations, domain.AgentIteration{Index: len(in.Iterations) + 1, Phase: "Executor", StepID: "evidence", Tool: "eino_tools_node", Observation: fmt.Sprintf("收集到 %d 条证据和 %d 条引用", len(in.Output.Evidence), len(in.Output.Citations)), StartedAt: time.Now(), EndedAt: time.Now()})
	return in, nil
}

func (r *EinoPlanExecuteReplanRunner) replan(ctx context.Context, in aiopsPERState) (aiopsPERState, error) {
	reason := "证据已满足报告生成条件"
	if len(in.Output.Alerts) == 0 {
		reason = "当前无活跃告警，跳过证据收集并生成无告警报告"
	}
	if len(in.Output.Citations) == 0 && len(in.Output.Alerts) > 0 {
		reason = "未检索到 SOP，继续使用日志和指标证据生成报告"
	}
	in.ReplanReason = reason
	in.Iterations = append(in.Iterations, domain.AgentIteration{Index: len(in.Iterations) + 1, Phase: "Replanner", StepID: "replan", Observation: reason, ReplanReason: reason, StartedAt: time.Now(), EndedAt: time.Now()})
	return in, nil
}

func (r *EinoPlanExecuteReplanRunner) report(ctx context.Context, in aiopsPERState) (aiopsPERState, error) {
	draft := buildAgentReport(in.Output.Alerts, in.Output.Citations, in.Output.Evidence)
	msg, err := in.Input.Model.Generate(ctx, []*schema.Message{schema.SystemMessage(in.Input.SystemPrompt), schema.UserMessage(draft)})
	if err != nil {
		return in, err
	}
	report := strings.TrimSpace(msg.Content)
	if report == "" {
		report = draft
	}
	in.Output.Report = report
	markPlanStep(in.Plan, "ReportGenerator", stepStatusSuccess, "")
	if in.Plan != nil {
		in.Plan.Status = stepStatusSuccess
		in.Plan.UpdatedAt = time.Now()
	}
	in.Iterations = append(in.Iterations, domain.AgentIteration{Index: len(in.Iterations) + 1, Phase: "ReportGenerator", StepID: "report", Tool: "chat_model", Observation: "生成结构化告警分析报告", StartedAt: time.Now(), EndedAt: time.Now()})
	return in, nil
}

type DeterministicAgentRunner struct {
	*EinoPlanExecuteReplanRunner
}

func NewDeterministicAgentRunner() *DeterministicAgentRunner {
	return &DeterministicAgentRunner{EinoPlanExecuteReplanRunner: NewEinoPlanExecuteReplanRunner()}
}

func invokableToolsAsBase(tools map[string]einotool.InvokableTool) []einotool.BaseTool {
	out := make([]einotool.BaseTool, 0, len(tools))
	for _, tool := range tools {
		out = append(out, tool)
	}
	return out
}

func invokeAIOpsToolNode(ctx context.Context, node *compose.ToolsNode, name string, input any) (string, error) {
	args := "{}"
	if input != nil {
		data, err := json.Marshal(input)
		if err != nil {
			return "", fmt.Errorf("marshal %s input: %w", name, err)
		}
		args = string(data)
	}
	msg := schema.AssistantMessage("", []schema.ToolCall{{
		ID:   "call-" + name,
		Type: "function",
		Function: schema.FunctionCall{
			Name:      name,
			Arguments: args,
		},
	}})
	outputs, err := node.Invoke(ctx, msg)
	if err != nil {
		return "", fmt.Errorf("%s failed: %w", name, err)
	}
	if len(outputs) == 0 {
		return "", fmt.Errorf("%s returned no tool message", name)
	}
	return outputs[0].Content, nil
}

func runAgentTool(ctx context.Context, tools map[string]einotool.InvokableTool, name string, input any) (string, error) {
	t, ok := tools[name]
	if !ok {
		return "", fmt.Errorf("agent tool %q is not configured", name)
	}
	args := "{}"
	if input != nil {
		data, err := json.Marshal(input)
		if err != nil {
			return "", fmt.Errorf("marshal %s input: %w", name, err)
		}
		args = string(data)
	}
	output, err := t.InvokableRun(ctx, args)
	if err != nil {
		return "", fmt.Errorf("%s failed: %w", name, err)
	}
	return output, nil
}

func alertTimeWindow(alert domain.Alert) (time.Time, time.Time) {
	if alert.StartsAt.IsZero() {
		now := time.Now()
		return now.Add(-1 * time.Hour), now
	}
	return alert.StartsAt.Add(-1 * time.Hour), alert.StartsAt.Add(time.Hour)
}

func buildAgentReport(alerts []domain.Alert, citations []domain.Citation, evidence []domain.Evidence) string {
	var b strings.Builder
	b.WriteString("告警分析报告\n\n")
	b.WriteString("一、活跃告警\n")
	if len(alerts) == 0 {
		b.WriteString("当前无活跃告警。\n\n")
	} else {
		for _, alert := range alerts {
			b.WriteString(fmt.Sprintf("- %s：服务=%s，级别=%s，状态=%s，地域=%s，描述=%s\n", alert.Name, alert.Service, alert.Severity, alert.Status, alert.Region, alert.Description))
		}
		b.WriteString("\n")
	}
	b.WriteString("二、SOP 匹配结果\n")
	if len(citations) == 0 {
		b.WriteString("未检索到匹配 SOP，证据不足时必须人工补充 SOP 或排查记录。\n\n")
	} else {
		for _, citation := range citations {
			b.WriteString(fmt.Sprintf("- %s：%s\n", citation.Source, summarizeText(citation.Content, 100)))
		}
		b.WriteString("\n")
	}
	b.WriteString("三、证据收集\n")
	if len(evidence) == 0 {
		b.WriteString("未收集到日志或指标证据，证据不足。\n\n")
	} else {
		for _, item := range evidence {
			b.WriteString(fmt.Sprintf("- [%s] %s：%s\n", item.Type, item.Source, item.Summary))
			for _, sample := range item.Samples {
				b.WriteString(fmt.Sprintf("  - %s\n", sample))
			}
		}
		b.WriteString("\n")
	}
	text := strings.ToLower(evidenceText(evidence))
	cause := "证据不足，暂无法判断明确根因。"
	conclusion := "当前证据不足，需要继续排查。"
	if strings.Contains(text, "panic") && strings.Contains(text, "restart_count") {
		cause = "日志存在 panic，且 restart_count 增加，根因倾向于应用 panic 触发 pod 重启。"
		conclusion = "应用 panic 导致服务实例重启，引发服务下线的可能性较高。"
	}
	b.WriteString("四、根因分析\n")
	b.WriteString(cause + "\n\n")
	b.WriteString("五、处理建议\n")
	b.WriteString("- 根据 SOP 和日志样本定位 panic 堆栈或异常代码路径。\n")
	b.WriteString("- 检查最近发布变更，人工评估是否需要回滚。\n")
	b.WriteString("- 持续观察 restart_count、error_rate 和服务实例健康状态。\n")
	b.WriteString("- 当前阶段只生成分析报告，不自动修复、不执行 SQL、不关闭告警。\n\n")
	b.WriteString("六、结论\n")
	b.WriteString(conclusion)
	return b.String()
}
