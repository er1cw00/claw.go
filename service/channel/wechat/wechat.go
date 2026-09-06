package wechat

import (
	"context"
	"errors"
	"sync"

	"github.com/er1cw00/claw.go/base"
	"github.com/er1cw00/claw.go/base/logger"
	"github.com/er1cw00/claw.go/model"

	//"github.com/er1cw00/claw.go/model"
	ch "github.com/er1cw00/claw.go/service/channel/base"
	wechat "github.com/er1cw00/wechat-clawbot-go"
)

type WeChatChannel struct {
	ch.BaseChannel
	storagePath  string
	mediaPath    string
	contextToken string
	context      context.Context
	wg           sync.WaitGroup
	bot          *wechat.Bot
}

func NewWeChatChannel(storagePath, mediaPath string) *WeChatChannel {
	return &WeChatChannel{
		storagePath: storagePath,
		mediaPath:   mediaPath,
	}
}

func (cc *WeChatChannel) Printf(format string, v ...any) {
	logger.Debugf(format, v...)
}

func (cc *WeChatChannel) Start() error {
	var (
		err  error = nil
		opts       = make([]wechat.Option, 0)
	)
	logger.Debugf("[WeChat] storage path: %s; media path: %s", cc.storagePath, cc.mediaPath)
	opts = append(opts, wechat.WithConfig(wechat.Config{Logger: cc}))
	opts = append(opts, wechat.WithStoragePath(cc.storagePath))

	if err = base.Mkdir(cc.mediaPath); err != nil {
		logger.Errorf("[WeChat] media saved path create fail! err: %v", err)
		return err
	}

	bot := wechat.NewBot(opts...)

	bot.OnError(cc.handleError)
	bot.OnStatus(cc.handleStatus)
	bot.OnMessage(cc.handleMessage)
	bot.OnQRCode(cc.handleQRCode)

	cc.context = context.Background()

	if err = bot.LoadSavedAccount(cc.context); err == nil {
		logger.Info("[WeChat] Loaded saved account..")
	} else if errors.Is(err, wechat.ErrNotLoggedIn) {
		if err := bot.Login(cc.context, true); err != nil {
			logger.Info("[WeChat] login fail: %v\n", err)
			return err
		}
	} else {
		logger.Errorf("[WeChat] load saved account failed: %v\n", err)
		return err
	}
	cc.bot = bot
	cc.startWithSessionRetry()
	logger.Infof("[WeChat] Bot is running. Media files will be saved to: %s\n", cc.mediaPath)

	return nil
}

func (cc *WeChatChannel) Stop() {
	logger.Infof("WeChat Channel Stop")
	cc.bot.Stop()
	cc.wg.Wait()
}

func (cc *WeChatChannel) Name() string {
	return ch.ChannelWeChat
}
func (cc *WeChatChannel) SendMessage(msg *model.OutboundMessage) error {
	var (
		err error = nil
	)
	if msg.Content != "" {
		_, err = cc.bot.SendText(cc.context, cc.bot.GetUserID(), msg.Content)
	}
	if err != nil {
		logger.Errorf("[WeChat] Send message to user fail, err: %v", err)
	}
	return nil
}

func (cc *WeChatChannel) startWithSessionRetry() {
	cc.wg.Add(1)
	loop := func(wg *sync.WaitGroup) {
		defer wg.Done()
		bot := cc.bot
		if err := bot.Start(cc.context); err != nil {
			if errors.Is(err, wechat.ErrSessionExpired) {
				logger.Warnf("[WeChat] Session expired, retrying QR login once...")
				if err := bot.Login(cc.context, true); err != nil {
					logger.Errorf("[WeChat] Login fail, err: %v", err)
					return
				}
				if err = bot.Start(cc.context); err != nil {
					logger.Errorf("[WeChat] Start bot fail, err: %v", err)
					return
				}
			}
		}
		logger.Warnf("[WeChat] Bot Session Stopped!")
	}
	go loop(&cc.wg)

	return
}
