package bus

import (
	"sync"
	"sync/atomic"

	"github.com/er1cw00/claw.go/base/logger"
	"github.com/er1cw00/claw.go/model"
)

const defaultBusBufferSize = 1024

type MessageBus struct {
	inbound  chan model.InboundMessage
	outbound chan model.OutboundMessage
	// outboundMedia chan OutboundMediaMessage
	// audioChunks   chan AudioChunk
	// voiceControls chan VoiceControl

	closeOnce      sync.Once
	done           chan struct{}
	closed         atomic.Bool
	wg             sync.WaitGroup
	streamDelegate atomic.Value // stores StreamDelegate
	eventPublisher atomic.Value // stores EventPublisher
}

func NewMessageBus() *MessageBus {
	return &MessageBus{
		inbound:  make(chan model.InboundMessage, defaultBusBufferSize),
		outbound: make(chan model.OutboundMessage, defaultBusBufferSize),
		// outboundMedia: make(chan OutboundMediaMessage, defaultBusBufferSize),
		// audioChunks:   make(chan AudioChunk, defaultBusBufferSize*4), // Audio chunks need more buffer.
		// voiceControls: make(chan VoiceControl, defaultBusBufferSize),
		done: make(chan struct{}),
	}
}

func (mb *MessageBus) Close() {
	mb.closeOnce.Do(func() {
		//mb.publishCloseEvent(model.KindBusCloseStarted, 0)
		// notify all blocked publishers to exit
		close(mb.done)

		// because every publisher will check mb.closed before acquiring wg
		// so we can be sure that new publishers will not be added new messages after this point
		mb.closed.Store(true)

		// wait for all ongoing Publish calls to finish, ensuring all messages have been sent to channels or exited
		mb.wg.Wait()
		// close channels safely
		close(mb.inbound)
		close(mb.outbound)

	})
}

// PublishInbound publishes a message from a channel to the agent.
func (b *MessageBus) PublishInbound(msg model.InboundMessage) {
	logger.Debugf("PublishInbound >> %v", msg)
	b.inbound <- msg
}

// ConsumeInbound returns the inbound message channel.
func (b *MessageBus) ConsumeInbound() <-chan model.InboundMessage {
	return b.inbound
}

// PublishOutbound publishes a response from the agent to channels.
func (b *MessageBus) PublishOutbound(msg model.OutboundMessage) {
	b.outbound <- msg
}

// ConsumeOutbound returns the outbound message channel.
func (b *MessageBus) ConsumeOutbound() <-chan model.OutboundMessage {
	return b.outbound
}

// InboundSize returns the number of pending inbound messages.
func (b *MessageBus) InboundSize() int {
	return len(b.inbound)
}

// OutboundSize returns the number of pending outbound messages.
func (b *MessageBus) OutboundSize() int {
	return len(b.outbound)
}
