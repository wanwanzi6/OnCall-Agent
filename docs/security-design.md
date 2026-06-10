# 安全设计

项目定位是故障排查辅助系统，不是自动处置系统。默认能力只读、可测试、可演示。

## 配置安全

- 仓库只提交 `.env.example` 和示例配置。
- 真实 API Key、Token、外部平台地址通过环境变量或本地未提交配置传入。
- 默认 `LLM_PROVIDER=mock`、`RAG_EMBEDDER_PROVIDER=mock`、`RAG_VECTOR_STORE_PROVIDER=memory`。
- 日志不打印 LLM 或 DashScope API Key。

## 工具边界

Agent 工具仅允许查询告警、SOP、日志、指标和当前时间。

禁止能力：

- 自动修复或重启服务。
- 执行 SQL。
- 执行系统命令。
- 关闭或确认告警。
- 编造日志、指标、SOP。

## 文件上传安全

- 仅允许 `.md`、`.markdown`、`.txt`。
- 限制文件大小。
- 清理上传文件名。
- 禁止路径穿越。
- 保存前确认目标路径位于上传目录内。

## trace 和错误处理

- 所有 API 统一响应结构。
- 所有 JSON API 返回 `trace_id`。
- 请求头 `X-Trace-ID` 会透传，否则服务端生成。
- Provider、Tool、Agent 和 LLM 调用通过 error 返回失败，API 层统一转换。
- HTTP 层使用 recovery 捕获 panic，避免进程退出。
