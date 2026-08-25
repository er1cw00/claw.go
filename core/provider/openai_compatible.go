package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
	"github.com/openai/openai-go/shared/constant"
)

// openAICompatibleProvider wraps the official OpenAI Go SDK.
// It works for OpenAI, OpenRouter, DeepSeek and any other OpenAI-compatible endpoint.
type openAICompatibleProvider struct {
	name   string
	client openai.Client
}

// newOpenAICompatibleProvider creates a provider backed by github.com/openai/openai-go.
// The baseURL can be overridden for OpenRouter/DeepSeek/private endpoints.
func newOpenAICompatibleProvider(name, apiKey, baseURL, proxy string) *openAICompatibleProvider {
	httpClient := &http.Client{Timeout: 120 * time.Second}
	if proxy != "" {
		if proxyURL, err := url.Parse(proxy); err == nil {
			httpClient.Transport = &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
		}
	}

	opts := []option.RequestOption{
		option.WithHTTPClient(httpClient),
	}
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(strings.TrimSuffix(baseURL, "/")))
	}

	return &openAICompatibleProvider{
		name:   name,
		client: openai.NewClient(opts...),
	}
}

func (p *openAICompatibleProvider) Name() string {
	return p.name
}

func (p *openAICompatibleProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	resp := &CompletionResponse{}
	if req.Model == "" {
		return resp, fmt.Errorf("model is required")
	}
	if len(req.Messages) == 0 {
		return resp, fmt.Errorf("at least one message is required")
	}

	messages := make([]openai.ChatCompletionMessageParamUnion, len(req.Messages))
	for i, m := range req.Messages {
		var msg openai.ChatCompletionMessageParamUnion
		switch m.Role {
		case RoleSystem:
			msg = openai.SystemMessage(m.Content)
		case RoleAssistant:
			if len(m.ToolCalls) > 0 {
				toolCallParams := make([]openai.ChatCompletionMessageToolCallParam, len(m.ToolCalls))
				for j, tc := range m.ToolCalls {
					toolCallParams[j] = openai.ChatCompletionMessageToolCallParam{
						ID:   tc.ID,
						Type: constant.Function(tc.Type),
						Function: openai.ChatCompletionMessageToolCallFunctionParam{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						},
					}
				}
				msg = openai.ChatCompletionMessageParamUnion{
					OfAssistant: &openai.ChatCompletionAssistantMessageParam{
						Content: openai.ChatCompletionAssistantMessageParamContentUnion{
							OfString: openai.String(m.Content),
						},
						ToolCalls: toolCallParams,
					},
				}
			} else {
				msg = openai.AssistantMessage(m.Content)
			}
		case RoleTool:
			msg = openai.ToolMessage(m.Content, m.ToolCallID)
		default:
			msg = openai.UserMessage(m.Content)
		}
		messages[i] = msg
	}

	params := openai.ChatCompletionNewParams{
		Model:    shared.ChatModel(req.Model),
		Messages: messages,
	}
	if req.MaxTokens > 0 {
		params.MaxTokens = openai.Int(int64(req.MaxTokens))
	}
	if req.Temperature != 0 {
		params.Temperature = openai.Float(req.Temperature)
	}
	if len(req.Tools) > 0 {
		toolParams := make([]openai.ChatCompletionToolParam, len(req.Tools))
		for i, t := range req.Tools {
			toolParams[i] = openai.ChatCompletionToolParam{
				Type: constant.Function(t.Type),
				Function: shared.FunctionDefinitionParam{
					Name:        t.Function.Name,
					Description: openai.String(t.Function.Description),
					Parameters:  shared.FunctionParameters(castToMap(t.Function.Parameters)),
				},
			}
		}
		params.Tools = toolParams
	}
	if req.ToolChoice != "" {
		params.ToolChoice = toToolChoice(req.ToolChoice)
	}

	chatResp, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return resp, fmt.Errorf("chat completion failed: %w", err)
	}
	if len(chatResp.Choices) == 0 {
		return resp, fmt.Errorf("no choices returned")
	}

	choice := chatResp.Choices[0]
	resp.Content = choice.Message.Content
	resp.FinishReason = choice.FinishReason
	if len(choice.Message.ToolCalls) > 0 {
		resp.ToolCalls = make([]ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			resp.ToolCalls[i] = ToolCall{
				ID:   tc.ID,
				Type: string(tc.Type),
				Function: ToolCallFunction{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
	}
	resp.PromptTokens = int(chatResp.Usage.PromptTokens)
	resp.CompletionTokens = int(chatResp.Usage.CompletionTokens)
	resp.TotalTokens = int(chatResp.Usage.TotalTokens)
	return resp, nil
}

// castToMap converts any JSON-like value to map[string]any for SDK parameters.
func castToMap(v any) map[string]any {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]any); ok {
		return m
	}
	if raw, ok := v.(json.RawMessage); ok && len(raw) > 0 {
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err == nil {
			return m
		}
	}
	return nil
}

// toToolChoice maps common tool_choice strings to the SDK union type.
func toToolChoice(choice string) openai.ChatCompletionToolChoiceOptionUnionParam {
	switch strings.ToLower(choice) {
	case "none", "auto", "required":
		return openai.ChatCompletionToolChoiceOptionUnionParam{
			OfAuto: openai.String(choice),
		}
	default:
		// Treat as a named function choice.
		return openai.ChatCompletionToolChoiceOptionParamOfChatCompletionNamedToolChoice(
			openai.ChatCompletionNamedToolChoiceFunctionParam{Name: choice},
		)
	}
}
