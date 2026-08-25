package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProviderFactory(t *testing.T) {
	cases := []string{"openai", "custom", "openrouter", "deepseek"}
	for _, name := range cases {
		p, err := NewProvider(name, "", "")
		if err != nil {
			t.Fatalf("NewProvider(%q) error: %v", name, err)
		}
		if p.Name() != name {
			t.Errorf("expected name %q, got %q", name, p.Name())
		}
	}

	if _, err := NewProvider("unknown", "", ""); err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestCompleteValidation(t *testing.T) {
	p := NewOpenAIProvider("", "")

	if _, err := p.Complete(context.Background(), &CompletionRequest{
		Messages: []Message{{Role: RoleUser, Content: "hello"}},
	}); err == nil {
		t.Error("expected error when model is empty")
	}

	if _, err := p.Complete(context.Background(), &CompletionRequest{
		Model: "gpt-4o",
	}); err == nil {
		t.Error("expected error when messages are empty")
	}

	if _, err := p.Complete(context.Background(), nil); err == nil {
		t.Error("expected error when request is nil")
	}
}

func TestToolCalling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		tools, ok := body["tools"].([]any)
		if !ok || len(tools) == 0 {
			t.Fatalf("expected tools in request, got %+v", body)
		}

		resp := map[string]any{
			"choices": []any{
				map[string]any{
					"finish_reason": "tool_calls",
					"message": map[string]any{
						"role":    "assistant",
						"content": "",
						"tool_calls": []any{
							map[string]any{
								"id":   "call_1",
								"type": "function",
								"function": map[string]any{
									"name":      "get_weather",
									"arguments": `{"location":"Beijing"}`,
								},
							},
						},
					},
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     10,
				"completion_tokens": 5,
				"total_tokens":      15,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewOpenAIProvider("test-key", server.URL)
	resp, err := p.Complete(context.Background(), &CompletionRequest{
		Model: "gpt-4o",
		Messages: []Message{
			{Role: RoleUser, Content: "What is the weather?"},
		},
		Tools: []Tool{
			{
				Type: "function",
				Function: ToolFunction{
					Name:        "get_weather",
					Description: "Get weather for a location",
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"location": map[string]any{"type": "string"},
						},
					},
				},
			},
		},
		ToolChoice: "auto",
	})
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Function.Name != "get_weather" {
		t.Errorf("unexpected function name: %s", resp.ToolCalls[0].Function.Name)
	}
	if resp.FinishReason != "tool_calls" {
		t.Errorf("unexpected finish reason: %s", resp.FinishReason)
	}
}
