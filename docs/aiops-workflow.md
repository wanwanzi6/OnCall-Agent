# AI Ops 工作流

AI Ops 支持 `rule` 和 `agent` 两种模式。默认使用 `rule`，保证测试和 demo 稳定；`agent` 需要显式开启，并支持失败 fallback 到 rule workflow。

## Rule Workflow

```mermaid
flowchart LR
  A["AlertCollector"] --> S["SOPRetriever"]
  S --> P["EvidencePlanner"]
  P --> E["EvidenceCollector"]
  E --> R["RootCauseAnalyzer"]
  R --> G["ReportGenerator"]
```

步骤说明：

- AlertCollector：查询活跃告警，无告警时生成明确报告并跳过后续分析。
- SOPRetriever：调用 KnowledgeService 检索 SOP，生成 citations 和 SOP evidence。
- EvidencePlanner：按告警生成日志和指标查询计划。
- EvidenceCollector：调用 LogProvider、MetricProvider，保留 evidence。
- RootCauseAnalyzer：规则判断 panic、restart_count 等证据。
- ReportGenerator：输出结构化报告。

## Agent Workflow

Agent 模式通过工具封装复用已有边界：

- `query_active_alerts`
- `query_internal_docs`
- `query_logs`
- `query_metrics`
- `get_current_time`

工具只读，不包含自动修复、SQL 执行、系统命令执行或关闭告警能力。

## Fallback

```text
mode=agent
  -> agent 成功：返回 agent 结果
  -> agent 失败且 fallback_to_rule=true：插入 AgentAnalyzer failed step，返回 rule 结果
  -> agent 失败且 fallback_to_rule=false：返回标准错误响应
```

## 数据流

AI Ops 响应统一返回：

- `alerts`
- `steps`
- `evidence`
- `citations`
- `report`
- `trace_id`
- `mode`
- `fallback_used`
