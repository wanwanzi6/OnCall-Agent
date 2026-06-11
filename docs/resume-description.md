# 简历描述

## 短版

基于 Go + Eino + Milvus + React 构建智能 Oncall 助手，支持 SOP 知识库 RAG 问答、告警自动分析、日志/指标证据收集、Eino Agent 工具调用和结构化故障报告生成。

## 量化简历版

智能 Oncall/AIOps 故障分析助手 | Go, Gin, React, TypeScript, RAG, Eino, Milvus

- 从 0 重构智能 Oncall 助手，围绕“告警理解 -> SOP 检索 -> 日志/指标取证 -> 根因分析 -> 报告生成”构建 6 步排障闭环。
- 设计 SOP 知识库 RAG 链路，支持文档上传、切片、Embedding、向量检索和 citation 返回；在 8 条自建排障 QA demo 评测集上，RAG Recall@3 达到 100%，MRR@3 为 0.708。
- 实现 rule workflow 与 Agent workflow 双模式告警分析，默认使用确定性 rule 链路保证演示稳定性，Agent 失败时支持 fallback 到 rule workflow。
- 封装查询活跃告警、检索内部文档、查询日志、查询指标、获取当前时间 5 个只读 Agent 工具，并限制自动修复、命令执行、关闭告警等高风险操作。
- 抽象告警、日志、指标 3 类 Provider，默认使用 mock provider 保证离线 demo 可复现，并预留 Prometheus 等真实监控系统接入能力。
- 在 demo 服务下线场景中，AI Ops 工作流 6/6 步骤成功执行，自动收集 5 条证据、3 条 SOP citation，并生成包含 panic、restart_count 和服务下线根因信号的结构化报告。
- 补齐 61 个核心测试用例，覆盖 RAG、配置解析、上传校验、API 响应、Agent 工具、fallback 等关键路径；`go test ./...` 和前端生产构建均可通过。

## 效果指标口径

当前效果数字来自 `scripts/evaluate_demo.go` 的本地 demo 评测：

- RAG Recall@1：50.0%（4/8）
- RAG Recall@3：100.0%（8/8）
- RAG MRR@3：0.708
- AI Ops 工作流成功率：6/6
- AI Ops 证据数：5
- AI Ops citation 数：3

这些数字适合在简历中写成“自建 demo 评测集”或“排障 demo 场景”，不建议写成生产准确率。

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
