package base

import (
	// "github.com/er1cw00/claw.go/base"
	// "github.com/er1cw00/claw.go/base/logger"

	"github.com/er1cw00/claw.go/model"
	"github.com/er1cw00/claw.go/service/bus"
)

const (
	ChannelTelegram = "Telegram"
	ChannelWeChat   = "WeChat"
)

type Channel interface {
	Start() error
	Stop()
	Name() string
	SendMessage(message *model.OutboundMessage) error
	PublishMessage(message *model.InboundMessage) error
}

type BaseChannel struct {
}

func (ch *BaseChannel) PublishMessage(message *model.InboundMessage) error {
	msgBus := bus.GetService().GetMessageBus()
	msgBus.PublishInbound(*message)
	return nil
}
