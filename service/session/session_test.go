package session

import (
	"crypto/md5"
	"fmt"
	"testing"

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
	s := NewSessionStore("").NewSession("key1")
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
	s := NewSessionStore("").NewSession("key1")
	_ = s.AddMessage(provider.Message{Role: provider.RoleUser, Content: "hello"})
	if len(s.GetHistory(10)) != 1 {
		t.Fatalf("expected 1 message before clear")
	}
	s.Clear()
	if len(s.GetHistory(10)) != 0 {
		t.Errorf("expected 0 messages after clear, got %d", len(s.GetHistory(10)))
	}
}
