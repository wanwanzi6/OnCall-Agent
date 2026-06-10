package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"oncall-agent/internal/infra/config"
	"oncall-agent/internal/infra/trace"
	"oncall-agent/internal/model/domain"
	aiopstools "oncall-agent/internal/tools/aiops"
)

const (
	stepStatusSuccess = "success"
	stepStatusFailed  = "failed"
	stepStatusSkipped = "skipped"
)

type AIOpsService struct {
	alertProvider  aiopstools.AlertProvider
	logProvider    aiopstools.LogProvider
	metricProvider aiopstools.MetricProvider
	knowledge      *KnowledgeService
	timeout        time.Duration
	sopTopK        int
	log            *slog.Logger
}

type evidencePlan struct {
	LogQuery    aiopstools.LogQuery
	MetricQuery aiopstools.MetricQuery
}

type rootCauseAnalysis struct {
	Cause           string
	Recommendations []string
	Conclusion      string
}

func NewAIOpsService(log *slog.Logger) *AIOpsService {
	cfg, err := config.Load("")
	if err != nil {
		cfg = &config.Config{
			AIOps: config.AIOpsConfig{
				AlertProvider:  aiopstools.ProviderMock,
				LogProvider:    aiopstools.ProviderMock,
				MetricProvider: aiopstools.ProviderMock,
				Timeout:        10 * time.Second,
				SOPTopK:        3,
			},
			Prometheus: config.PrometheusConfig{BaseURL: "http://localhost:9090", Timeout: 5 * time.Second},
		}
	}
	service, err := NewAIOpsServiceFromConfig(*cfg, log, nil)
	if err != nil {
		return NewAIOpsServiceWithProviders(
			log,
			aiopstools.NewMockAlertProvider(),
			aiopstools.NewMockLogProvider(),
			aiopstools.NewMockMetricProvider(),
			nil,
			cfg.AIOps,
		)
	}
	return service
}

func NewAIOpsServiceFromConfig(cfg config.Config, log *slog.Logger, knowledge *KnowledgeService) (*AIOpsService, error) {
	providers, err := aiopstools.NewProviders(cfg)
	if err != nil {
		return nil, err
	}
	return NewAIOpsServiceWithProviders(log, providers.Alert, providers.Log, providers.Metric, knowledge, cfg.AIOps), nil
}

func NewAIOpsServiceWithProviders(log *slog.Logger, alertProvider aiopstools.AlertProvider, logProvider aiopstools.LogProvider, metricProvider aiopstools.MetricProvider, knowledge *KnowledgeService, cfg config.AIOpsConfig) *AIOpsService {
	if log == nil {
		log = slog.Default()
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
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.SOPTopK <= 0 {
		cfg.SOPTopK = 3
	}
	return &AIOpsService{
		alertProvider:  alertProvider,
		logProvider:    logProvider,
		metricProvider: metricProvider,
		knowledge:      knowledge,
		timeout:        cfg.Timeout,
		sopTopK:        cfg.SOPTopK,
		log:            log,
	}
}

func (s *AIOpsService) Analyze(ctx context.Context, alertName, service string) (domain.AIOpsAnalyzeResult, error) {
	traceID := trace.FromContext(ctx)
	if traceID == "" {
		traceID = trace.NewID()
		ctx = trace.WithTraceID(ctx, traceID)
	}
	workflowCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	result := domain.AIOpsAnalyzeResult{TraceID: traceID}
	s.log.InfoContext(ctx, "aiops analyze requested",
		"trace_id", traceID,
		"service_name", service,
		"alert_name", alertName,
	)

	alerts, step := s.collectAlerts(workflowCtx, aiopstools.AlertFilter{AlertName: alertName, Service: service})
	result.Steps = append(result.Steps, step)
	result.Alerts = alerts
	if step.Status == stepStatusFailed {
		result.Steps = append(result.Steps,
			s.skippedStep(ctx, "SOPRetriever", "告警采集失败，跳过 SOP 检索", alerts, result.Evidence, step.Error),
			s.skippedStep(ctx, "EvidencePlanner", "告警采集失败，跳过证据规划", alerts, result.Evidence, step.Error),
			s.skippedStep(ctx, "EvidenceCollector", "告警采集失败，跳过证据收集", alerts, result.Evidence, step.Error),
			s.skippedStep(ctx, "RootCauseAnalyzer", "告警采集失败，跳过根因分析", alerts, result.Evidence, step.Error),
		)
		result.Report = s.generateReport(alerts, nil, result.Evidence, rootCauseAnalysis{
			Cause:      "告警采集失败，无法判断根因。",
			Conclusion: "请先检查告警 provider 配置和连通性。",
		})
		result.Steps = append(result.Steps, s.successStep(ctx, "ReportGenerator", "基于失败状态生成降级报告", alerts, result.Evidence))
		return result, nil
	}
	if len(alerts) == 0 {
		result.Steps = append(result.Steps,
			s.skippedStep(ctx, "SOPRetriever", "当前无活跃告警，跳过 SOP 检索", alerts, result.Evidence, ""),
			s.skippedStep(ctx, "EvidencePlanner", "当前无活跃告警，跳过证据规划", alerts, result.Evidence, ""),
			s.skippedStep(ctx, "EvidenceCollector", "当前无活跃告警，跳过证据收集", alerts, result.Evidence, ""),
			s.skippedStep(ctx, "RootCauseAnalyzer", "当前无活跃告警，跳过根因分析", alerts, result.Evidence, ""),
		)
		result.Report = "告警分析报告\n\n当前无活跃告警，无需进一步分析。"
		result.Steps = append(result.Steps, s.successStep(ctx, "ReportGenerator", "生成无活跃告警报告", alerts, result.Evidence))
		return result, nil
	}

	citations, sopEvidence, step := s.retrieveSOPs(workflowCtx, alerts)
	result.Citations = citations
	result.Evidence = append(result.Evidence, sopEvidence...)
	result.Steps = append(result.Steps, step)

	plans, step := s.planEvidence(ctx, alerts)
	result.Steps = append(result.Steps, step)

	collectedEvidence, step := s.collectEvidence(workflowCtx, plans)
	result.Evidence = append(result.Evidence, collectedEvidence...)
	result.Steps = append(result.Steps, step)

	analysis, step := s.analyzeRootCause(ctx, result.Evidence)
	result.Steps = append(result.Steps, step)

	result.Report = s.generateReport(alerts, citations, result.Evidence, analysis)
	result.Steps = append(result.Steps, s.successStep(ctx, "ReportGenerator", "生成结构化告警分析报告", alerts, result.Evidence))
	return result, nil
}

func (s *AIOpsService) collectAlerts(ctx context.Context, filter aiopstools.AlertFilter) ([]domain.Alert, domain.WorkflowStep) {
	start := time.Now()
	alerts, err := s.alertProvider.QueryActiveAlerts(ctx, filter)
	if err != nil {
		s.logProviderCallFailed(ctx, "mock_or_configured_alert", filter.AlertName+" "+filter.Service, err)
		step := s.step(ctx, "AlertCollector", stepStatusFailed, "查询活跃告警失败", err, start, time.Now(), nil, nil)
		return nil, step
	}
	s.logProviderCall(ctx, "mock_or_configured_alert", filter.AlertName+" "+filter.Service, len(alerts))
	summary := fmt.Sprintf("查询到 %d 条活跃告警", len(alerts))
	return alerts, s.step(ctx, "AlertCollector", stepStatusSuccess, summary, nil, start, time.Now(), alerts, nil)
}

func (s *AIOpsService) retrieveSOPs(ctx context.Context, alerts []domain.Alert) ([]domain.Citation, []domain.Evidence, domain.WorkflowStep) {
	start := time.Now()
	if s.knowledge == nil {
		return nil, nil, s.step(ctx, "SOPRetriever", stepStatusSkipped, "KnowledgeService 未配置，跳过 SOP 检索", nil, start, time.Now(), alerts, nil)
	}

	citations := make([]domain.Citation, 0)
	evidence := make([]domain.Evidence, 0)
	for _, alert := range alerts {
		query := strings.TrimSpace(alert.Name + " " + alert.Description)
		results, err := s.knowledge.Search(ctx, query, s.sopTopK)
		if err != nil {
			return citations, evidence, s.step(ctx, "SOPRetriever", stepStatusFailed, "SOP 检索失败", err, start, time.Now(), alerts, evidence)
		}
		for _, result := range results {
			citation := domain.Citation{
				DocumentID: result.Chunk.DocumentID,
				ChunkID:    result.Chunk.ID,
				Source:     firstNonEmptyString(result.Source, result.TitlePath, result.Chunk.DocumentID),
				Score:      result.Score,
				Content:    result.Chunk.Content,
			}
			citations = append(citations, citation)
			evidence = append(evidence, domain.Evidence{
				ID:        fmt.Sprintf("sop-%d", len(evidence)+1),
				Type:      aiopstools.EvidenceTypeSOP,
				Source:    citation.Source,
				Query:     query,
				Summary:   summarizeText(citation.Content, 120),
				Samples:   []string{citation.Content},
				Metadata:  map[string]string{"document_id": citation.DocumentID, "chunk_id": citation.ChunkID},
				CreatedAt: time.Now(),
			})
		}
	}
	summary := fmt.Sprintf("检索到 %d 条 SOP 片段", len(citations))
	return citations, evidence, s.step(ctx, "SOPRetriever", stepStatusSuccess, summary, nil, start, time.Now(), alerts, evidence)
}

func (s *AIOpsService) planEvidence(ctx context.Context, alerts []domain.Alert) ([]evidencePlan, domain.WorkflowStep) {
	start := time.Now()
	plans := make([]evidencePlan, 0, len(alerts))
	for _, alert := range alerts {
		keyword := alert.Service
		if strings.Contains(alert.Name, "服务下线") || strings.Contains(alert.Description, "panic") {
			keyword = "panic"
		}
		from := alert.StartsAt.Add(-1 * time.Hour)
		to := alert.StartsAt.Add(time.Hour)
		if alert.StartsAt.IsZero() {
			now := time.Now()
			from = now.Add(-1 * time.Hour)
			to = now
		}
		plans = append(plans, evidencePlan{
			LogQuery: aiopstools.LogQuery{
				Service: alert.Service,
				Keyword: keyword,
				Region:  alert.Region,
				From:    from,
				To:      to,
				Limit:   20,
			},
			MetricQuery: aiopstools.MetricQuery{
				Service: alert.Service,
				Metric:  "restart_count,error_rate",
				Region:  alert.Region,
				From:    from,
				To:      to,
			},
		})
	}
	summary := fmt.Sprintf("生成 %d 组日志和指标查询计划", len(plans))
	return plans, s.step(ctx, "EvidencePlanner", stepStatusSuccess, summary, nil, start, time.Now(), alerts, nil)
}

func (s *AIOpsService) collectEvidence(ctx context.Context, plans []evidencePlan) ([]domain.Evidence, domain.WorkflowStep) {
	start := time.Now()
	evidence := make([]domain.Evidence, 0, len(plans)*2)
	errorTexts := make([]string, 0)
	for i, plan := range plans {
		logs, err := s.logProvider.QueryLogs(ctx, plan.LogQuery)
		if err != nil {
			s.logProviderCallFailed(ctx, "mock_or_configured_log", plan.LogQuery.Keyword+" "+plan.LogQuery.Service, err)
			errorTexts = append(errorTexts, "log provider: "+err.Error())
			evidence = append(evidence, providerErrorEvidence(fmt.Sprintf("log-%d-error", i+1), aiopstools.EvidenceTypeLog, "mock_or_configured_log", err))
		} else {
			s.logProviderCall(ctx, "mock_or_configured_log", plan.LogQuery.Keyword+" "+plan.LogQuery.Service, len(logs))
			evidence = append(evidence, logsToEvidence(fmt.Sprintf("log-%d", i+1), plan.LogQuery, logs))
		}

		metrics, err := s.metricProvider.QueryMetrics(ctx, plan.MetricQuery)
		if err != nil {
			s.logProviderCallFailed(ctx, "mock_or_configured_metric", plan.MetricQuery.Metric+" "+plan.MetricQuery.Service, err)
			errorTexts = append(errorTexts, "metric provider: "+err.Error())
			evidence = append(evidence, providerErrorEvidence(fmt.Sprintf("metric-%d-error", i+1), aiopstools.EvidenceTypeMetric, "mock_or_configured_metric", err))
		} else {
			s.logProviderCall(ctx, "mock_or_configured_metric", plan.MetricQuery.Metric+" "+plan.MetricQuery.Service, len(metrics))
			evidence = append(evidence, metricsToEvidence(fmt.Sprintf("metric-%d", i+1), plan.MetricQuery, metrics))
		}
	}
	status := stepStatusSuccess
	var err error
	summary := fmt.Sprintf("收集到 %d 条证据", len(evidence))
	if len(errorTexts) > 0 {
		status = stepStatusFailed
		err = errors.New(strings.Join(errorTexts, "; "))
		summary = "部分 provider 采集失败，已保留错误证据并继续分析"
	}
	return evidence, s.step(ctx, "EvidenceCollector", status, summary, err, start, time.Now(), nil, evidence)
}

func (s *AIOpsService) analyzeRootCause(ctx context.Context, evidence []domain.Evidence) (rootCauseAnalysis, domain.WorkflowStep) {
	start := time.Now()
	text := strings.ToLower(evidenceText(evidence))
	analysis := rootCauseAnalysis{
		Cause: "证据不足，暂无法判断明确根因。",
		Recommendations: []string{
			"补充最近 1 小时应用日志、容器重启事件和发布记录。",
			"人工确认告警影响范围后再执行处置。",
		},
		Conclusion: "当前证据不足，需要继续排查。",
	}
	if strings.Contains(text, "panic") {
		analysis.Cause = "应用 panic 导致 pod 重启，引发服务实例下线。"
		analysis.Recommendations = []string{
			"根据 panic 堆栈定位空指针或异常代码路径。",
			"检查最近发布变更并评估是否需要回滚。",
			"持续观察 restart_count、error_rate 和服务实例健康状态。",
		}
		analysis.Conclusion = "本次告警更可能由应用 panic 导致服务实例重启，引发服务下线。"
	}
	if strings.Contains(text, "restart_count") && strings.Contains(text, "panic") {
		analysis.Cause = "日志存在 panic，且 restart_count 增加，根因倾向于应用 panic 触发 pod 重启。"
		analysis.Conclusion = "应用 panic 导致服务实例重启，引发服务下线的可能性较高。"
	}
	return analysis, s.step(ctx, "RootCauseAnalyzer", stepStatusSuccess, analysis.Cause, nil, start, time.Now(), nil, evidence)
}

func (s *AIOpsService) generateReport(alerts []domain.Alert, citations []domain.Citation, evidence []domain.Evidence, analysis rootCauseAnalysis) string {
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
		b.WriteString("未检索到匹配 SOP，报告将基于告警和 mock 证据生成。\n\n")
	} else {
		for _, citation := range citations {
			b.WriteString(fmt.Sprintf("- %s：%s\n", citation.Source, summarizeText(citation.Content, 100)))
		}
		b.WriteString("\n")
	}

	b.WriteString("三、证据收集\n")
	if len(evidence) == 0 {
		b.WriteString("未收集到日志或指标证据。\n\n")
	} else {
		for _, item := range evidence {
			b.WriteString(fmt.Sprintf("- [%s] %s：%s\n", item.Type, item.Source, item.Summary))
			for _, sample := range item.Samples {
				b.WriteString(fmt.Sprintf("  - %s\n", sample))
			}
		}
		b.WriteString("\n")
	}

	b.WriteString("四、根因分析\n")
	b.WriteString(analysis.Cause + "\n\n")

	b.WriteString("五、处理建议\n")
	for _, recommendation := range analysis.Recommendations {
		b.WriteString("- " + recommendation + "\n")
	}
	b.WriteString("- 当前阶段只生成分析报告，不自动修复、不执行 SQL、不关闭告警。\n\n")

	b.WriteString("六、结论\n")
	b.WriteString(analysis.Conclusion)
	return b.String()
}

func (s *AIOpsService) skippedStep(ctx context.Context, name, summary string, alerts []domain.Alert, evidence []domain.Evidence, errText string) domain.WorkflowStep {
	start := time.Now()
	var err error
	if errText != "" {
		err = errors.New(errText)
	}
	return s.step(ctx, name, stepStatusSkipped, summary, err, start, time.Now(), alerts, evidence)
}

func (s *AIOpsService) successStep(ctx context.Context, name, summary string, alerts []domain.Alert, evidence []domain.Evidence) domain.WorkflowStep {
	start := time.Now()
	return s.step(ctx, name, stepStatusSuccess, summary, nil, start, time.Now(), alerts, evidence)
}

func (s *AIOpsService) step(ctx context.Context, name, status, summary string, err error, startedAt, endedAt time.Time, alerts []domain.Alert, evidence []domain.Evidence) domain.WorkflowStep {
	step := domain.WorkflowStep{
		Name:      name,
		Status:    status,
		Summary:   summary,
		StartedAt: startedAt,
		EndedAt:   endedAt,
		TraceID:   trace.FromContext(ctx),
	}
	if err != nil {
		step.Error = err.Error()
	}
	s.log.InfoContext(ctx, "aiops workflow step completed",
		"trace_id", step.TraceID,
		"step_name", name,
		"status", status,
		"alert_count", len(alerts),
		"evidence_count", len(evidence),
		"error", step.Error,
	)
	return step
}

func (s *AIOpsService) logProviderCall(ctx context.Context, providerName, querySummary string, resultCount int) {
	s.log.InfoContext(ctx, "aiops provider call completed",
		"trace_id", trace.FromContext(ctx),
		"provider_name", providerName,
		"query_summary", summarizeText(querySummary, 120),
		"result_count", resultCount,
	)
}

func (s *AIOpsService) logProviderCallFailed(ctx context.Context, providerName, querySummary string, err error) {
	s.log.ErrorContext(ctx, "aiops provider call failed",
		"trace_id", trace.FromContext(ctx),
		"provider_name", providerName,
		"query_summary", summarizeText(querySummary, 120),
		"result_count", 0,
		"error", err.Error(),
	)
}

func logsToEvidence(id string, query aiopstools.LogQuery, logs []aiopstools.LogEntry) domain.Evidence {
	samples := make([]string, 0, len(logs))
	for _, entry := range logs {
		samples = append(samples, fmt.Sprintf("%s %s %s", entry.Timestamp.Format(time.RFC3339), entry.Level, entry.Message))
	}
	return domain.Evidence{
		ID:        id,
		Type:      aiopstools.EvidenceTypeLog,
		Source:    "mock_log",
		Query:     fmt.Sprintf("service=%s keyword=%s region=%s", query.Service, query.Keyword, query.Region),
		Summary:   fmt.Sprintf("查询到 %d 条日志，关键字=%s", len(logs), query.Keyword),
		Samples:   samples,
		Metadata:  map[string]string{"service": query.Service, "region": query.Region},
		CreatedAt: time.Now(),
		Logs:      samples,
	}
}

func metricsToEvidence(id string, query aiopstools.MetricQuery, points []aiopstools.MetricPoint) domain.Evidence {
	samples := make([]string, 0, len(points))
	for _, point := range points {
		samples = append(samples, fmt.Sprintf("%s %s=%.4g", point.Timestamp.Format(time.RFC3339), point.Metric, point.Value))
	}
	return domain.Evidence{
		ID:        id,
		Type:      aiopstools.EvidenceTypeMetric,
		Source:    "mock_metric",
		Query:     fmt.Sprintf("service=%s metric=%s region=%s", query.Service, query.Metric, query.Region),
		Summary:   fmt.Sprintf("查询到 %d 个指标点", len(points)),
		Samples:   samples,
		Metadata:  map[string]string{"service": query.Service, "region": query.Region},
		CreatedAt: time.Now(),
	}
}

func providerErrorEvidence(id, evidenceType, source string, err error) domain.Evidence {
	return domain.Evidence{
		ID:        id,
		Type:      evidenceType,
		Source:    source,
		Summary:   "provider 调用失败：" + err.Error(),
		Metadata:  map[string]string{"error": err.Error()},
		CreatedAt: time.Now(),
	}
}

func evidenceText(evidence []domain.Evidence) string {
	var b strings.Builder
	for _, item := range evidence {
		b.WriteString(item.Summary)
		b.WriteByte('\n')
		for _, sample := range item.Samples {
			b.WriteString(sample)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func summarizeText(text string, maxLen int) string {
	text = strings.Join(strings.Fields(text), " ")
	if maxLen <= 0 || len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
