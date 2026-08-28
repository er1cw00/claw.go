package model

import (
	"time"
)

type InboundSender struct {
	Channel  string // telegram, wechat
	SenderID string // User identifier
	ChatID   string // Chat/channel identifier
}

// InboundMessage is a message received from a chat channel.
type InboundMessage struct {
	Channel            string         // telegram, wechat
	SenderID           string         // User identifier
	ChatID             string         // Chat/channel identifier
	Content            string         // Message text
	Timestamp          time.Time      // Message timestamp
	Media              []string       // Media URLs
	Metadata           map[string]any // Channel-specific data
	SessionKeyOverride *string        // Optional override for thread-scoped sessions
}

// SessionKey returns the unique key for session identification.
func (m InboundMessage) SessionKey() string {
	if m.SessionKeyOverride != nil {
		return *m.SessionKeyOverride
	}
	return m.Channel + ":" + m.ChatID
}

func (m InboundMessage) Sender() *InboundSender {
	return &InboundSender{
		Channel:  m.Channel,
		SenderID: m.SenderID,
		ChatID:   m.ChatID,
	}
}

// OutboundMessage is a message to send to a chat channel.
//
// Metadata can carry routing (message_id, ...), trace flags (_progress),
// and optional OUTBOUND_META_AGENT_UI blobs for rich clients; non-WebUI
// channels may ignore unknown keys.
type OutboundMessage struct {
	Channel  string
	ChatID   string
	Content  string
	ReplyTo  *string
	Media    []string
	Metadata map[string]any
	Buttons  [][]string
}
