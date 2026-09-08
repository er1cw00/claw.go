package provider

import (
	"context"
	"fmt"
	"strings"
)

type MessageRole string

const (
	RoleSystem    MessageRole = "system"
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleTool      MessageRole = "tool"
)

// Message represents a single message in a chat completion request.
type Message struct {
	Role       MessageRole `json:"role"`
	Content    string      `json:"content"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
	Name       string      `json:"name,omitempty"`
	Timestamp  int64       `json:"timestamp,omitempty"`
}

// Tool describes a function tool the model may call.
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction describes the function schema.
type ToolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

// ToolCall represents a single tool call requested by the model.
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction holds the function name and JSON-encoded arguments.
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// CompletionRequest is the input for a non-streaming chat completion.
type CompletionRequest struct {
	Model       string
	Messages    []Message
	MaxTokens   int
	Temperature float64
	Tools       []Tool
	ToolChoice  string // "auto", "none", "required", or "function_name"
	SessionID   string // optional session identifier, used as x-opencode-session by OpenCode
}

// CompletionResponse is the result of a non-streaming chat completion.
type CompletionResponse struct {
	Content          string
	ToolCalls        []ToolCall
	FinishReason     string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// LLMProvider defines the interface implemented by all LLM providers.
type LLMProvider interface {
	// Name returns the provider identifier (e.g. "openai", "openrouter", "deepseek").
	Name() string
	// Complete sends a chat completion request and returns the generated content.
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)
}

// NewProvider returns an LLMProvider implementation by name.
// Supported names: "openai", "openrouter", "deepseek", "opencode".
func NewProvider(name, apiKey, baseURL string) (LLMProvider, error) {
	switch strings.ToLower(name) {
	case "openai":
		return NewOpenAIProvider(apiKey, baseURL), nil
	case "custom":
		return newOpenAICompatibleProvider("custom", apiKey, baseURL, ""), nil
	case "openrouter":
		return NewOpenRouterProvider(apiKey, baseURL), nil
	case "deepseek":
		return NewDeepSeekProvider(apiKey, baseURL), nil
	case "opencode":
		return NewOpenCodeProvider(apiKey, baseURL), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", name)
	}
}
