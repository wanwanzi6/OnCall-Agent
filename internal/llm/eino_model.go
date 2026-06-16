package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"oncall-agent/internal/infra/config"
)

func NewEinoToolCallingModel(cfg config.LLMConfig) (einomodel.ToolCallingChatModel, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if provider == "" {
		provider = ProviderMock
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	switch provider {
	case ProviderMock:
		return MockEinoChatModel{}, nil
	case ProviderOpenAICompatible:
		if strings.TrimSpace(cfg.APIKey) == "" {
			return nil, fmt.Errorf("llm api_key is required when provider is %q", ProviderOpenAICompatible)
		}
		if strings.TrimSpace(cfg.Model) == "" {
			return nil, fmt.Errorf("llm model is required when provider is %q", ProviderOpenAICompatible)
		}
		baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		return &OpenAICompatibleEinoChatModel{
			apiKey:  cfg.APIKey,
			baseURL: baseURL,
			model:   cfg.Model,
			client:  &http.Client{Timeout: cfg.Timeout},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported llm provider %q", cfg.Provider)
	}
}

type MockEinoChatModel struct {
	tools []*schema.ToolInfo
}

func (m MockEinoChatModel) WithTools(tools []*schema.ToolInfo) (einomodel.ToolCallingChatModel, error) {
	m.tools = tools
	return m, nil
}

func (m MockEinoChatModel) Generate(_ context.Context, input []*schema.Message, _ ...einomodel.Option) (*schema.Message, error) {
	if len(input) == 0 {
		return schema.AssistantMessage("证据不足，未收到用户问题。", nil), nil
	}
	if toolMsg := latestToolMessage(input); toolMsg != nil {
		return schema.AssistantMessage(buildMockToolAnswer(toolMsg.Content), nil), nil
	}
	if hasTool(m.tools, "retrieve_knowledge") {
		args, _ := json.Marshal(map[string]any{"query": latestUserContent(input), "top_k": 3})
		return schema.AssistantMessage("", []schema.ToolCall{{
			ID:   "mock-call-retrieve-knowledge",
			Type: "function",
			Function: schema.FunctionCall{
				Name:      "retrieve_knowledge",
				Arguments: string(args),
			},
		}}), nil
	}
	return schema.AssistantMessage(latestUserContent(input), nil), nil
}

func (m MockEinoChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...einomodel.Option) (*schema.StreamReader[*schema.Message], error) {
	msg, err := m.Generate(ctx, input, opts...)
	if err != nil {
		return nil, err
	}
	reader, writer := schema.Pipe[*schema.Message](1)
	go func() {
		defer writer.Close()
		_ = writer.Send(msg, nil)
	}()
	return reader, nil
}

type OpenAICompatibleEinoChatModel struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
	tools   []*schema.ToolInfo
}

func (m *OpenAICompatibleEinoChatModel) WithTools(tools []*schema.ToolInfo) (einomodel.ToolCallingChatModel, error) {
	if m == nil {
		return nil, fmt.Errorf("llm client is not configured")
	}
	next := *m
	next.tools = tools
	return &next, nil
}

func (m *OpenAICompatibleEinoChatModel) Generate(ctx context.Context, input []*schema.Message, _ ...einomodel.Option) (*schema.Message, error) {
	if m == nil || m.client == nil {
		return nil, fmt.Errorf("llm client is not configured")
	}
	body := openAIChatRequestWithTools{
		Model:    m.model,
		Messages: toOpenAIMessages(input),
		Tools:    toOpenAITools(m.tools),
	}
	if len(body.Tools) > 0 {
		body.ToolChoice = "auto"
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal llm request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create llm request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.apiKey)

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call llm: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read llm response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("llm returned status %d", resp.StatusCode)
	}
	var decoded openAIChatResponseWithTools
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, fmt.Errorf("parse llm response: %w", err)
	}
	if len(decoded.Choices) == 0 {
		return nil, fmt.Errorf("llm returned empty response")
	}
	msg := decoded.Choices[0].Message
	toolCalls := make([]schema.ToolCall, 0, len(msg.ToolCalls))
	for _, tc := range msg.ToolCalls {
		toolCalls = append(toolCalls, schema.ToolCall{
			ID:   tc.ID,
			Type: firstNonEmpty(tc.Type, "function"),
			Function: schema.FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		})
	}
	return schema.AssistantMessage(msg.Content, toolCalls), nil
}

func (m *OpenAICompatibleEinoChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...einomodel.Option) (*schema.StreamReader[*schema.Message], error) {
	msg, err := m.Generate(ctx, input, opts...)
	if err != nil {
		return nil, err
	}
	reader, writer := schema.Pipe[*schema.Message](1)
	go func() {
		defer writer.Close()
		_ = writer.Send(msg, nil)
	}()
	return reader, nil
}

type openAIChatRequestWithTools struct {
	Model      string          `json:"model"`
	Messages   []openAIMessage `json:"messages"`
	Tools      []openAITool    `json:"tools,omitempty"`
	ToolChoice string          `json:"tool_choice,omitempty"`
}

type openAITool struct {
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

type openAIChatResponseWithTools struct {
	Choices []struct {
		Message struct {
			Role      string `json:"role"`
			Content   string `json:"content"`
			ToolCalls []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
}

func toOpenAIMessages(messages []*schema.Message) []openAIMessage {
	out := make([]openAIMessage, 0, len(messages))
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		role := string(msg.Role)
		if role == "" {
			role = "user"
		}
		item := openAIMessage{Role: role, Content: msg.Content, ToolCallID: msg.ToolCallID, Name: msg.ToolName}
		for _, tc := range msg.ToolCalls {
			item.ToolCalls = append(item.ToolCalls, schemaToolCall{
				ID:   tc.ID,
				Type: firstNonEmpty(tc.Type, "function"),
				Function: schemaFunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
		out = append(out, item)
	}
	return out
}

func toOpenAITools(tools []*schema.ToolInfo) []openAITool {
	out := make([]openAITool, 0, len(tools))
	for _, tool := range tools {
		if tool == nil {
			continue
		}
		var params any
		if tool.ParamsOneOf != nil {
			if schemaValue, err := tool.ParamsOneOf.ToJSONSchema(); err == nil {
				params = schemaValue
			}
		}
		out = append(out, openAITool{
			Type: "function",
			Function: openAIFunction{
				Name:        tool.Name,
				Description: tool.Desc,
				Parameters:  params,
			},
		})
	}
	return out
}

func latestToolMessage(messages []*schema.Message) *schema.Message {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i] != nil && messages[i].Role == schema.Tool {
			return messages[i]
		}
	}
	return nil
}

func latestUserContent(messages []*schema.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i] != nil && messages[i].Role == schema.User {
			return strings.TrimSpace(messages[i].Content)
		}
	}
	return ""
}

func hasTool(tools []*schema.ToolInfo, name string) bool {
	for _, tool := range tools {
		if tool != nil && tool.Name == name {
			return true
		}
	}
	return false
}

func buildMockToolAnswer(toolContent string) string {
	var payload struct {
		Results []struct {
			Source string `json:"source"`
			Chunk  struct {
				Content string `json:"content"`
			} `json:"chunk"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(toolContent), &payload); err == nil && len(payload.Results) > 0 {
		item := payload.Results[0]
		content := strings.TrimSpace(item.Chunk.Content)
		if len(content) > 180 {
			content = content[:180] + "..."
		}
		source := firstNonEmpty(item.Source, "知识库")
		return "根据知识库检索结果，" + content + "\n\n引用：" + source
	}
	return "知识库中没有检索到相关内容，请先上传对应 SOP 文档。"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
