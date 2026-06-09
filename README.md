# OnCall Agent

OnCall Agent 是面向后端研发、SRE、平台工程师和一线 Oncall 值班人员的智能故障分析助手。项目定位是“故障排查辅助系统”，不是自动处置系统。

当前阶段已完成阶段 2：在阶段 1 后端骨架基础上，补齐配置安全、统一错误响应、Trace ID、工具调用边界、文件上传安全、基础日志和关键测试。暂不接真实 Milvus、LLM、Prometheus、日志平台或 MCP，所有 AI Ops 能力仍为确定性 Mock。

## 技术栈

- Go 1.22+
- Gin HTTP Framework
- YAML 配置：`gopkg.in/yaml.v3`
- Mock 数据：内置确定性数据，不依赖外部服务
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
    - .txt
```

也可以通过环境变量覆盖：

```bash
APP_ENV=dev \
SERVER_PORT=8080 \
MOCK_ENABLED=true \
KNOWLEDGE_UPLOAD_DIR=data/uploads \
KNOWLEDGE_MAX_FILE_SIZE_BYTES=2097152 \
go run ./cmd/server
```

可复制 `.env.example` 或 `configs/config.example.yaml` 作为本地配置模板。仓库不应提交真实 API Key、Token、密钥、个人路径或真实外部平台地址；LLM Key、Embedding Key、Milvus 地址、日志平台 Token 等后续接入时必须从环境变量或配置文件读取。

## 测试

```bash
go test ./...
```

当前测试覆盖配置加载、Trace ID 生成和透传、统一响应 trace_id、文件名清理、文件类型校验、文件大小校验、Mock AI Ops workflow 正常执行，以及 Mock tool 失败时 workflow 降级而不退出服务。

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

### SSE 流式聊天

```http
POST /api/chat/stream
Content-Type: application/json

{
  "message": "服务下线告警应该怎么处理？"
}
```

### 知识库上传

支持 multipart。阶段 2 只允许 `.md` 和 `.txt`，限制文件大小，清理文件名，禁止路径穿越，上传目录由配置控制，保存前会确认目标路径位于上传目录内：

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
- `service` 提供 `Chat`、`StreamChat`、`UploadMetadata`、`SaveUpload`、`Analyze`，业务层和工具层通过 `error` 返回失败。
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

- RAG、Milvus、LLM、MCP 目录已预留，后续可按适配层逐步接入。当前 mock 模式不会访问外部网络、真实告警平台、真实日志平台或真实知识库。

## 后续计划

- 阶段 3：接入文档 loader/splitter/embedder/indexer 的 mock-to-real 适配层。
- 阶段 4：接入向量索引和检索，保持密钥与地址配置化。
- 阶段 5：接入 LLM Provider，支持普通聊天和 RAG 问答。
- 阶段 6：将 AI Ops Mock tools 替换为真实告警、日志和知识库工具。
- 工程化：继续补充配置校验、集成测试、Docker Compose 和可观测性适配。
