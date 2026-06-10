# Demo 演示指南

本指南使用默认 `mock + memory + rule` 模式，不需要外部服务。

## 1. 启动后端

```bash
go run ./cmd/server
```

默认地址：`http://localhost:8080`。

## 2. 启动前端

```bash
cd web/frontend
npm install
npm run dev
```

默认地址：`http://localhost:5173`。

## 3. 上传 SOP

在 Knowledge 页面上传：

```text
docs/demo/告警处理手册.md
```

上传后搜索：

```text
服务下线 panic restart_count
```

预期看到 `告警处理手册.md` 和相关 chunk。

## 4. Chat RAG

在 Chat 页面输入：

```text
服务下线告警应该怎么处理？
```

预期看到回答、citations 和 trace_id。

## 5. AI Ops 分析

在 AI Ops 页面触发分析：

```text
alert_name=服务下线
service=billing-service
```

预期看到：

- 服务下线告警。
- SOP citations。
- panic 日志 evidence。
- restart_count 指标 evidence。
- 根因：应用 panic 导致服务实例重启，引发服务下线。
- 结构化 Markdown 报告。

## 6. Reports

切到 Reports 页面查看历史报告，验证复制、下载和删除。

## curl 快速演示

```bash
curl -X POST http://localhost:8080/api/knowledge/upload \
  -H "X-Trace-ID: trace-demo-upload" \
  -F "file=@docs/demo/告警处理手册.md"

curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -H "X-Trace-ID: trace-demo-chat" \
  -d '{"message":"服务下线告警应该怎么处理？"}'

curl -X POST http://localhost:8080/api/aiops/analyze \
  -H "Content-Type: application/json" \
  -H "X-Trace-ID: trace-demo-aiops" \
  -d '{"alert_name":"服务下线","service":"billing-service"}'
```
