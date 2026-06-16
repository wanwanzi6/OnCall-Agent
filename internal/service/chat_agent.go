package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	einomodel "github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	"oncall-agent/internal/infra/trace"
	"oncall-agent/internal/model/domain"
)

const chatAgentSystemPrompt = `你是智能 Oncall RAG 问答助手。

规则：
- 必须调用 retrieve_knowledge 检索知识库后再回答。
- 只能基于工具返回的 SOP、文档片段和引用回答。
- 如果没有检索结果，必须明确说明证据不足。
- 不允许编造来源、日志、指标或 SOP。
- 不允许执行修复动作、SQL、系统命令或关闭告警。`

type ChatAgent struct {
	model     einomodel.ToolCallingChatModel
	knowledge *KnowledgeService
	log       *slog.Logger
}

type chatToolRecorder struct {
	mu      sync.Mutex
	outputs []domain.KnowledgeSearchResult
	steps   []domain.WorkflowStep
}

type retrieveKnowledgeInput struct {
	Query string `json:"query"`
	TopK  int    `json:"top_k,omitempty"`
}

func NewChatAgent(model einomodel.ToolCallingChatModel, knowledge *KnowledgeService, log *slog.Logger) *ChatAgent {
	if log == nil {
		log = slog.Default()
	}
	return &ChatAgent{model: model, knowledge: knowledge, log: log}
}

func (a *ChatAgent) Chat(ctx context.Context, message string, mockEnabled bool) (domain.ChatResult, error) {
	traceID := ensureTraceID(&ctx)
	plan := newAgentPlan("回答知识库问答", []domain.AgentPlanStep{
		{ID: "retrieve", Name: "RetrieveKnowledge", Tool: "retrieve_knowledge", Status: stepStatusSkipped},
		{ID: "answer", Name: "Answer", Tool: "react_agent", Status: stepStatusSkipped},
	})
	recorder := &chatToolRecorder{}
	tools := []einotool.BaseTool{
		newRetrieveKnowledgeTool(a.knowledge, recorder),
		newCurrentTimeTool(recorder),
	}
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: a.model,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools:               tools,
			ExecuteSequentially: true,
		},
		MessageModifier: react.NewPersonaModifier(chatAgentSystemPrompt),
		MaxStep:         8,
		GraphName:       "ChatReActAgent",
	})
	if err != nil {
		return domain.ChatResult{}, err
	}
	start := time.Now()
	msg, err := agent.Generate(ctx, []*schema.Message{schema.UserMessage(message)})
	if err != nil {
		return domain.ChatResult{}, err
	}
	steps := recorder.Steps()
	iterations := make([]domain.AgentIteration, 0)
	for i, step := range steps {
		iterations = append(iterations, domain.AgentIteration{
			Index:       i + 1,
			Phase:       step.Name,
			StepID:      step.Name,
			Tool:        step.Tool,
			Observation: step.Summary,
			StartedAt:   step.StartedAt,
			EndedAt:     step.EndedAt,
		})
	}
	answerStep := domain.WorkflowStep{Name: "Answer", Status: stepStatusSuccess, Summary: "基于检索结果生成回答", StartedAt: start, EndedAt: time.Now(), TraceID: traceID}
	steps = append(steps, answerStep)
	iterations = append(iterations, domain.AgentIteration{Index: len(iterations) + 1, Phase: "Answer", StepID: "answer", Tool: "react_agent", Observation: answerStep.Summary, StartedAt: answerStep.StartedAt, EndedAt: answerStep.EndedAt})
	markPlanStep(plan, "RetrieveKnowledge", stepStatusSuccess, "")
	markPlanStep(plan, "Answer", stepStatusSuccess, "")
	plan.Status = stepStatusSuccess
	plan.UpdatedAt = time.Now()

	citations, sources := recorder.Citations()
	answer := strings.TrimSpace(msg.Content)
	if answer == "" {
		answer = "知识库中没有检索到相关内容，请先上传对应 SOP 文档。"
	}
	return domain.ChatResult{
		TraceID:    traceID,
		Answer:     answer,
		Sources:    sources,
		Citations:  citations,
		Mock:       mockEnabled,
		Plan:       plan,
		Iterations: iterations,
		Steps:      steps,
	}, nil
}

func newRetrieveKnowledgeTool(knowledge *KnowledgeService, recorder *chatToolRecorder) einotool.InvokableTool {
	return &chatInvokableTool{
		name: "retrieve_knowledge",
		desc: "检索内部 SOP 和知识库片段，只读工具。",
		params: map[string]*schema.ParameterInfo{
			"query": {Type: schema.String, Desc: "检索问题", Required: true},
			"top_k": {Type: schema.Integer, Desc: "返回数量"},
		},
		run: func(ctx context.Context, args string) (string, int, error) {
			var input retrieveKnowledgeInput
			if err := json.Unmarshal([]byte(defaultJSONObject(args)), &input); err != nil {
				return "", 0, fmt.Errorf("parse retrieve_knowledge input: %w", err)
			}
			if strings.TrimSpace(input.Query) == "" {
				return "", 0, fmt.Errorf("query is required")
			}
			if knowledge == nil {
				output := domain.KnowledgeSearchResult{TraceID: trace.FromContext(ctx), Results: nil}
				recorder.RecordOutput(output)
				return marshalToolOutput(output), 0, nil
			}
			result, err := knowledge.SearchWithTrace(ctx, input.Query, input.TopK)
			if err != nil {
				return "", 0, err
			}
			recorder.RecordOutput(result)
			return marshalToolOutput(result), len(result.Results), nil
		},
		recorder: recorder,
	}
}

func newCurrentTimeTool(recorder *chatToolRecorder) einotool.InvokableTool {
	return &chatInvokableTool{
		name: "get_current_time",
		desc: "获取当前时间，只读工具。",
		run: func(ctx context.Context, args string) (string, int, error) {
			now := time.Now()
			return marshalToolOutput(currentTimeOutput{Now: now, RFC3339: now.Format(time.RFC3339)}), 1, nil
		},
		recorder: recorder,
	}
}

type chatInvokableTool struct {
	name     string
	desc     string
	params   map[string]*schema.ParameterInfo
	run      func(context.Context, string) (string, int, error)
	recorder *chatToolRecorder
}

func (t *chatInvokableTool) Info(context.Context) (*schema.ToolInfo, error) {
	info := &schema.ToolInfo{Name: t.name, Desc: t.desc}
	if len(t.params) > 0 {
		info.ParamsOneOf = schema.NewParamsOneOfByParams(t.params)
	}
	return info, nil
}

func (t *chatInvokableTool) InvokableRun(ctx context.Context, args string, _ ...einotool.Option) (string, error) {
	start := time.Now()
	output, count, err := t.run(ctx, args)
	status := stepStatusSuccess
	summary := fmt.Sprintf("%s 返回 %d 条结果", t.name, count)
	errText := ""
	if err != nil {
		status = stepStatusFailed
		summary = t.name + " 调用失败"
		errText = err.Error()
	}
	t.recorder.RecordStep(domain.WorkflowStep{
		Name:      "ChatTool:" + t.name,
		Tool:      t.name,
		Status:    status,
		Summary:   summary,
		Error:     errText,
		StartedAt: start,
		EndedAt:   time.Now(),
		TraceID:   trace.FromContext(ctx),
	})
	if err != nil {
		return "", err
	}
	return output, nil
}

func (r *chatToolRecorder) RecordOutput(output domain.KnowledgeSearchResult) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.outputs = append(r.outputs, output)
}

func (r *chatToolRecorder) RecordStep(step domain.WorkflowStep) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.steps = append(r.steps, step)
}

func (r *chatToolRecorder) Steps() []domain.WorkflowStep {
	r.mu.Lock()
	defer r.mu.Unlock()
	steps := make([]domain.WorkflowStep, len(r.steps))
	copy(steps, r.steps)
	return steps
}

func (r *chatToolRecorder) Citations() ([]domain.Citation, []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	citations := make([]domain.Citation, 0)
	sources := make([]string, 0)
	seen := make(map[string]bool)
	for _, output := range r.outputs {
		for _, result := range output.Results {
			citation := domain.Citation{
				ChunkID:    result.Chunk.ID,
				DocumentID: result.Chunk.DocumentID,
				Source:     result.Source,
				Score:      result.Score,
				Content:    result.Chunk.Content,
			}
			citations = append(citations, citation)
			if result.Source != "" && !seen[result.Source] {
				seen[result.Source] = true
				sources = append(sources, result.Source)
			}
		}
	}
	return citations, sources
}
