package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

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

func testRouter(t *testing.T) http.Handler {
	t.Helper()
	cfg := &config.Config{
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
		},
	}
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
