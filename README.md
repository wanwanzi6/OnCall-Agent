# OnCall Agent

OnCall Agent 是面向后端研发、SRE、平台工程师和一线 Oncall 值班人员的智能故障分析助手。项目定位是“故障排查辅助系统”，不是自动处置系统。

当前阶段已完成阶段 6：在阶段 5 后端能力基础上新增 Web 运维控制台，把 SOP 上传、知识库检索、RAG Chat、AI Ops 告警分析、报告查看、Markdown 复制/下载串成可演示闭环。后端 `/api/aiops/analyze` 仍支持 `rule` 和 `agent` 两种配置模式；默认是稳定、可测试、无外部依赖的 `rule` 模式，显式启用 `agent` 模式后可 fallback 到 rule workflow。

## 技术栈

- Go 1.23+
- Gin HTTP Framework
- YAML 配置：`gopkg.in/yaml.v3`
- Mock 数据：内置确定性数据，不依赖外部服务
- RAG：Markdown/TXT Loader、标题感知 Splitter、Mock/Eino DashScope Embedder、Memory/Milvus VectorStore
- AI Ops：RuleBasedAnalyzer、EinoAgentAnalyzer、Mock Alert/Log/Metric provider、可选 Prometheus Alert provider、模板化/LLM 报告生成
- 前端：React、Vite、TypeScript、lucide-react、marked、DOMPurify
- 日志：Go `log/slog`

## 目录结构

```text
├── cmd/server              # 服务入口
├── configs                 # YAML 配置
├── internal/api            # HTTP controller
├── internal/service        # 业务 service
├── internal/agent          # Agent 编排
├── internal/rag            # RAG 预留模块
├── internal/tools          # Tool/provider 接口与 Mock 实现
├── internal/infra          # 配置、日志、LLM、Milvus、存储适配层
├── internal/model          # request/response/domain 模型
├── web/frontend            # React + Vite Web 控制台
├── docs                    # 产品和架构文档
└── README.md
```

## 启动方式

```bash
go mod tidy
go run ./cmd/server
```

默认监听 `:8080`。配置文件位于 `configs/config.yaml`：

```yaml
server:
  port: 8080

app:
  env: dev

mock:
  enabled: true

knowledge:
  upload_dir: data/uploads
  max_file_size_bytes: 2097152
  allowed_exts:
    - .md
    - .markdown
    - .txt

rag:
  chunk_size: 800
  chunk_overlap: 100
  embedding_dim: 64
  default_top_k: 3
  embedder_provider: mock
  vector_store_provider: memory

aiops:
  mode: rule
  fallback_to_rule: true
  alert_provider: mock
  log_provider: mock
  metric_provider: mock
  agent:
    max_steps: 12
    timeout: 60s
  timeout: 10s
  sop_top_k: 3

llm:
  provider: mock
  api_key: ${LLM_API_KEY}
  base_url: ${LLM_BASE_URL}
  model: ${LLM_MODEL}
  timeout: 30s

embedding:
  dashscope:
    api_key: ${DASHSCOPE_API_KEY}
    model: text-embedding-v4
    dimensions: 1024
    timeout: 30s

milvus:
  address: localhost:19530
  database: agent
  collection: oncall_knowledge
  vector_field: vector
  timeout: 10s

prometheus:
  base_url: http://localhost:9090
  timeout: 5s
```

也可以通过环境变量覆盖：

```bash
APP_ENV=dev \
SERVER_PORT=8080 \
MOCK_ENABLED=true \
KNOWLEDGE_UPLOAD_DIR=data/uploads \
KNOWLEDGE_MAX_FILE_SIZE_BYTES=2097152 \
RAG_CHUNK_SIZE=800 \
RAG_CHUNK_OVERLAP=100 \
RAG_EMBEDDING_DIM=64 \
RAG_DEFAULT_TOP_K=3 \
RAG_EMBEDDER_PROVIDER=mock \
RAG_VECTOR_STORE_PROVIDER=memory \
AIOPS_ALERT_PROVIDER=mock \
AIOPS_LOG_PROVIDER=mock \
AIOPS_METRIC_PROVIDER=mock \
AIOPS_MODE=rule \
AIOPS_FALLBACK_TO_RULE=true \
AIOPS_AGENT_MAX_STEPS=12 \
AIOPS_AGENT_TIMEOUT=60s \
AIOPS_TIMEOUT=10s \
AIOPS_SOP_TOP_K=3 \
LLM_PROVIDER=mock \
LLM_API_KEY= \
LLM_BASE_URL= \
LLM_MODEL= \
LLM_TIMEOUT=30s \
go run ./cmd/server
```

可复制 `.env.example` 或 `configs/config.example.yaml` 作为本地配置模板。仓库不应提交真实 API Key、Token、密钥、个人路径或真实外部平台地址；LLM Key、Embedding Key、Milvus 地址、日志平台 Token 等后续接入时必须从环境变量或配置文件读取。

## 前端控制台

前端位于 `web/frontend`，默认访问后端 `http://localhost:8080/api`。

安装依赖并启动开发服务器：

```bash
cd web/frontend
npm install
npm run dev
```

默认 Vite 地址为 `http://localhost:5173`。如需指定后端 API 地址：

```bash
VITE_API_BASE_URL=http://localhost:8080/api npm run dev
```

Windows PowerShell：

```powershell
$env:VITE_API_BASE_URL="http://localhost:8080/api"
npm run dev
```

构建前端：

```bash
cd web/frontend
npm run build
```

控制台页面：

- `Knowledge`：上传 `.md`、`.markdown`、`.txt` SOP，查看文档索引，手动搜索知识库。
- `Chat`：RAG 问答，展示 answer、citations 和 trace_id，最近对话保存在 localStorage。
- `AI Ops`：触发告警分析，展示 alerts、workflow steps、evidence、citations、report、trace_id，支持复制和下载 Markdown 报告。
- `Reports`：查看浏览器 localStorage 中最近 AI Ops 报告，支持详情、删除、复制和下载。
- `Settings`：展示 API base URL、health、mock、RAG provider 等非敏感状态。

Markdown 渲染使用 `marked` 解析，并通过 `DOMPurify` 清理后再写入页面，避免直接渲染未清理的模型输出。

## RAG Provider

默认模式不依赖外部服务：

```yaml
rag:
  embedder_provider: mock
  vector_store_provider: memory
```

真实模式使用 Eino DashScope Embedder 和 Milvus VectorStore：

```bash
DASHSCOPE_API_KEY=your-local-key \
RAG_EMBEDDER_PROVIDER=dashscope \
RAG_VECTOR_STORE_PROVIDER=milvus \
MILVUS_ADDRESS=localhost:19530 \
MILVUS_DATABASE=agent \
MILVUS_COLLECTION=oncall_knowledge \
go run ./cmd/server
```

DashScope 相关配置：

```bash
DASHSCOPE_API_KEY=
DASHSCOPE_EMBEDDING_MODEL=text-embedding-v4
DASHSCOPE_EMBEDDING_DIM=1024
DASHSCOPE_EMBEDDING_TIMEOUT=30s
```

Milvus 相关配置：

```bash
MILVUS_ADDRESS=localhost:19530
MILVUS_DATABASE=agent
MILVUS_COLLECTION=oncall_knowledge
MILVUS_VECTOR_FIELD=vector
MILVUS_TIMEOUT=10s
```

启动本地 Milvus standalone：

```bash
docker compose -f deployments/docker-compose.milvus.yml up -d
```

Milvus 数据目录位于 `deployments/.data/`，该目录已加入 `.gitignore`。

## AI Ops Provider

默认模式不依赖真实告警平台、日志平台、指标平台或真实 LLM：

```yaml
aiops:
  mode: rule
  fallback_to_rule: true
  alert_provider: mock
  log_provider: mock
  metric_provider: mock
  agent:
    max_steps: 12
    timeout: 60s
  timeout: 10s
  sop_top_k: 3

llm:
  provider: mock
  timeout: 30s
```

Mock provider 的固定 demo 数据：

- Alert：`服务下线`，服务 `billing-service`，级别 `critical`，地域 `ap-guangzhou`。
- Log：包含 `panic: runtime error: invalid memory address or nil pointer dereference` 和 `pod restarted due to application panic`。
- Metric：包含 `restart_count` 增加和 `error_rate` 短时间升高。

可选 Prometheus Alert provider 只在显式配置时启用，默认测试不会访问外部网络：

```bash
AIOPS_ALERT_PROVIDER=prometheus \
PROMETHEUS_BASE_URL=http://localhost:9090 \
PROMETHEUS_TIMEOUT=5s \
go run ./cmd/server
```

Prometheus provider 查询 `/api/v1/alerts`，只映射 `firing` alerts。日志和指标 provider 当前仅支持 mock；真实日志平台和指标平台留到后续阶段。

### AI Ops Analyzer 模式

`AIOpsService` 只依赖统一的 `AIOpsAnalyzer` 接口，当前有两个实现：

- `RuleBasedAnalyzer`：阶段 4 规则工作流，默认启用，稳定、可测试、不依赖真实 LLM。
- `EinoAgentAnalyzer`：阶段 5 Agent 编排，显式设置 `aiops.mode: agent` 才启用。

启用 agent 模式：

```yaml
aiops:
  mode: agent
  fallback_to_rule: true
  agent:
    max_steps: 12
    timeout: 60s

llm:
  provider: mock
  timeout: 30s
```

使用 openai-compatible LLM：

```bash
AIOPS_MODE=agent \
AIOPS_FALLBACK_TO_RULE=true \
LLM_PROVIDER=openai-compatible \
LLM_API_KEY=your-local-key \
LLM_BASE_URL=https://api.openai.com/v1 \
LLM_MODEL=your-model \
LLM_TIMEOUT=30s \
go run ./cmd/server
```

`LLM_PROVIDER=openai-compatible` 时必须配置 `LLM_API_KEY` 和 `LLM_MODEL`；服务不会在日志中打印 API key。默认 `LLM_PROVIDER=mock` 不访问外部网络。

Agent 工具列表：

- `query_active_alerts`：调用 `AlertProvider.QueryActiveAlerts`。
- `query_internal_docs`：调用 `KnowledgeService.Search` 检索 SOP。
- `query_logs`：调用 `LogProvider.QueryLogs`。
- `query_metrics`：调用 `MetricProvider.QueryMetrics`。
- `get_current_time`：返回当前时间。

Agent 安全约束：

- 只生成分析报告，不自动修复。
- 不执行 SQL。
- 不请求或执行系统命令。
- 不关闭告警。
- 不编造日志、指标或 SOP；证据不足时必须明确说明。
- 所有 LLM 调用、agent 总流程和 tool 调用都有 timeout。

Fallback 行为：

- `mode=rule`：直接走 `RuleBasedAnalyzer`。
- `mode=agent` 且成功：返回 agent 结果。
- `mode=agent` 失败且 `fallback_to_rule=true`：记录 `AgentAnalyzer` failed step，再走 rule workflow，响应中 `fallback_used=true`。
- `mode=agent` 失败且 `fallback_to_rule=false`：返回错误响应，响应仍包含 API 层 trace_id。

## 测试

前端构建：

```bash
cd web/frontend
npm run build
```

当前前端未配置单独 lint 脚本；`npm run build` 会执行 TypeScript 类型检查和 Vite 生产构建。

后端测试：

```bash
go test ./...
```

当前测试覆盖配置加载、Trace ID 生成和透传、统一响应 trace_id、文件名清理、文件类型校验、文件大小校验、AI Ops mock providers、provider factory、RuleBasedAnalyzer、EinoAgentAnalyzer mock 编排、Agent tool wrappers、fallback、prompt 安全约束、AIOps Analyze API、RAG Loader/Splitter/Mock Embedder/Memory VectorStore、KnowledgeService 索引检索删除闭环，以及 ChatService 的 mock RAG citations。

默认测试不依赖真实 LLM、DashScope、Milvus、Prometheus 或外部网络。Milvus 集成测试默认跳过：

```bash
RUN_MILVUS_INTEGRATION_TEST=1 go test ./internal/rag -run TestMilvusVectorStoreIntegration
```

可选 Agent 集成测试预留使用环境变量显式开启；默认不运行真实 LLM：

```bash
RUN_AGENT_INTEGRATION_TEST=1 \
AIOPS_MODE=agent \
LLM_PROVIDER=openai-compatible \
LLM_API_KEY=your-local-key \
LLM_BASE_URL=https://api.openai.com/v1 \
LLM_MODEL=your-model \
go test ./internal/service -run AgentIntegration
```

## API 列表

### 健康检查

```http
GET /api/health
```

### 普通聊天

```http
POST /api/chat
Content-Type: application/json

{
  "message": "服务下线告警应该怎么处理？"
}
```

`/api/chat` 会先检索知识库。如果命中知识库，会返回 mock RAG 回答和 `citations`；如果没有命中，会提示先上传 SOP 文档。mock+memory 和 dashscope+milvus 模式使用同一套 API 响应结构。

响应示例：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "answer": "根据知识库内容，服务下线告警通常需要先查询最近 1 小时 panic 日志。",
    "sources": ["告警处理手册.md"],
    "citations": [
      {
        "chunk_id": "chk_xxx",
        "document_id": "doc_xxx",
        "source": "告警处理手册.md",
        "score": 0.87,
        "content": "服务下线告警通常需要先查询最近 1 小时 panic 日志。"
      }
    ],
    "mock": true
  },
  "trace_id": "..."
}
```

### SSE 流式聊天

```http
POST /api/chat/stream
Content-Type: application/json

{
  "message": "服务下线告警应该怎么处理？"
}
```

### 知识库上传

支持 multipart。当前只允许 `.md`、`.markdown` 和 `.txt`，限制文件大小，清理文件名，禁止路径穿越，上传目录由配置控制，保存前会确认目标路径位于上传目录内。文件保存后会执行 Loader -> Splitter -> Embedder -> VectorStore 索引：

```http
POST /api/knowledge/upload
Content-Type: multipart/form-data

file=@告警处理手册.md
```

也支持 Mock JSON 调试：

```http
POST /api/knowledge/upload
Content-Type: application/json

{
  "file_name": "告警处理手册.md",
  "size": 2048
}
```

JSON 调试模式只做上传元数据校验，不读取或索引本地文件。

### 知识库检索

```http
POST /api/knowledge/search
Content-Type: application/json

{
  "query": "服务下线怎么处理",
  "top_k": 3
}
```

响应 `data` 示例：

```json
{
  "results": [
    {
      "score": 0.87,
      "source": "告警处理手册.md",
      "title_path": "服务下线 > 日志排查",
      "chunk": {
        "id": "chk_xxx",
        "document_id": "doc_xxx",
        "content": "服务下线后先查询最近 1 小时 panic 日志。",
        "index": 0,
        "metadata": {
          "source_file": "告警处理手册.md",
          "title_path": "服务下线 > 日志排查"
        }
      }
    }
  ]
}
```

### 知识库文档

```http
GET /api/knowledge/documents
DELETE /api/knowledge/documents/:id
```

### AI Ops 分析

```http
POST /api/aiops/analyze
Content-Type: application/json

{
  "alert_name": "服务下线",
  "service": "billing-service"
}
```

`alert_name` 和 `service` 可为空；默认 mock provider 会返回稳定的 `billing-service` 服务下线告警。响应 `data` 包含完整工作流结果：

```json
{
  "trace_id": "trace_xxx",
  "report": "告警分析报告\n\n一、活跃告警\n...",
  "alerts": [],
  "steps": [],
  "evidence": [],
  "citations": [],
  "mode": "rule",
  "fallback_used": false
}
```

默认 `rule` 工作流：

```text
AlertCollector -> SOPRetriever -> EvidencePlanner -> EvidenceCollector -> RootCauseAnalyzer -> ReportGenerator
```

工作流行为：

- `AlertCollector` 查询活跃告警；无活跃告警时返回明确报告，后续步骤标记为 `skipped`。
- `SOPRetriever` 复用 `KnowledgeService.Search`，按告警名称和描述检索 SOP，生成 `citations` 和 `sop` evidence。
- `EvidencePlanner` 用规则生成日志和指标查询计划；服务下线场景默认查询 `panic`，时间范围为告警前后 1 小时。
- `EvidenceCollector` 调用日志和指标 provider，生成 `log` 和 `metric` evidence。某个 provider 失败时记录 failed step 和错误 evidence，并继续生成报告。
- `RootCauseAnalyzer` 使用规则判断根因；如果日志包含 panic 且指标包含 restart_count，则倾向于应用 panic 导致 pod 重启。
- `ReportGenerator` 生成结构化报告，不执行自动修复、不执行 SQL、不关闭告警。

报告结构：

```text
告警分析报告

一、活跃告警
...

二、SOP 匹配结果
...

三、证据收集
...

四、根因分析
...

五、处理建议
...

六、结论
...
```

## 阶段 6 Demo

完整演示链路：

```text
上传 SOP 文档
-> 查看知识库文档和索引状态
-> 手动搜索知识库
-> Chat RAG 问答并查看 citations
-> AI Ops 触发告警分析
-> 查看 alerts、workflow steps、evidence、citations、report、trace_id
-> Reports 查看历史报告
-> 复制或下载 Markdown 报告
```

1. 启动后端：

```bash
go run ./cmd/server
```

2. 启动前端：

```bash
cd web/frontend
npm install
npm run dev
```

3. 在浏览器打开 `http://localhost:5173`。

4. 准备 SOP 文档 `告警处理手册.md`：

```markdown
# 服务下线
告警解释：服务下线可能因为服务 panic，导致 pod 重启造成的。
解决方案：
1. 根据关键字 "panic" 查询最近 1 小时日志。
2. 结合 panic 堆栈分析导致服务重启的代码问题。
3. 检查 restart_count 是否增加。
```

5. 在 `Knowledge` 页面上传 SOP，确认文档列表出现 `document_id`，然后搜索 `服务下线 panic restart_count`，检查 `score/source/title_path/content`。

也可以用 curl 上传并索引 SOP：

```bash
curl -X POST http://localhost:8080/api/knowledge/upload \
  -H "X-Trace-ID: trace-demo-upload" \
  -F "file=@告警处理手册.md"
```

6. 在 `Chat` 页面输入：

```text
服务下线告警应该怎么处理？
```

预期能看到 RAG answer、citations 和 trace_id。如果没有上传或没有命中 SOP，会提示先上传对应 SOP 文档。

7. 在 `AI Ops` 页面点击“触发分析”，默认参数：

```text
alert_name=服务下线
service=billing-service
```

也可以用 curl 触发 AI Ops 分析：

```bash
curl -X POST http://localhost:8080/api/aiops/analyze \
  -H "Content-Type: application/json" \
  -H "X-Trace-ID: trace-demo-aiops" \
  -d '{"alert_name":"服务下线","service":"billing-service"}'
```

8. 预期页面和报告能看到：

- `服务下线` 告警。
- SOP 匹配结果和 citations。
- `panic` 日志。
- `restart_count` 指标。
- 根因判断：应用 panic 导致服务实例重启，引发服务下线。
- trace_id，可复制。
- Markdown 报告，可复制或下载。

9. 切到 `Reports` 页面，查看刚才保存到 localStorage 的历史报告，可查看详情、删除、复制或下载。

截图占位：阶段 6 控制台截图待补充，建议补充 `Knowledge`、`Chat`、`AI Ops`、`Reports` 四张页面截图。

## 统一响应

```go
type APIResponse struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
    TraceID string      `json:"trace_id,omitempty"`
}
```

所有 JSON API 响应都会包含 `trace_id`。如果请求 Header 存在 `X-Trace-ID`，服务会优先透传；否则服务端生成新的 trace_id，并写入响应 Header 与响应体。业务错误由 API 层统一转换为标准响应。

## 当前阶段说明

- `controller` 只负责参数解析、调用 service、返回 JSON/SSE。
- `service` 提供 `Chat`、`StreamChat`、`UploadMetadata`、`SaveUpload`、`IndexFile`、`Search`、`ListDocuments`、`DeleteDocument`、`Analyze`，业务层和工具/provider 层通过 `error` 返回失败。
- 请求 trace_id 贯穿 API、Service、AI Ops workflow、provider call、RAG 检索、日志和 API 响应。
- 基础日志包含 trace_id、method、path、service_name、step_name、provider_name、query_summary、result_count、error 等字段，为后续接入 OpenTelemetry 或日志平台预留结构。
- AI Ops Service 依赖 `AIOpsAnalyzer` 接口，可配置选择 `rule` 或 `agent`。Agent 通过工具复用 `AlertProvider`、`LogProvider`、`MetricProvider` 和 `KnowledgeService`，不重复实现 provider 逻辑。
- 所有 provider、tool、agent 和 LLM 调用都带 context timeout，失败通过 `error` 返回，不会导致服务退出。
- RAG 支持两种 provider：`mock + memory` 和 `dashscope + milvus`。Service 层只依赖 `Embedder`、`VectorStore` 接口，不直接依赖 Eino 或 Milvus 具体类型。
- RAG 流程日志包含 trace_id、document_id、chunk_count、query、top_k、result_count、embedder_provider、vector_store_provider。默认 mock 模式不会访问外部网络、真实模型、Milvus、真实告警平台、真实日志平台或真实指标平台。

## 常见错误

- `dashscope api key is required`：启用 `RAG_EMBEDDER_PROVIDER=dashscope` 时未配置 `DASHSCOPE_API_KEY`。
- `milvus address is required`：启用 `RAG_VECTOR_STORE_PROVIDER=milvus` 时未配置 `MILVUS_ADDRESS`。
- `milvus ... failed`：Milvus 未启动、地址错误、collection 创建失败或网络超时。
- `embedding dimension mismatch`：Embedder 返回向量维度与 Milvus collection 配置维度不一致。
- `load milvus collection failed`：collection 创建后加载失败，通常需要检查 Milvus standalone 日志。
- `unsupported aiops ... provider`：AIOps provider 配置值非法，当前默认支持 `mock`，Alert provider 额外支持 `prometheus`。
- `prometheus base_url is required`：启用 `AIOPS_ALERT_PROVIDER=prometheus` 时未配置 Prometheus 地址。
- `unsupported aiops mode`：`AIOPS_MODE` 只支持 `rule` 或 `agent`。
- `llm api_key is required`：启用 `LLM_PROVIDER=openai-compatible` 时未配置 `LLM_API_KEY`。

## 后续计划

- 前端测试完善：补充组件测试、API mock 测试和端到端 demo smoke test。
- Docker Compose 一键启动：后端、前端、Milvus、Prometheus mock profile。
- 报告持久化：将当前 localStorage 历史报告升级为后端 Report API。
- 真实 Prometheus：补充更完整的 label/annotation 映射、过滤条件和集成测试。
- 真实日志平台：接入可配置 LogProvider，并保持默认 mock 模式。
- 真实指标平台：接入 Prometheus Metrics 或其他 MetricProvider。
- 原生 Eino ReAct：在现有 AgentRunner 边界内替换为完整 `react.NewAgent` tool-calling 编排。
- 工程化：继续补充配置校验、集成测试、Docker Compose 和可观测性适配。
