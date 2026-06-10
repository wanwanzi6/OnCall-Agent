package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"oncall-agent/internal/infra/config"
	"oncall-agent/internal/infra/trace"
	"oncall-agent/internal/model/response"
	"oncall-agent/internal/service"
	aiopstools "oncall-agent/internal/tools/aiops"
)

func TestTraceIDIsPropagatedToResponse(t *testing.T) {
	router := testRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.Header.Set(trace.HeaderName, "trace-from-client")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if got := w.Header().Get(trace.HeaderName); got != "trace-from-client" {
		t.Fatalf("trace header = %q", got)
	}

	var body response.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body.TraceID != "trace-from-client" {
		t.Fatalf("trace_id = %q", body.TraceID)
	}
}

func TestTraceIDIsGeneratedForResponse(t *testing.T) {
	router := testRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/api/chat", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	var body response.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body.TraceID == "" {
		t.Fatal("trace_id should be generated for error response")
	}
}

func TestAIOpsAnalyzeResponseContainsWorkflowFields(t *testing.T) {
	router := testRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/api/aiops/analyze", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(trace.HeaderName, "trace-api-aiops")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	var body response.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body.TraceID != "trace-api-aiops" {
		t.Fatalf("outer trace_id = %q", body.TraceID)
	}
	data, ok := body.Data.(map[string]any)
	if !ok {
		t.Fatalf("data = %T, want object", body.Data)
	}
	for _, field := range []string{"trace_id", "report", "alerts", "steps", "evidence", "citations"} {
		if _, ok := data[field]; !ok {
			t.Fatalf("data missing field %q: %+v", field, data)
		}
	}
	if data["trace_id"] != "trace-api-aiops" {
		t.Fatalf("data trace_id = %v", data["trace_id"])
	}
}

func TestAIOpsAnalyzeAgentFallbackResponseContainsTraceAndFallbackInfo(t *testing.T) {
	cfg := testRouterConfig(t)
	cfg.AIOps.Mode = service.AnalyzerModeAgent
	cfg.AIOps.FallbackToRule = true
	cfg.AIOps.Agent = config.AgentConfig{MaxSteps: 1, Timeout: 5 * time.Second}
	router := testRouterWithConfig(t, cfg)
	req := httptest.NewRequest(http.MethodPost, "/api/aiops/analyze", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(trace.HeaderName, "trace-api-fallback")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	var body response.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	data, ok := body.Data.(map[string]any)
	if !ok {
		t.Fatalf("data = %T, want object", body.Data)
	}
	if data["trace_id"] != "trace-api-fallback" {
		t.Fatalf("data trace_id = %v", data["trace_id"])
	}
	if data["fallback_used"] != true {
		t.Fatalf("fallback_used = %v, want true", data["fallback_used"])
	}
}

func TestUploadDocumentThenKnowledgeSearchSucceeds(t *testing.T) {
	router := testRouter(t)
	uploadSOP(t, router, "trace-upload-search")

	req := httptest.NewRequest(http.MethodPost, "/api/knowledge/search", bytes.NewBufferString(`{"query":"服务下线 panic restart_count","top_k":3}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(trace.HeaderName, "trace-search")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	data := responseData(t, w.Body.Bytes())
	results := data["results"].([]any)
	if len(results) == 0 {
		t.Fatalf("expected search results: %+v", data)
	}
}

func TestUploadSOPThenChatReturnsCitations(t *testing.T) {
	router := testRouter(t)
	uploadSOP(t, router, "trace-upload-chat")

	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewBufferString(`{"message":"服务下线告警应该怎么处理？"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(trace.HeaderName, "trace-chat-rag")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	data := responseData(t, w.Body.Bytes())
	citations := data["citations"].([]any)
	if len(citations) == 0 {
		t.Fatalf("expected chat citations: %+v", data)
	}
	if data["mock"] != true {
		t.Fatalf("mock = %v, want true", data["mock"])
	}
}

func TestUploadSOPThenAIOpsReturnsFullWorkflow(t *testing.T) {
	router := testRouter(t)
	uploadSOP(t, router, "trace-upload-aiops")

	req := httptest.NewRequest(http.MethodPost, "/api/aiops/analyze", bytes.NewBufferString(`{"alert_name":"服务下线","service":"billing-service"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(trace.HeaderName, "trace-aiops-full")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	data := responseData(t, w.Body.Bytes())
	for _, field := range []string{"alerts", "steps", "evidence", "citations", "report"} {
		if _, ok := data[field]; !ok {
			t.Fatalf("missing %s in aiops response: %+v", field, data)
		}
	}
	if len(data["alerts"].([]any)) == 0 || len(data["steps"].([]any)) == 0 || len(data["evidence"].([]any)) == 0 || len(data["citations"].([]any)) == 0 {
		t.Fatalf("expected full aiops workflow data: %+v", data)
	}
	report := data["report"].(string)
	if !bytes.Contains([]byte(report), []byte("应用 panic")) {
		t.Fatalf("report missing expected root cause:\n%s", report)
	}
}

func uploadSOP(t *testing.T, router http.Handler, traceID string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "告警处理手册.md")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	_, _ = part.Write([]byte(`# 服务下线

告警解释：服务下线可能因为服务 panic，导致 pod 重启造成的。

## 处理步骤

1. 根据关键字 "panic" 查询最近 1 小时日志。
2. 结合 panic 堆栈分析导致服务重启的代码问题。
3. 检查 restart_count 是否增加。
`))
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/knowledge/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set(trace.HeaderName, traceID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("upload status = %d body=%s", w.Code, w.Body.String())
	}
	data := responseData(t, w.Body.Bytes())
	if data["doc_id"] == "" || data["chunk_count"].(float64) == 0 {
		t.Fatalf("unexpected upload response: %+v", data)
	}
}

func responseData(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var body response.APIResponse
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body.Code != 0 {
		t.Fatalf("response code = %d message=%s", body.Code, body.Message)
	}
	data, ok := body.Data.(map[string]any)
	if !ok {
		t.Fatalf("data = %T, want object", body.Data)
	}
	return data
}

func testRouter(t *testing.T) http.Handler {
	t.Helper()
	return testRouterWithConfig(t, testRouterConfig(t))
}

func testRouterConfig(t *testing.T) *config.Config {
	t.Helper()
	return &config.Config{
		Server: config.ServerConfig{Port: 8080},
		App:    config.AppConfig{Env: "test"},
		Mock:   config.MockConfig{Enabled: true},
		Knowledge: config.KnowledgeConfig{
			UploadDir:        t.TempDir(),
			MaxFileSizeBytes: 1024,
			AllowedExts:      []string{".md", ".txt"},
		},
		RAG: config.RAGConfig{ChunkSize: 800, ChunkOverlap: 100, EmbeddingDim: 64, DefaultTopK: 3},
		AIOps: config.AIOpsConfig{
			AlertProvider:  "mock",
			LogProvider:    "mock",
			MetricProvider: "mock",
			Mode:           service.AnalyzerModeRule,
			FallbackToRule: true,
			Agent:          config.AgentConfig{MaxSteps: 12, Timeout: 60 * time.Second},
			Timeout:        10 * time.Second,
			SOPTopK:        3,
		},
		LLM: config.LLMConfig{Provider: "mock", Timeout: 30 * time.Second},
	}
}

func testRouterWithConfig(t *testing.T, cfg *config.Config) http.Handler {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	knowledgeService := service.NewKnowledgeService(true, cfg.Knowledge, cfg.RAG, log)
	services := Services{
		Chat:      service.NewChatService(true, log, knowledgeService),
		Knowledge: knowledgeService,
		AIOps: service.NewAIOpsServiceWithProviders(
			log,
			aiopstools.NewMockAlertProvider(),
			aiopstools.NewMockLogProvider(),
			aiopstools.NewMockMetricProvider(),
			knowledgeService,
			cfg.AIOps,
		),
	}
	return NewRouter(cfg, services, log)
}
