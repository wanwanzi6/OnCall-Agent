# 评测说明

本项目的效果指标通过本地 demo 评测脚本生成，默认不依赖外部 LLM、DashScope、Milvus、Prometheus 或网络。

运行命令：

```bash
go run ./scripts/evaluate_demo.go
```

当前评测对象：

- 知识库文档：`docs/demo/告警处理手册.md`
- RAG 评测集：8 条自建排障 QA query
- AI Ops 场景：`alert_name=服务下线`，`service=billing-service`
- 默认配置：`mock + memory + rule`

## RAG 指标

- Recall@1：Top 1 结果包含预期 SOP 片段的 query 占比。
- Recall@3：Top 3 结果中至少 1 条包含预期 SOP 片段的 query 占比。
- MRR@3：预期 SOP 片段在 Top 3 中首次命中的倒数排名均值。

当前结果：

```text
RAG Recall@1: 50.0% (4/8)
RAG Recall@3: 100.0% (8/8)
RAG MRR@3: 0.708
```

## AI Ops 指标

- 工作流成功率：AlertCollector、SOPRetriever、EvidencePlanner、EvidenceCollector、RootCauseAnalyzer、ReportGenerator 的成功步骤占比。
- 证据数：一次告警分析收集到的 SOP、日志、指标 evidence 数量。
- citation 数：报告中引用的 SOP 片段数量。
- 根因信号覆盖：报告是否包含 `panic`、`restart_count`、`服务下线` 等预期根因信号。

当前结果：

```text
AIOps workflow success steps: 6/6
AIOps evidence count: 5
AIOps citation count: 3
AIOps report contains expected root-cause signals: true
```

## 简历使用建议

可以写：

```text
在 8 条自建排障 QA demo 评测集上，RAG Recall@3 达到 100%，MRR@3 为 0.708；在服务下线 demo 场景中，AI Ops 工作流 6/6 步骤成功执行，自动收集 5 条证据和 3 条 SOP citation，并生成包含 panic、restart_count 等根因信号的结构化报告。
```

不建议写：

```text
RAG 准确率 100%，故障定位准确率 100%。
```

原因是当前评测集规模较小，且默认使用 mock embedding 和 mock provider，更适合作为 demo 可复现指标，而不是生产准确率。
