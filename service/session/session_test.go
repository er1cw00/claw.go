package session

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/er1cw00/claw.go/base"
	"github.com/er1cw00/claw.go/core/provider"
)

func TestSessionKey(t *testing.T) {
	agent := "agent1"
	channel := "telegram"
	chatID := "12345"

	got := SessionKey(agent, channel, chatID)
	want := fmt.Sprintf("%x", md5.Sum([]byte(agent+":"+channel+":"+chatID)))

	if got != want {
		t.Errorf("SessionKey(%q, %q, %q) = %q, want %q", agent, channel, chatID, got, want)
	}
	if len(got) != 32 {
		t.Errorf("expected md5 hex length 32, got %d", len(got))
	}
}

func TestGetHistory(t *testing.T) {

	s := GetService().GetOrCreate("key-history")
	for i := 0; i < 5; i++ {
		_ = s.AddMessage(provider.Message{Role: provider.RoleUser, Content: fmt.Sprintf("msg%d", i)})
	}

	history := s.GetHistory(2)
	if len(history) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(history))
	}
	if history[0].Content != "msg3" || history[1].Content != "msg4" {
		t.Errorf("unexpected last messages: %+v", history)
	}

	if len(s.GetHistory(0)) != 0 {
		t.Error("expected empty history for limit 0")
	}

	if len(s.GetHistory(10)) != 5 {
		t.Errorf("expected all messages when limit > total, got %d", len(s.GetHistory(10)))
	}
}

func TestClear(t *testing.T) {

	s := GetService().GetOrCreate("key-clear")
	_ = s.AddMessage(provider.Message{Role: provider.RoleUser, Content: "hello"})
	if len(s.GetHistory(10)) != 1 {
		t.Fatalf("expected 1 message before clear")
	}
	s.Reset()
	if len(s.GetHistory(10)) != 0 {
		t.Errorf("expected 0 messages after reset, got %d", len(s.GetHistory(10)))
	}
}

func TestEstimateMessageTokens(t *testing.T) {

	// Empty message should return the baseline of 4 tokens.
	if got := GetService().EstimateMessageTokens(provider.Message{}); got != 4 {
		t.Errorf("EstimateMessageTokens(empty) = %d, want 4", got)
	}

	// Text-only message should be > 4 tokens.
	textMsg := provider.Message{Role: provider.RoleUser, Content: "hello world"}
	if got := GetService().EstimateMessageTokens(textMsg); got <= 4 {
		t.Errorf("EstimateMessageTokens(text) = %d, want > 4", got)
	}

	// Message with name/tool_call_id/tool_calls should increase token count.
	toolCallMsg := provider.Message{
		Role:       provider.RoleAssistant,
		Content:    "call tool",
		Name:       "test-function",
		ToolCallID: "call_123",
		ToolCalls: []provider.ToolCall{
			{
				ID:   "call_123",
				Type: "function",
				Function: provider.ToolCallFunction{
					Name:      "test-function",
					Arguments: `{"foo":"bar"}`,
				},
			},
		},
	}
	if got := GetService().EstimateMessageTokens(toolCallMsg); got <= GetService().EstimateMessageTokens(textMsg) {
		t.Errorf("EstimateMessageTokens(tool call) = %d, want > EstimateMessageTokens(text) = %d", got, GetService().EstimateMessageTokens(textMsg))
	}
}

func TestMain(m *testing.M) {
	var (
		err     error = nil
		cwd           = base.Cwd()
		cfgPath       = filepath.Join(cwd, "../../main/goclaw.yaml")
	)
	if _, err := os.Stat(cfgPath); err != nil {
		panic(err)
	}
	if err := base.Start(cfgPath); err != nil {
		panic(err)
	}
	service := GetService()
	if err = service.Start(); err != nil {
		panic(err)
	}

	m.Run()
	service.Stop()
}
