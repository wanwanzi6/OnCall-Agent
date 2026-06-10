package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	einotool "github.com/cloudwego/eino/components/tool"

	"oncall-agent/internal/infra/config"
	"oncall-agent/internal/infra/trace"
	"oncall-agent/internal/llm"
	"oncall-agent/internal/model/domain"
	"oncall-agent/internal/model/request"
	aiopstools "oncall-agent/internal/tools/aiops"
)

type EinoAgentAnalyzer struct {
	runner      AgentRunner
	model       llm.ChatModel
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
	Model        llm.ChatModel
	MaxSteps     int
}

type AgentRunOutput struct {
	Report    string
	Alerts    []domain.Alert
	Evidence  []domain.Evidence
	Citations []domain.Citation
}

func NewEinoAgentAnalyzerFromConfig(log *slog.Logger, alertProvider aiopstools.AlertProvider, logProvider aiopstools.LogProvider, metricProvider aiopstools.MetricProvider, knowledge *KnowledgeService, cfg config.Config) (*EinoAgentAnalyzer, error) {
	model, err := llm.NewChatModel(cfg.LLM)
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
	return NewEinoAgentAnalyzer(log, model, NewDeterministicAgentRunner(), alertProvider, logProvider, metricProvider, knowledge, cfg.AIOps.SOPTopK, toolTimeout, timeout, maxSteps), nil
}

func NewEinoAgentAnalyzer(log *slog.Logger, model llm.ChatModel, runner AgentRunner, alertProvider aiopstools.AlertProvider, logProvider aiopstools.LogProvider, metricProvider aiopstools.MetricProvider, knowledge *KnowledgeService, sopTopK int, toolTimeout, timeout time.Duration, maxSteps int) *EinoAgentAnalyzer {
	if log == nil {
		log = slog.Default()
	}
	if model == nil {
		model = llm.MockChatModel{}
	}
	if runner == nil {
		runner = NewDeterministicAgentRunner()
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
		TraceID:   traceID,
		Mode:      AnalyzerModeAgent,
		Report:    output.Report,
		Alerts:    output.Alerts,
		Steps:     tools.Recorder.Steps(),
		Evidence:  output.Evidence,
		Citations: output.Citations,
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

type DeterministicAgentRunner struct{}

func NewDeterministicAgentRunner() *DeterministicAgentRunner {
	return &DeterministicAgentRunner{}
}

func (r *DeterministicAgentRunner) Run(ctx context.Context, input AgentRunInput) (AgentRunOutput, error) {
	if input.Model == nil {
		return AgentRunOutput{}, fmt.Errorf("chat model is required")
	}
	tools := input.Tools
	if len(tools) == 0 {
		return AgentRunOutput{}, fmt.Errorf("agent tools are required")
	}
	if input.MaxSteps > 0 && input.MaxSteps < 4 {
		return AgentRunOutput{}, fmt.Errorf("agent max_steps %d is too small for required analysis flow", input.MaxSteps)
	}

	alertOutput, err := runAgentTool(ctx, tools, "query_active_alerts", queryActiveAlertsInput{
		AlertName: input.Request.AlertName,
		Service:   input.Request.Service,
	})
	if err != nil {
		return AgentRunOutput{}, err
	}
	var activeAlerts queryActiveAlertsOutput
	if err := json.Unmarshal([]byte(alertOutput), &activeAlerts); err != nil {
		return AgentRunOutput{}, fmt.Errorf("parse query_active_alerts output: %w", err)
	}

	if _, err := runAgentTool(ctx, tools, "get_current_time", nil); err != nil {
		return AgentRunOutput{}, err
	}

	output := AgentRunOutput{Alerts: activeAlerts.Alerts}
	for _, alert := range activeAlerts.Alerts {
		query := strings.TrimSpace(alert.Name + " " + alert.Description)
		docsOutput, err := runAgentTool(ctx, tools, "query_internal_docs", queryInternalDocsInput{Query: query})
		if err != nil {
			return AgentRunOutput{}, err
		}
		var docs queryInternalDocsOutput
		if err := json.Unmarshal([]byte(docsOutput), &docs); err != nil {
			return AgentRunOutput{}, fmt.Errorf("parse query_internal_docs output: %w", err)
		}
		output.Citations = append(output.Citations, docs.Results...)
		output.Evidence = append(output.Evidence, docs.Evidence...)

		from, to := alertTimeWindow(alert)
		keyword := alert.Service
		if strings.Contains(alert.Name, "服务下线") || strings.Contains(strings.ToLower(alert.Description), "panic") {
			keyword = "panic"
		}
		logsOutput, err := runAgentTool(ctx, tools, "query_logs", queryLogsInput{
			Service: alert.Service,
			Keyword: keyword,
			Region:  alert.Region,
			From:    from.Format(time.RFC3339),
			To:      to.Format(time.RFC3339),
			Limit:   20,
		})
		if err != nil {
			return AgentRunOutput{}, err
		}
		var logs queryLogsOutput
		if err := json.Unmarshal([]byte(logsOutput), &logs); err != nil {
			return AgentRunOutput{}, fmt.Errorf("parse query_logs output: %w", err)
		}
		output.Evidence = append(output.Evidence, logs.Evidence)

		metricsOutput, err := runAgentTool(ctx, tools, "query_metrics", queryMetricsInput{
			Service: alert.Service,
			Metric:  "restart_count,error_rate",
			Region:  alert.Region,
			From:    from.Format(time.RFC3339),
			To:      to.Format(time.RFC3339),
		})
		if err != nil {
			return AgentRunOutput{}, err
		}
		var metrics queryMetricsOutput
		if err := json.Unmarshal([]byte(metricsOutput), &metrics); err != nil {
			return AgentRunOutput{}, fmt.Errorf("parse query_metrics output: %w", err)
		}
		output.Evidence = append(output.Evidence, metrics.Evidence)
	}

	draft := buildAgentReport(output.Alerts, output.Citations, output.Evidence)
	report, err := input.Model.Generate(ctx, input.SystemPrompt, draft)
	if err != nil {
		return AgentRunOutput{}, err
	}
	if strings.TrimSpace(report) == "" {
		report = draft
	}
	output.Report = report
	return output, nil
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
