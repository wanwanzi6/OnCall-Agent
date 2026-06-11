# 项目经历

## 智能 Oncall 助手（AIOps 故障分析系统）

**技术栈**：Go 1.23+ / Gin / Eino（字节跳动 LLM 编排框架）/ Milvus / DashScope / RAG / React 18 + TypeScript / Docker

### 项目简介

从零构建面向 SRE 和后端研发的智能故障排查辅助系统，围绕"告警采集 → SOP 知识检索 → 日志/指标取证 → 根因分析 → 报告生成"形成完整排障闭环。系统提供 rule 确定性工作流与 Eino Agent 智能化分析双模式，默认以 mock + memory 模式运行，无外部依赖即可启动演示，同时支持 DashScope 嵌入、Milvus 向量库、Prometheus 告警、OpenAI 兼容 LLM 等真实生产组件的显式切换。

### 核心职责与实现

#### 1. RAG 知识库引擎设计实现

- 抽象 Loader / Splitter / Embedder / VectorStore 四大接口，通过工厂方法支持 mock ↔ DashScope、memory ↔ Milvus 的可配置切换。
- 实现 **Markdown 感知的文本切片器**：按 `#/##/###` 标题层级分段，段内按预设 chunk_size（800 字符）+ overlap（100 字符）切块，保留标题路径作为语义上下文。
- 内存向量库基于**余弦相似度**进行 Top-K 检索；Milvus 实现基于 HTTP REST API，支持自动建集合、upsert、search、delete。
- 上传 SOP 文档后，Chat 页面支持带 citation 来源和 trace_id 的流式 RAG 问答。

#### 2. AIOps 双模式告警分析编排

- **Rule 模式**：确定性执行 6 步工作流（AlertCollector → SOPRetriever → EvidencePlanner → EvidenceCollector → RootCauseAnalyzer → ReportGenerator），每步骤记录状态（success/failed/skipped）、耗时、输入输出快照；根因分析基于证据中的 panic / restart_count 关键字做规则判断。
- **Agent 模式**：基于 **Eino 框架**编排 LLM + 5 个只读 Agent 工具（query_active_alerts / query_internal_docs / query_logs / query_metrics / get_current_time），工具定义使用 JSON Schema；实现**确定性 AgentRunner** 按固定流程调用工具，避免 LLM 工具选择不稳定导致的演示失败。
- **Fallback 机制**：Agent 失败时自动降级到 Rule 模式，在前端工作流时间轴中插入 AgentAnalyzer failed 节点，保留完整审计路径。

#### 3. Provider 抽象与可插拔数据源

- 定义 AlertProvider / LogProvider / MetricProvider 三大接口，默认 **Mock 实现**返回稳定可复现的 demo 数据（`billing-service` 服务下线、`panic: runtime error` 日志、`restart_count` 递增指标）。
- 实现 **Prometheus AlertManager API** 真实告警查询，为后续接入真实 Prometheus metrics 和日志平台预留扩展点。

#### 4. 全链路可观测性与工程保障

- **全链路 trace_id**：前端自动生成 `web-{uuid}`，通过 HTTP Header `X-Trace-ID` 透传至后端，后端的 Gin middleware 提取或生成 trace_id 写入 context 和统一响应体 `{code, message, data, trace_id}`。
- **安全设计**：Agent 工具仅读不写（不执行 SQL/命令/关闭告警）；文件上传做扩展名校验 + 大小限制 + 路径穿越防护；前端 Markdown 渲染经 DOMPurify 清洗。
- **测试体系**：61 个核心测试用例覆盖 RAG、配置解析、上传校验、API 响应、Agent 工具、fallback 等关键路径；`go test ./...` 和前端生产构建均通过。

#### 5. 前端控制台构建

- React + TypeScript + Vite 实现 5 页面控制台（Knowledge / Chat / AI Ops / Reports / Settings），展示 Agent 工作流执行轨迹时间轴、证据链面板、citation 引用来源和结构化分析报告，报告支持 localStorage 历史持久化。

### 效果数据

- 在 8 条自建排障 QA 评测集上，RAG Recall@3 = 100%，MRR@3 = 0.708。
- AI Ops 工作流在 demo 场景下 6/6 步骤成功执行，自动收集 5 条证据、3 条 SOP citation，生成包含 panic 堆栈、restart_count 和服务下线根因信号的结构化报告。
- 默认模式 `mock + memory + rule` 零外部依赖可启动，支持 Docker Compose 一键部署。

### STAR 速览

| 维度 | 内容 |
|---|---|
| **背景** | Oncall 排障需要在告警、SOP、日志、指标间频繁切换，信息分散且依赖人工经验 |
| **任务** | 从零构建可演示、可测试、可扩展的智能 Oncall 助手，默认不依赖外部服务 |
| **行动** | 用 Go 分层实现 API/Service/RAG/Provider/Agent 边界；Eino 编排 Agent 工具；React 构建控制台；Mock/Memory/Rule 默认可运行，DashScope/Milvus/Agent 显式切换 |
| **结果** | 形成上传 SOP → RAG 问答 → 告警分析 → 证据收集 → 报告展示的完整闭环；61 测试用例通过；Docker Compose 一键部署 |

### 面试讲解要点

1. **架构**：前端控制台 → Gin API → KnowledgeService（RAG）+ AIOpsService（Rule/Agent 双模式）→ Provider（Mock/Prometheus）+ Agent Tool（5 只读工具）
2. **难点**：既要接入 RAG/Agent，又要保证默认环境稳定、无外部依赖、测试可复现
3. **取舍**：默认 Rule 保证确定性（适合 demo 和测试），Agent 作为显式增强并支持 fallback
4. **扩展**：真实 LLM（OpenAI 兼容接口）、DashScope 嵌入、Milvus 向量库、Prometheus 均通过配置显式启用
5. **后续**：接入真实 Prometheus metrics / 日志平台 / 报告持久化 / OpenTelemetry tracing
