# OnCall Agent

OnCall Agent 是面向后端研发、SRE、平台工程师和一线 Oncall 值班人员的智能故障分析助手。项目定位是“故障排查辅助系统”，不是自动处置系统。

当前阶段已完成阶段 4：在阶段 3B 可配置 RAG provider 基础上，`/api/aiops/analyze` 已升级为稳定、可测试、可演示的 AI Ops 告警分析工作流。默认仍使用 Mock Alert/Log/Metric provider、Mock Embedder 和 Memory VectorStore，便于本地测试、CI 和无外部依赖 demo；也可以通过配置切换到 Eino DashScope Embedder + Milvus VectorStore，以及可选 Prometheus Alert provider。

## 技术栈

- Go 1.23+
- Gin HTTP Framework
- YAML 配置：`gopkg.in/yaml.v3`
- Mock 数据：内置确定性数据，不依赖外部服务
- RAG：Markdown/TXT Loader、标题感知 Splitter、Mock/Eino DashScope Embedder、Memory/Milvus VectorStore
- AI Ops：Mock Alert/Log/Metric provider、可选 Prometheus Alert provider、模板化报告生成
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
├── web/frontend            # 前端预留目录
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
  alert_provider: mock
  log_provider: mock
  metric_provider: mock
  timeout: 10s
  sop_top_k: 3

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
AIOPS_TIMEOUT=10s \
AIOPS_SOP_TOP_K=3 \
go run ./cmd/server
```

可复制 `.env.example` 或 `configs/config.example.yaml` 作为本地配置模板。仓库不应提交真实 API Key、Token、密钥、个人路径或真实外部平台地址；LLM Key、Embedding Key、Milvus 地址、日志平台 Token 等后续接入时必须从环境变量或配置文件读取。

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
  alert_provider: mock
  log_provider: mock
  metric_provider: mock
  timeout: 10s
  sop_top_k: 3
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

## 测试

```bash
go test ./...
```

当前测试覆盖配置加载、Trace ID 生成和透传、统一响应 trace_id、文件名清理、文件类型校验、文件大小校验、AI Ops mock providers、provider factory、AI Ops workflow、AIOps Analyze API、RAG Loader/Splitter/Mock Embedder/Memory VectorStore、KnowledgeService 索引检索删除闭环，以及 ChatService 的 mock RAG citations。

默认测试不依赖 DashScope、Milvus 或外部网络。Milvus 集成测试默认跳过：

```bash
RUN_MILVUS_INTEGRATION_TEST=1 go test ./internal/rag -run TestMilvusVectorStoreIntegration
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
  "citations": []
}
```

阶段 4 固定工作流：

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

## 阶段 4 Demo

准备 SOP 文档 `告警处理手册.md`：

```markdown
# 服务下线
告警解释：服务下线可能因为服务 panic，导致 pod 重启造成的。
解决方案：
1. 根据关键字 "panic" 查询最近 1 小时日志。
2. 结合 panic 堆栈分析导致服务重启的代码问题。
3. 检查 restart_count 是否增加。
```

上传并索引 SOP：

```bash
curl -X POST http://localhost:8080/api/knowledge/upload \
  -H "X-Trace-ID: trace-demo-upload" \
  -F "file=@告警处理手册.md"
```

触发 AI Ops 分析：

```bash
curl -X POST http://localhost:8080/api/aiops/analyze \
  -H "Content-Type: application/json" \
  -H "X-Trace-ID: trace-demo-aiops" \
  -d '{"alert_name":"服务下线","service":"billing-service"}'
```

预期报告能看到：

- `服务下线` 告警。
- SOP 匹配结果和 citations。
- `panic` 日志。
- `restart_count` 指标。
- 根因判断：应用 panic 导致服务实例重启，引发服务下线。

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
- AI Ops Service 依赖 `AlertProvider`、`LogProvider`、`MetricProvider` 接口，不直接依赖具体实现。所有 provider 调用都带 context timeout，失败不会导致服务退出。
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

## 后续计划

- 真实 Prometheus：补充更完整的 label/annotation 映射、过滤条件和集成测试。
- 真实日志平台：接入可配置 LogProvider，并保持默认 mock 模式。
- 真实指标平台：接入 Prometheus Metrics 或其他 MetricProvider。
- LLM ReportGenerator：将模板报告升级为可配置 LLM 生成，同时保留规则兜底。
- 前端报告页：展示步骤、证据、引用和结构化报告。
- 工程化：继续补充配置校验、集成测试、Docker Compose 和可观测性适配。
