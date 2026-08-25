package bus

import (
	"context"
	// "sort"
	// "strconv"
	"sync"
	// "sync/atomic"
	// "time"
	"github.com/er1cw00/claw.go/model"
)

// EventBus is an in-process runtime event broadcaster.
type EventBus struct {
	mu sync.RWMutex
	// subs        map[uint64]*eventSubscription
	// orderedSubs []*eventSubscription
	// closed      bool

	// nextSubID atomic.Uint64
	// published atomic.Uint64
	// matched   atomic.Uint64
	// delivered atomic.Uint64
	// dropped   atomic.Uint64
	// blocked   atomic.Uint64
}

func NewEventBus() *EventBus {
	return &EventBus{
		//subs: make(map[uint64]*eventSubscription),
	}
}

func (b *EventBus) Close() error {
	return nil
}

func (b *EventBus) Publish(ctx context.Context, evt model.Event) {

}
func (b *EventBus) publish(ctx context.Context, evt model.Event, nonBlocking bool) {

}
func (b *EventBus) subscribe() {

}

func (b *EventBus) unsubscribe(id uint64) {

}
