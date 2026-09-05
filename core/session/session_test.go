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

func newTestSessionStore(t *testing.T) *SessionStore {
	t.Helper()
	root, _ := filepath.Abs("../..")
	cfgPath := filepath.Join(root, "main", "goclaw.yaml")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatalf("config file not found: %v", err)
	}
	if err := base.Start(cfgPath); err != nil {
		t.Fatalf("failed to start base: %v", err)
	}
	t.Cleanup(base.Stop)

	storage := filepath.Join(base.GetSettings().Workspace, "session")

	store := NewSessionStore(storage)
	store.Start()
	t.Cleanup(store.Stop)
	return store
}

func TestSessionKey(t *testing.T) {
	agent := "agent1"
	channel := "telegram"
	chatID := "12345"

	got := sessionKey(agent, channel, chatID)
	want := fmt.Sprintf("%x", md5.Sum([]byte(agent+":"+channel+":"+chatID)))

	if got != want {
		t.Errorf("SessionKey(%q, %q, %q) = %q, want %q", agent, channel, chatID, got, want)
	}
	if len(got) != 32 {
		t.Errorf("expected md5 hex length 32, got %d", len(got))
	}
}

func TestGetHistory(t *testing.T) {
	store := newTestSessionStore(t)
	s := store.GetOrCreate("key-history")
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
	store := newTestSessionStore(t)
	s := store.GetOrCreate("key-clear")
	_ = s.AddMessage(provider.Message{Role: provider.RoleUser, Content: "hello"})
	if len(s.GetHistory(10)) != 1 {
		t.Fatalf("expected 1 message before clear")
	}
	s.Clear()
	if len(s.GetHistory(10)) != 0 {
		t.Errorf("expected 0 messages after clear, got %d", len(s.GetHistory(10)))
	}
}

func TestEstimateMessageTokens(t *testing.T) {
	store := newTestSessionStore(t)

	// Empty message should return the baseline of 4 tokens.
	if got := store.EstimateMessageTokens(provider.Message{}); got != 4 {
		t.Errorf("EstimateMessageTokens(empty) = %d, want 4", got)
	}

	// Text-only message should be > 4 tokens.
	textMsg := provider.Message{Role: provider.RoleUser, Content: "hello world"}
	if got := store.EstimateMessageTokens(textMsg); got <= 4 {
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
	if got := store.EstimateMessageTokens(toolCallMsg); got <= store.EstimateMessageTokens(textMsg) {
		t.Errorf("EstimateMessageTokens(tool call) = %d, want > EstimateMessageTokens(text) = %d", got, store.EstimateMessageTokens(textMsg))
	}
}
