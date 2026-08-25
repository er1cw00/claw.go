package wechat

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/er1cw00/claw.go/base/logger"
	"github.com/er1cw00/claw.go/model"
	wechat "github.com/er1cw00/wechat-clawbot-go"
	"github.com/mdp/qrterminal/v3"
)

func (cc *WeChatChannel) handleStatus(ctx context.Context, message string) {
	logger.Infof("[WeChat] Status %s", message)
}
func (cc *WeChatChannel) handleError(ctx context.Context, err error) {
	logger.Infof("[WeChat] Error %v", err)
}

func (cc *WeChatChannel) handleQRCode(ctx context.Context, qrCodeURL string) {
	logger.Infof("==================================================")
	logger.Infof("Scan this QR code with WeChat:")
	logger.Infof("==================================================")
	qrterminal.Generate(qrCodeURL, qrterminal.L, os.Stdout)
	logger.Infof("==================================================")
}

func (cc *WeChatChannel) handleMessage(ctx context.Context, message *wechat.WeixinMessage) error {
	logger.Debug("==================================================")
	logger.Debugf("New message from: %s", message.FromUserID)
	logger.Debugf("ClientId: %s", message.ClientID)
	logger.Debugf("Session: %s", message.SessionID)
	logger.Debugf("Context Token: %s", message.ContextToken)

	text, media, err := cc.bot.ProcessMessage(ctx, message, cc.mediaPath)
	if err != nil {
		logger.Errorf("[WeChat] process message fail, err: %v", err)
		return err
	}
	if text != "" {
		logger.Debugf("Text: %s", text)
	}
	if media != nil {
		logger.Debugf("[%d received]", media.Type)
		logger.Debugf("  File: %s", media.FileName)
		logger.Debugf("  Path: %s", media.FilePath)
		logger.Debugf("  Size: %d bytes", media.FileSize)
		if message.FromUserID != "" && message.ContextToken != "" {
			_, _ = cc.bot.SendText(ctx, message.FromUserID, fmt.Sprintf("Received %d: %s (%d bytes)", media.Type, media.FileName, media.FileSize))
		}
	}
	logger.Debugf("==================================================")
	inbound := &model.InboundMessage{
		Channel:   cc.Name(),
		SenderID:  message.FromUserID,
		ChatID:    message.FromUserID,
		Content:   text,
		Timestamp: time.Now(),
	}
	if err = cc.BaseChannel.PublishMessage(inbound); err != nil {
		logger.Errorf("[WeChat] Publish message fail, err: %v", err)
		return err
	}

	return nil
}
