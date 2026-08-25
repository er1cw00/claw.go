package model

import (
	"reflect"
	"testing"
)

func TestInboundMessageSessionKey(t *testing.T) {
	msg := InboundMessage{
		Channel: "telegram",
		ChatID:  "chat-1",
	}

	if got, want := msg.SessionKey(), "telegram:chat-1"; got != want {
		t.Fatalf("SessionKey() = %q, want %q", got, want)
	}
}

func TestInboundMessageSessionKeyOverride(t *testing.T) {
	override := "thread-session"
	msg := InboundMessage{
		Channel:            "telegram",
		ChatID:             "chat-1",
		SessionKeyOverride: &override,
	}

	if got := msg.SessionKey(); got != override {
		t.Fatalf("SessionKey() = %q, want %q", got, override)
	}
}

func TestMessageBusInboundQueue(t *testing.T) {
	bus := NewBus()
	msg := InboundMessage{
		Channel:  "telegram",
		SenderID: "user-1",
		ChatID:   "chat-1",
		Content:  "hello",
	}

	bus.PublishInbound(msg)

	if got, want := bus.InboundSize(), 1; got != want {
		t.Fatalf("InboundSize() = %d, want %d", got, want)
	}

	got := <-bus.ConsumeInbound()
	if !reflect.DeepEqual(got, msg) {
		t.Fatalf("ConsumeInbound() = %#v, want %#v", got, msg)
	}

	if got, want := bus.InboundSize(), 0; got != want {
		t.Fatalf("InboundSize() = %d, want %d", got, want)
	}
}

func TestMessageBusOutboundQueue(t *testing.T) {
	bus := NewBus()
	replyTo := "message-1"
	msg := OutboundMessage{
		Channel: "telegram",
		ChatID:  "chat-1",
		Content: "hi",
		ReplyTo: &replyTo,
		Media:   []string{"https://example.com/a.png"},
		Metadata: map[string]any{
			"message_id": "message-2",
		},
		Buttons: [][]string{{"OK", "Cancel"}},
	}

	bus.PublishOutbound(msg)

	if got, want := bus.OutboundSize(), 1; got != want {
		t.Fatalf("OutboundSize() = %d, want %d", got, want)
	}

	got := <-bus.ConsumeOutbound()
	if got.Channel != msg.Channel || got.ChatID != msg.ChatID || got.Content != msg.Content || got.ReplyTo != msg.ReplyTo {
		t.Fatalf("ConsumeOutbound() = %#v, want %#v", got, msg)
	}
	if len(got.Media) != 1 || got.Media[0] != msg.Media[0] {
		t.Fatalf("ConsumeOutbound().Media = %#v, want %#v", got.Media, msg.Media)
	}
	if got.Metadata["message_id"] != msg.Metadata["message_id"] {
		t.Fatalf("ConsumeOutbound().Metadata = %#v, want %#v", got.Metadata, msg.Metadata)
	}
	if len(got.Buttons) != 1 || len(got.Buttons[0]) != 2 || got.Buttons[0][0] != "OK" || got.Buttons[0][1] != "Cancel" {
		t.Fatalf("ConsumeOutbound().Buttons = %#v, want %#v", got.Buttons, msg.Buttons)
	}

	if got, want := bus.OutboundSize(), 0; got != want {
		t.Fatalf("OutboundSize() = %d, want %d", got, want)
	}
}
