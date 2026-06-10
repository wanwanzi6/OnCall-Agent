# 简历描述

## 短版

基于 Go + Eino + Milvus + React 构建智能 Oncall 助手，支持 SOP 知识库 RAG 问答、告警自动分析、日志/指标证据收集、Eino Agent 工具调用和结构化故障报告生成。

## 亮点版

- 设计 RAG 知识库模块，支持文档上传、切片、向量检索和引用来源返回。
- 基于规则工作流和 Eino Agent 双模式实现告警分析，兼顾稳定性与智能化。
- 抽象告警、日志、指标 Provider，支持 mock 演示和真实数据源扩展。
- 构建前端控制台，展示 Agent 执行轨迹、证据链和分析报告。
- 完善 trace、错误处理、工具边界和测试体系，提升工程可靠性。

## STAR 描述

- 背景：Oncall 排障需要在 SOP、告警、日志、指标之间频繁切换，信息分散且依赖人工经验。
- 任务：从 0 重构一个可演示、可测试、可扩展的智能 Oncall 助手，默认不依赖外部服务。
- 行动：用 Go 分层实现 API、Service、RAG、Provider 和 Agent 边界；用 React 构建控制台；引入 mock/memory/rule 默认模式和 DashScope/Milvus/Agent 显式扩展模式；补齐 trace、错误处理、上传安全和测试。
- 结果：形成上传 SOP、RAG 问答、告警分析、证据收集、报告展示的完整闭环，默认 `go test ./...` 和前端 build 可运行，项目可用于演示和简历讲解。

## 面试讲解要点

- 背景：排障链路长，知识和证据分散，目标是提高一线定位效率。
- 架构：前端控制台 + Go API + KnowledgeService + AIOpsService + Provider + Agent Tool。
- 难点：既要接入 RAG/Agent，又要保证默认环境稳定、无外部依赖、测试可复现。
- 取舍：默认 rule workflow 保证确定性；Agent 作为显式增强，并支持 fallback。
- 结果：形成可启动、可测试、可部署、可演示的工程闭环。
- 后续优化：接入真实 Prometheus/日志平台、报告持久化、OpenTelemetry、前端自动化测试。
