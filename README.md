# OnCall Agent

OnCall Agent 是面向后端研发、SRE、平台工程师和一线 Oncall 值班人员的智能故障分析助手。项目定位是“故障排查辅助系统”，不是自动处置系统。

当前阶段已完成阶段 3A：在阶段 2 的配置安全、统一错误响应、Trace ID、工具边界和上传安全基础上，补齐本地 RAG 知识库闭环。当前 RAG 使用 Mock Embedder + Memory VectorStore，不接真实 LLM、真实 Embedding API、Milvus、Prometheus、日志平台或 MCP。

## 技术栈

- Go 1.22+
- Gin HTTP Framework
- YAML 配置：`gopkg.in/yaml.v3`
- Mock 数据：内置确定性数据，不依赖外部服务
- RAG：本地 Markdown/TXT Loader、标题感知 Splitter、确定性 Mock Embedder、内存向量库
- 日志：Go `log/slog`

## 目录结构

```text
├── cmd/server              # 服务入口
├── configs                 # YAML 配置
├── internal/api            # HTTP controller
├── internal/service        # 业务 service
├── internal/agent          # Agent 编排
├── internal/rag            # RAG 预留模块
├── internal/tools          # Mock 工具
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
go run ./cmd/server
```

可复制 `.env.example` 或 `configs/config.example.yaml` 作为本地配置模板。仓库不应提交真实 API Key、Token、密钥、个人路径或真实外部平台地址；LLM Key、Embedding Key、Milvus 地址、日志平台 Token 等后续接入时必须从环境变量或配置文件读取。

## 测试

```bash
go test ./...
```

当前测试覆盖配置加载、Trace ID 生成和透传、统一响应 trace_id、文件名清理、文件类型校验、文件大小校验、Mock AI Ops workflow、RAG Loader/Splitter/Mock Embedder/Memory VectorStore、KnowledgeService 索引检索删除闭环，以及 ChatService 的 mock RAG citations。

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

阶段 3A 中，`/api/chat` 会先检索本地知识库。如果命中知识库，会返回 mock RAG 回答和 `citations`；如果没有命中，会提示先上传 SOP 文档。

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

支持 multipart。阶段 3A 只允许 `.md`、`.markdown` 和 `.txt`，限制文件大小，清理文件名，禁止路径穿越，上传目录由配置控制，保存前会确认目标路径位于上传目录内。文件保存后会执行 Loader -> Splitter -> Mock Embedder -> Memory VectorStore 索引：

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
  "service": "payment-api"
}
```

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
- `service` 提供 `Chat`、`StreamChat`、`UploadMetadata`、`SaveUpload`、`IndexFile`、`Search`、`ListDocuments`、`DeleteDocument`、`Analyze`，业务层和工具层通过 `error` 返回失败。
- 请求 trace_id 贯穿 API、Service、Agent workflow、Tool call、日志和 API 响应。
- 基础日志包含 trace_id、method、path、service_name、agent_step_name、tool_name 和 error 等字段，为后续接入 OpenTelemetry 或日志平台预留结构。
- Mock 工具实现统一接口：

```go
type Tool interface {
    Name() string
    Timeout() time.Duration
    Execute(ctx context.Context, input any) (any, error)
}
```

- 每次工具执行都带 context timeout。工具失败不会导致服务退出，AI Ops workflow 会记录失败步骤并生成降级报告。
- AI Ops 当前为确定性 Mock 工作流：

```text
AlertCollector -> SOPRetriever -> EvidenceCollector -> ReportGenerator
```

- RAG 当前实现为本地 Mock：Markdown/TXT Loader 读取文档，Splitter 按 Markdown 标题和长度切片，Mock Embedder 生成稳定归一化向量，Memory VectorStore 用 cosine similarity 检索 chunk。
- RAG 流程日志包含 trace_id、document_id、chunk_count、query、top_k、result_count。当前 mock 模式不会访问外部网络、真实模型、Milvus、真实告警平台、真实日志平台或真实知识库。

## 后续计划

- 阶段 3B：在现有 `Loader`、`Splitter`、`Embedder`、`VectorStore` 接口边界下，替换为 Eino Embedder + Milvus VectorStore，保持密钥与地址配置化。
- 阶段 4：接入更完整的向量索引、召回排序和持久化文档记录。
- 阶段 5：接入 LLM Provider，支持普通聊天和 RAG 问答。
- 阶段 6：将 AI Ops Mock tools 替换为真实告警、日志和知识库工具。
- 工程化：继续补充配置校验、集成测试、Docker Compose 和可观测性适配。
