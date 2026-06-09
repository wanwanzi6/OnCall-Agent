# OnCall Agent

OnCall Agent 是面向后端研发、SRE、平台工程师和一线 Oncall 值班人员的智能故障分析助手。项目定位是“故障排查辅助系统”，不是自动处置系统。

当前阶段是阶段 1 后端骨架：先建立可运行、分层清晰、方便扩展的 Go 服务。暂不接 Milvus、LLM、MCP，也不实现完整 AI Ops，只提供 Mock API 打通工程结构。

## 技术栈

- Go 1.22+
- Gin HTTP Framework
- YAML 配置：`gopkg.in/yaml.v3`
- Mock 数据：内置确定性数据，不依赖外部服务

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
```

也可以通过环境变量覆盖：

```bash
APP_ENV=dev SERVER_PORT=8080 MOCK_ENABLED=true go run ./cmd/server
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

### SSE 流式聊天

```http
POST /api/chat/stream
Content-Type: application/json

{
  "message": "服务下线告警应该怎么处理？"
}
```

### 知识库上传

支持 multipart：

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

## 当前阶段说明

- `controller` 只负责参数解析、调用 service、返回 JSON/SSE。
- `service` 提供 `Chat`、`StreamChat`、`Upload`、`Analyze`。
- AI Ops 当前为确定性 Mock 工作流：

```text
AlertCollector -> SOPRetriever -> EvidenceCollector -> ReportGenerator
```

- RAG、Milvus、LLM、MCP 目录已预留，后续可按适配层逐步接入。

## 后续计划

- 接入文档 loader/splitter/embedder/indexer。
- 接入 Milvus 向量索引和检索。
- 接入 LLM Provider，支持普通聊天和 RAG 问答。
- 将 AI Ops Mock tools 替换为真实告警、日志和知识库工具。
- 增加配置校验、日志字段规范、单元测试和 Docker Compose。
