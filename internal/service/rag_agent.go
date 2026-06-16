package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/cloudwego/eino/compose"

	"oncall-agent/internal/infra/trace"
	"oncall-agent/internal/model/domain"
	"oncall-agent/internal/rag"
)

type RAGAgent struct {
	log         *slog.Logger
	loader      rag.Loader
	splitter    rag.Splitter
	embedder    rag.Embedder
	vectorStore rag.VectorStore
	defaultTopK int
}

type ragIndexState struct {
	FilePath   string
	TraceID    string
	Doc        domain.Document
	Chunks     []domain.Chunk
	Texts      []string
	Vectors    [][]float32
	Result     domain.IndexResult
	Plan       *domain.AgentPlan
	Iterations []domain.AgentIteration
	Steps      []domain.WorkflowStep
}

type ragSearchState struct {
	Query      string
	TopK       int
	TraceID    string
	Vector     []float32
	Results    []domain.SearchResult
	Plan       *domain.AgentPlan
	Iterations []domain.AgentIteration
	Steps      []domain.WorkflowStep
}

func NewRAGAgent(log *slog.Logger, loader rag.Loader, splitter rag.Splitter, embedder rag.Embedder, vectorStore rag.VectorStore, defaultTopK int) *RAGAgent {
	if log == nil {
		log = slog.Default()
	}
	return &RAGAgent{log: log, loader: loader, splitter: splitter, embedder: embedder, vectorStore: vectorStore, defaultTopK: defaultTopK}
}

func (a *RAGAgent) IndexFile(ctx context.Context, filePath string) (domain.IndexResult, domain.Document, *domain.AgentPlan, []domain.AgentIteration, []domain.WorkflowStep, error) {
	traceID := ensureTraceID(&ctx)
	state := ragIndexState{FilePath: filePath, TraceID: traceID, Plan: newAgentPlan("索引知识文档", []domain.AgentPlanStep{
		{ID: "validate", Name: "Validate", Tool: "upload_policy", Status: stepStatusSkipped, Rationale: "文件已在 API 层完成上传校验"},
		{ID: "load", Name: "Load", Tool: "rag_loader", Status: stepStatusSkipped},
		{ID: "split", Name: "Split", Tool: "rag_splitter", Status: stepStatusSkipped},
		{ID: "embed", Name: "EmbedDocuments", Tool: "rag_embedder", Status: stepStatusSkipped},
		{ID: "upsert", Name: "Upsert", Tool: "vector_store", Status: stepStatusSkipped},
		{ID: "report", Name: "IndexReport", Status: stepStatusSkipped},
	})}

	wf := compose.NewWorkflow[ragIndexState, ragIndexState]()
	wf.AddLambdaNode("Validate", compose.InvokableLambda(a.indexValidate)).AddInput(compose.START)
	wf.AddLambdaNode("Load", compose.InvokableLambda(a.indexLoad)).AddInput("Validate")
	wf.AddLambdaNode("Split", compose.InvokableLambda(a.indexSplit)).AddInput("Load")
	wf.AddLambdaNode("EmbedDocuments", compose.InvokableLambda(a.indexEmbed)).AddInput("Split")
	wf.AddLambdaNode("Upsert", compose.InvokableLambda(a.indexUpsert)).AddInput("EmbedDocuments")
	wf.AddLambdaNode("IndexReport", compose.InvokableLambda(a.indexReport)).AddInput("Upsert")
	wf.End().AddInput("IndexReport")
	runnable, err := wf.Compile(ctx, compose.WithGraphName("RAGIndexWorkflow"))
	if err != nil {
		return domain.IndexResult{}, domain.Document{}, state.Plan, state.Iterations, state.Steps, err
	}
	out, err := runnable.Invoke(ctx, state)
	if err != nil {
		return domain.IndexResult{}, domain.Document{}, state.Plan, state.Iterations, state.Steps, err
	}
	return out.Result, out.Doc, out.Plan, out.Iterations, out.Steps, nil
}

func (a *RAGAgent) Search(ctx context.Context, query string, topK int) (domain.KnowledgeSearchResult, error) {
	traceID := ensureTraceID(&ctx)
	if topK <= 0 {
		topK = a.defaultTopK
	}
	state := ragSearchState{Query: query, TopK: topK, TraceID: traceID, Plan: newAgentPlan("检索知识库证据", []domain.AgentPlanStep{
		{ID: "normalize", Name: "NormalizeQuery", Status: stepStatusSkipped},
		{ID: "embed_query", Name: "EmbedQuery", Tool: "rag_embedder", Status: stepStatusSkipped},
		{ID: "retrieve", Name: "Retrieve", Tool: "vector_store", Status: stepStatusSkipped},
		{ID: "filter", Name: "Filter", Status: stepStatusSkipped},
		{ID: "citation", Name: "CitationBuild", Status: stepStatusSkipped},
	})}
	wf := compose.NewWorkflow[ragSearchState, ragSearchState]()
	wf.AddLambdaNode("NormalizeQuery", compose.InvokableLambda(a.searchNormalize)).AddInput(compose.START)
	wf.AddLambdaNode("EmbedQuery", compose.InvokableLambda(a.searchEmbed)).AddInput("NormalizeQuery")
	wf.AddLambdaNode("Retrieve", compose.InvokableLambda(a.searchRetrieve)).AddInput("EmbedQuery")
	wf.AddLambdaNode("Filter", compose.InvokableLambda(a.searchFilter)).AddInput("Retrieve")
	wf.AddLambdaNode("CitationBuild", compose.InvokableLambda(a.searchReport)).AddInput("Filter")
	wf.End().AddInput("CitationBuild")
	runnable, err := wf.Compile(ctx, compose.WithGraphName("RAGSearchWorkflow"))
	if err != nil {
		return domain.KnowledgeSearchResult{TraceID: traceID, Plan: state.Plan}, err
	}
	out, err := runnable.Invoke(ctx, state)
	if err != nil {
		return domain.KnowledgeSearchResult{TraceID: traceID, Plan: state.Plan}, err
	}
	return domain.KnowledgeSearchResult{
		TraceID:    traceID,
		Results:    out.Results,
		Plan:       out.Plan,
		Iterations: out.Iterations,
		Steps:      out.Steps,
	}, nil
}

func (a *RAGAgent) indexValidate(ctx context.Context, in ragIndexState) (ragIndexState, error) {
	if strings.TrimSpace(in.FilePath) == "" {
		return a.failIndexStep(ctx, in, "Validate", "file_path is required", fmt.Errorf("file_path is required"))
	}
	return a.okIndexStep(ctx, in, "Validate", "确认索引输入文件路径")
}

func (a *RAGAgent) indexLoad(ctx context.Context, in ragIndexState) (ragIndexState, error) {
	doc, err := a.loader.Load(ctx, in.FilePath)
	if err != nil {
		return a.failIndexStep(ctx, in, "Load", "文档加载失败", err)
	}
	in.Doc = doc
	return a.okIndexStep(ctx, in, "Load", "加载文档 "+doc.Name)
}

func (a *RAGAgent) indexSplit(ctx context.Context, in ragIndexState) (ragIndexState, error) {
	chunks, err := a.splitter.Split(ctx, in.Doc)
	if err != nil {
		return a.failIndexStep(ctx, in, "Split", "文档切分失败", err)
	}
	in.Chunks = chunks
	in.Texts = make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		in.Texts = append(in.Texts, chunk.Content)
	}
	return a.okIndexStep(ctx, in, "Split", fmt.Sprintf("切分为 %d 个 chunk", len(chunks)))
}

func (a *RAGAgent) indexEmbed(ctx context.Context, in ragIndexState) (ragIndexState, error) {
	vectors, err := a.embedder.EmbedDocuments(ctx, in.Texts)
	if err != nil {
		return a.failIndexStep(ctx, in, "EmbedDocuments", "文档向量化失败", err)
	}
	in.Vectors = vectors
	return a.okIndexStep(ctx, in, "EmbedDocuments", fmt.Sprintf("生成 %d 个向量", len(vectors)))
}

func (a *RAGAgent) indexUpsert(ctx context.Context, in ragIndexState) (ragIndexState, error) {
	if err := a.vectorStore.Upsert(ctx, in.Chunks, in.Vectors); err != nil {
		return a.failIndexStep(ctx, in, "Upsert", "向量写入失败", err)
	}
	return a.okIndexStep(ctx, in, "Upsert", "写入向量索引")
}

func (a *RAGAgent) indexReport(ctx context.Context, in ragIndexState) (ragIndexState, error) {
	in.Result = domain.IndexResult{DocumentID: in.Doc.ID, ChunkCount: len(in.Chunks)}
	in.Plan.Status = stepStatusSuccess
	in.Plan.UpdatedAt = time.Now()
	return a.okIndexStep(ctx, in, "IndexReport", fmt.Sprintf("完成索引 document_id=%s", in.Doc.ID))
}

func (a *RAGAgent) searchNormalize(ctx context.Context, in ragSearchState) (ragSearchState, error) {
	in.Query = strings.Join(strings.Fields(in.Query), " ")
	if in.Query == "" {
		return a.failSearchStep(ctx, in, "NormalizeQuery", "query is required", fmt.Errorf("query is required"))
	}
	return a.okSearchStep(ctx, in, "NormalizeQuery", "规范化检索问题")
}

func (a *RAGAgent) searchEmbed(ctx context.Context, in ragSearchState) (ragSearchState, error) {
	vector, err := a.embedder.EmbedQuery(ctx, in.Query)
	if err != nil {
		return a.failSearchStep(ctx, in, "EmbedQuery", "查询向量化失败", err)
	}
	in.Vector = vector
	return a.okSearchStep(ctx, in, "EmbedQuery", "生成查询向量")
}

func (a *RAGAgent) searchRetrieve(ctx context.Context, in ragSearchState) (ragSearchState, error) {
	results, err := a.vectorStore.Search(ctx, in.Vector, in.TopK)
	if err != nil {
		return a.failSearchStep(ctx, in, "Retrieve", "向量检索失败", err)
	}
	in.Results = results
	return a.okSearchStep(ctx, in, "Retrieve", fmt.Sprintf("召回 %d 个 chunk", len(results)))
}

func (a *RAGAgent) searchFilter(ctx context.Context, in ragSearchState) (ragSearchState, error) {
	filtered := make([]domain.SearchResult, 0, len(in.Results))
	for _, result := range in.Results {
		if result.Score > 0 {
			filtered = append(filtered, result)
		}
	}
	in.Results = filtered
	return a.okSearchStep(ctx, in, "Filter", fmt.Sprintf("过滤后保留 %d 个 chunk", len(filtered)))
}

func (a *RAGAgent) searchReport(ctx context.Context, in ragSearchState) (ragSearchState, error) {
	in.Plan.Status = stepStatusSuccess
	in.Plan.UpdatedAt = time.Now()
	return a.okSearchStep(ctx, in, "CitationBuild", "构建检索结果和引用元数据")
}

func (a *RAGAgent) okIndexStep(ctx context.Context, in ragIndexState, name, summary string) (ragIndexState, error) {
	in.Steps, in.Iterations = appendAgentStep(ctx, in.Steps, in.Iterations, len(in.Iterations)+1, name, stepStatusSuccess, summary, "", in.TraceID)
	markPlanStep(in.Plan, name, stepStatusSuccess, "")
	return in, nil
}

func (a *RAGAgent) failIndexStep(ctx context.Context, in ragIndexState, name, summary string, err error) (ragIndexState, error) {
	in.Steps, in.Iterations = appendAgentStep(ctx, in.Steps, in.Iterations, len(in.Iterations)+1, name, stepStatusFailed, summary, err.Error(), in.TraceID)
	markPlanStep(in.Plan, name, stepStatusFailed, err.Error())
	in.Plan.Status = stepStatusFailed
	in.Plan.UpdatedAt = time.Now()
	return in, err
}

func (a *RAGAgent) okSearchStep(ctx context.Context, in ragSearchState, name, summary string) (ragSearchState, error) {
	in.Steps, in.Iterations = appendAgentStep(ctx, in.Steps, in.Iterations, len(in.Iterations)+1, name, stepStatusSuccess, summary, "", in.TraceID)
	markPlanStep(in.Plan, name, stepStatusSuccess, "")
	return in, nil
}

func (a *RAGAgent) failSearchStep(ctx context.Context, in ragSearchState, name, summary string, err error) (ragSearchState, error) {
	in.Steps, in.Iterations = appendAgentStep(ctx, in.Steps, in.Iterations, len(in.Iterations)+1, name, stepStatusFailed, summary, err.Error(), in.TraceID)
	markPlanStep(in.Plan, name, stepStatusFailed, err.Error())
	in.Plan.Status = stepStatusFailed
	in.Plan.UpdatedAt = time.Now()
	return in, err
}

func newAgentPlan(goal string, steps []domain.AgentPlanStep) *domain.AgentPlan {
	now := time.Now()
	return &domain.AgentPlan{Goal: goal, Status: "running", Steps: steps, CreatedAt: now, UpdatedAt: now}
}

func markPlanStep(plan *domain.AgentPlan, name, status, errText string) {
	if plan == nil {
		return
	}
	for i := range plan.Steps {
		if plan.Steps[i].Name == name || plan.Steps[i].ID == strings.ToLower(name) {
			plan.Steps[i].Status = status
			plan.Steps[i].Error = errText
			plan.UpdatedAt = time.Now()
			return
		}
	}
}

func appendAgentStep(ctx context.Context, steps []domain.WorkflowStep, iterations []domain.AgentIteration, index int, name, status, summary, errText, traceID string) ([]domain.WorkflowStep, []domain.AgentIteration) {
	now := time.Now()
	if traceID == "" {
		traceID = trace.FromContext(ctx)
	}
	step := domain.WorkflowStep{Name: name, Status: status, Summary: summary, Error: errText, StartedAt: now, EndedAt: now, TraceID: traceID}
	iteration := domain.AgentIteration{Index: index, Phase: name, StepID: name, Tool: name, Observation: summary, StartedAt: now, EndedAt: now}
	if errText != "" {
		iteration.Observation = summary + ": " + errText
	}
	return append(steps, step), append(iterations, iteration)
}

func ensureTraceID(ctx *context.Context) string {
	traceID := trace.FromContext(*ctx)
	if traceID == "" {
		traceID = trace.NewID()
		*ctx = trace.WithTraceID(*ctx, traceID)
	}
	return traceID
}
