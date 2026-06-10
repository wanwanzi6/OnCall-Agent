package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"oncall-agent/internal/infra/trace"
	"oncall-agent/internal/model/domain"
	aiopstools "oncall-agent/internal/tools/aiops"
)

type ToolCallRecorder struct {
	mu      sync.Mutex
	records []domain.WorkflowStep
}

func NewToolCallRecorder() *ToolCallRecorder {
	return &ToolCallRecorder{}
}

func (r *ToolCallRecorder) Record(step domain.WorkflowStep) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, step)
}

func (r *ToolCallRecorder) Steps() []domain.WorkflowStep {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	steps := make([]domain.WorkflowStep, len(r.records))
	copy(steps, r.records)
	return steps
}

type AIOpsAgentTools struct {
	ActiveAlerts einotool.InvokableTool
	InternalDocs einotool.InvokableTool
	Logs         einotool.InvokableTool
	Metrics      einotool.InvokableTool
	CurrentTime  einotool.InvokableTool
	Recorder     *ToolCallRecorder
}

func NewAIOpsAgentTools(log *slog.Logger, alertProvider aiopstools.AlertProvider, logProvider aiopstools.LogProvider, metricProvider aiopstools.MetricProvider, knowledge *KnowledgeService, sopTopK int, timeout time.Duration) *AIOpsAgentTools {
	if log == nil {
		log = slog.Default()
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	if sopTopK <= 0 {
		sopTopK = 3
	}
	recorder := NewToolCallRecorder()
	return &AIOpsAgentTools{
		ActiveAlerts: newJSONAgentTool(log, recorder, "query_active_alerts", "查询当前活跃告警，只读取告警状态，不关闭或修改告警。", timeout, activeAlertParams(), func(ctx context.Context, args string) (string, int, error) {
			var input queryActiveAlertsInput
			if err := json.Unmarshal([]byte(defaultJSONObject(args)), &input); err != nil {
				return "", 0, fmt.Errorf("parse query_active_alerts input: %w", err)
			}
			alerts, err := alertProvider.QueryActiveAlerts(ctx, aiopstools.AlertFilter{
				AlertName: input.AlertName,
				Service:   input.Service,
				Region:    input.Region,
				Labels:    input.Labels,
			})
			if err != nil {
				return "", 0, err
			}
			return marshalToolOutput(queryActiveAlertsOutput{Alerts: alerts}), len(alerts), nil
		}),
		InternalDocs: newJSONAgentTool(log, recorder, "query_internal_docs", "基于告警信息检索内部 SOP 和知识库文档。", timeout, internalDocsParams(), func(ctx context.Context, args string) (string, int, error) {
			var input queryInternalDocsInput
			if err := json.Unmarshal([]byte(defaultJSONObject(args)), &input); err != nil {
				return "", 0, fmt.Errorf("parse query_internal_docs input: %w", err)
			}
			if strings.TrimSpace(input.Query) == "" {
				return "", 0, fmt.Errorf("query is required")
			}
			topK := input.TopK
			if topK <= 0 {
				topK = sopTopK
			}
			if knowledge == nil {
				return marshalToolOutput(queryInternalDocsOutput{Results: nil, Evidence: nil}), 0, nil
			}
			results, err := knowledge.Search(ctx, input.Query, topK)
			if err != nil {
				return "", 0, err
			}
			output := queryInternalDocsOutput{Results: make([]domain.Citation, 0, len(results)), Evidence: make([]domain.Evidence, 0, len(results))}
			for i, result := range results {
				citation := domain.Citation{
					DocumentID: result.Chunk.DocumentID,
					ChunkID:    result.Chunk.ID,
					Source:     firstNonEmptyString(result.Source, result.TitlePath, result.Chunk.DocumentID),
					Score:      result.Score,
					Content:    result.Chunk.Content,
				}
				output.Results = append(output.Results, citation)
				output.Evidence = append(output.Evidence, domain.Evidence{
					ID:        fmt.Sprintf("agent-sop-%d", i+1),
					Type:      aiopstools.EvidenceTypeSOP,
					Source:    citation.Source,
					Query:     input.Query,
					Summary:   summarizeText(citation.Content, 120),
					Samples:   []string{citation.Content},
					Metadata:  map[string]string{"document_id": citation.DocumentID, "chunk_id": citation.ChunkID},
					CreatedAt: time.Now(),
				})
			}
			return marshalToolOutput(output), len(output.Results), nil
		}),
		Logs: newJSONAgentTool(log, recorder, "query_logs", "按服务、关键字、地域和时间范围查询日志证据。", timeout, logsParams(), func(ctx context.Context, args string) (string, int, error) {
			var input queryLogsInput
			if err := json.Unmarshal([]byte(defaultJSONObject(args)), &input); err != nil {
				return "", 0, fmt.Errorf("parse query_logs input: %w", err)
			}
			query, err := input.toQuery()
			if err != nil {
				return "", 0, err
			}
			logs, err := logProvider.QueryLogs(ctx, query)
			if err != nil {
				return "", 0, err
			}
			evidence := logsToEvidence("agent-log-1", query, logs)
			return marshalToolOutput(queryLogsOutput{Logs: logs, Evidence: evidence}), len(logs), nil
		}),
		Metrics: newJSONAgentTool(log, recorder, "query_metrics", "按服务、指标、地域和时间范围查询指标证据。", timeout, metricsParams(), func(ctx context.Context, args string) (string, int, error) {
			var input queryMetricsInput
			if err := json.Unmarshal([]byte(defaultJSONObject(args)), &input); err != nil {
				return "", 0, fmt.Errorf("parse query_metrics input: %w", err)
			}
			query, err := input.toQuery()
			if err != nil {
				return "", 0, err
			}
			points, err := metricProvider.QueryMetrics(ctx, query)
			if err != nil {
				return "", 0, err
			}
			evidence := metricsToEvidence("agent-metric-1", query, points)
			return marshalToolOutput(queryMetricsOutput{Metrics: points, Evidence: evidence}), len(points), nil
		}),
		CurrentTime: newJSONAgentTool(log, recorder, "get_current_time", "获取当前时间，用于没有告警触发时间时确定查询窗口。", timeout, nil, func(ctx context.Context, args string) (string, int, error) {
			now := time.Now()
			return marshalToolOutput(currentTimeOutput{Now: now, RFC3339: now.Format(time.RFC3339)}), 1, nil
		}),
		Recorder: recorder,
	}
}

func (t *AIOpsAgentTools) List() []einotool.InvokableTool {
	if t == nil {
		return nil
	}
	return []einotool.InvokableTool{t.ActiveAlerts, t.InternalDocs, t.Logs, t.Metrics, t.CurrentTime}
}

func (t *AIOpsAgentTools) ByName() map[string]einotool.InvokableTool {
	tools := make(map[string]einotool.InvokableTool)
	for _, item := range t.List() {
		info, err := item.Info(context.Background())
		if err == nil && info != nil {
			tools[info.Name] = item
		}
	}
	return tools
}

type jsonAgentTool struct {
	log      *slog.Logger
	recorder *ToolCallRecorder
	info     *schema.ToolInfo
	timeout  time.Duration
	run      func(ctx context.Context, args string) (output string, resultCount int, err error)
}

func newJSONAgentTool(log *slog.Logger, recorder *ToolCallRecorder, name, desc string, timeout time.Duration, params map[string]*schema.ParameterInfo, run func(context.Context, string) (string, int, error)) *jsonAgentTool {
	info := &schema.ToolInfo{Name: name, Desc: desc}
	if len(params) > 0 {
		info.ParamsOneOf = schema.NewParamsOneOfByParams(params)
	}
	return &jsonAgentTool{log: log, recorder: recorder, info: info, timeout: timeout, run: run}
}

func (t *jsonAgentTool) Info(context.Context) (*schema.ToolInfo, error) {
	return t.info, nil
}

func (t *jsonAgentTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...einotool.Option) (string, error) {
	start := time.Now()
	toolCtx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	output, count, err := t.run(toolCtx, argumentsInJSON)
	status := stepStatusSuccess
	summary := fmt.Sprintf("%s 返回 %d 条结果", t.info.Name, count)
	errText := ""
	if err != nil {
		status = stepStatusFailed
		summary = t.info.Name + " 调用失败"
		errText = err.Error()
	}
	step := domain.WorkflowStep{
		Name:      "AgentTool:" + t.info.Name,
		Tool:      t.info.Name,
		Status:    status,
		Summary:   summary,
		Error:     errText,
		StartedAt: start,
		EndedAt:   time.Now(),
		TraceID:   trace.FromContext(ctx),
	}
	t.recorder.Record(step)
	t.log.InfoContext(ctx, "aiops agent tool completed",
		"trace_id", step.TraceID,
		"tool_name", t.info.Name,
		"status", status,
		"summary", summary,
		"error", errText,
	)
	if err != nil {
		return "", err
	}
	return output, nil
}

type queryActiveAlertsInput struct {
	AlertName string            `json:"alert_name,omitempty"`
	Service   string            `json:"service,omitempty"`
	Region    string            `json:"region,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
}

type queryActiveAlertsOutput struct {
	Alerts []domain.Alert `json:"alerts"`
}

type queryInternalDocsInput struct {
	Query string `json:"query"`
	TopK  int    `json:"top_k,omitempty"`
}

type queryInternalDocsOutput struct {
	Results  []domain.Citation `json:"results"`
	Evidence []domain.Evidence `json:"evidence"`
}

type queryLogsInput struct {
	Service string `json:"service"`
	Keyword string `json:"keyword,omitempty"`
	Region  string `json:"region,omitempty"`
	From    string `json:"from,omitempty"`
	To      string `json:"to,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

func (i queryLogsInput) toQuery() (aiopstools.LogQuery, error) {
	from, to, err := parseTimeRange(i.From, i.To)
	if err != nil {
		return aiopstools.LogQuery{}, err
	}
	if i.Limit <= 0 {
		i.Limit = 20
	}
	return aiopstools.LogQuery{Service: i.Service, Keyword: i.Keyword, Region: i.Region, From: from, To: to, Limit: i.Limit}, nil
}

type queryLogsOutput struct {
	Logs     []aiopstools.LogEntry `json:"logs"`
	Evidence domain.Evidence       `json:"evidence"`
}

type queryMetricsInput struct {
	Service string `json:"service"`
	Metric  string `json:"metric"`
	Region  string `json:"region,omitempty"`
	From    string `json:"from,omitempty"`
	To      string `json:"to,omitempty"`
}

func (i queryMetricsInput) toQuery() (aiopstools.MetricQuery, error) {
	from, to, err := parseTimeRange(i.From, i.To)
	if err != nil {
		return aiopstools.MetricQuery{}, err
	}
	return aiopstools.MetricQuery{Service: i.Service, Metric: i.Metric, Region: i.Region, From: from, To: to}, nil
}

type queryMetricsOutput struct {
	Metrics  []aiopstools.MetricPoint `json:"metrics"`
	Evidence domain.Evidence          `json:"evidence"`
}

type currentTimeOutput struct {
	Now     time.Time `json:"now"`
	RFC3339 string    `json:"rfc3339"`
}

func activeAlertParams() map[string]*schema.ParameterInfo {
	return map[string]*schema.ParameterInfo{
		"alert_name": {Type: schema.String, Desc: "告警名称，可选"},
		"service":    {Type: schema.String, Desc: "服务名，可选"},
		"region":     {Type: schema.String, Desc: "地域，可选"},
	}
}

func internalDocsParams() map[string]*schema.ParameterInfo {
	return map[string]*schema.ParameterInfo{
		"query": {Type: schema.String, Desc: "用于检索 SOP 的查询文本", Required: true},
		"top_k": {Type: schema.Integer, Desc: "返回结果数量，可选"},
	}
}

func logsParams() map[string]*schema.ParameterInfo {
	return map[string]*schema.ParameterInfo{
		"service": {Type: schema.String, Desc: "服务名", Required: true},
		"keyword": {Type: schema.String, Desc: "日志关键字"},
		"region":  {Type: schema.String, Desc: "地域"},
		"from":    {Type: schema.String, Desc: "起始时间 RFC3339"},
		"to":      {Type: schema.String, Desc: "结束时间 RFC3339"},
		"limit":   {Type: schema.Integer, Desc: "最大日志条数"},
	}
}

func metricsParams() map[string]*schema.ParameterInfo {
	return map[string]*schema.ParameterInfo{
		"service": {Type: schema.String, Desc: "服务名", Required: true},
		"metric":  {Type: schema.String, Desc: "指标名，多个指标可用逗号分隔", Required: true},
		"region":  {Type: schema.String, Desc: "地域"},
		"from":    {Type: schema.String, Desc: "起始时间 RFC3339"},
		"to":      {Type: schema.String, Desc: "结束时间 RFC3339"},
	}
}

func parseTimeRange(fromText, toText string) (time.Time, time.Time, error) {
	var from time.Time
	var to time.Time
	var err error
	if strings.TrimSpace(fromText) != "" {
		from, err = time.Parse(time.RFC3339, fromText)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("parse from: %w", err)
		}
	}
	if strings.TrimSpace(toText) != "" {
		to, err = time.Parse(time.RFC3339, toText)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("parse to: %w", err)
		}
	}
	if from.IsZero() || to.IsZero() {
		now := time.Now()
		if to.IsZero() {
			to = now
		}
		if from.IsZero() {
			from = to.Add(-1 * time.Hour)
		}
	}
	return from, to, nil
}

func marshalToolOutput(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return `{"error":"marshal tool output failed"}`
	}
	return string(data)
}

func defaultJSONObject(text string) string {
	if strings.TrimSpace(text) == "" {
		return "{}"
	}
	return text
}
