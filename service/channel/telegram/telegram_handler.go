package telegram

import (
	"fmt"
	"time"

	"github.com/er1cw00/claw.go/base/logger"
	"github.com/er1cw00/claw.go/model"
	tele "gopkg.in/telebot.v4"
)

func (cc *TelegramChannel) handleText(c tele.Context) error {
	var (
		err    error = nil
		sender       = c.Sender()
		chat         = c.Chat()
		text         = c.Text()
	)
	logger.Debugf("==================================================")
	logger.Debugf("uid: %d, chat id: %d", sender.ID, chat.ID)
	inbound := &model.InboundMessage{
		Channel:   cc.Name(),
		SenderID:  fmt.Sprintf("%d", sender.ID),
		ChatID:    fmt.Sprintf("%d", chat.ID),
		Content:   text,
		Timestamp: time.Now(),
	}
	if err = cc.BaseChannel.PublishMessage(inbound); err != nil {
		logger.Errorf("[WeChat] Publish message fail, err: %v", err)
		return err
	}

	return nil
}
func (cc *TelegramChannel) handlePhoto(c tele.Context) error {
	return nil
}
func (cc *TelegramChannel) handleVideo(c tele.Context) error {
	return nil
}
func (cc *TelegramChannel) handleDocument(c tele.Context) error {
	return nil
}
func (cc *TelegramChannel) handleAnimation(c tele.Context) error {
	return nil
}
func (cc *TelegramChannel) handleEdited(c tele.Context) error {
	return nil
}
