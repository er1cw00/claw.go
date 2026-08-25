package model

import (
	"time"
)

type Kind string

// String returns the string representation of the event kind.
func (k Kind) String() string {
	return string(k)
}

const (
	// KindGatewayStart is emitted when gateway startup reaches runtime bootstrap.
	KindGatewayStart Kind = "gateway.start"
	// KindGatewayReady is emitted when gateway services are started and ready.
	KindGatewayReady Kind = "gateway.ready"
	// KindGatewayShutdown is emitted when gateway shutdown starts.
	KindGatewayShutdown Kind = "gateway.shutdown"
)

// Event is the runtime event envelope shared across PicoClaw components.
type Event struct {
	ID      string         `json:"id"`
	Kind    Kind           `json:"kind"`
	Time    time.Time      `json:"time"`
	Payload any            `json:"payload,omitempty"`
	Attrs   map[string]any `json:"attrs,omitempty"`
	//Source  Source         `json:"source"`
	// Scope       Scope          `json:"scope,omitempty"`
	// Correlation Correlation    `json:"correlation,omitempty"`
	// Severity    Severity       `json:"severity,omitempty"`
}
