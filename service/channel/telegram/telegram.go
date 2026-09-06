package telegram

import (
	"net/http"
	"strconv"
	"time"

	"github.com/er1cw00/claw.go/base"
	"github.com/er1cw00/claw.go/base/logger"
	"github.com/er1cw00/claw.go/model"
	ch "github.com/er1cw00/claw.go/service/channel/base"
	tele "gopkg.in/telebot.v4"
)

type TelegramChannel struct {
	ch.BaseChannel
	proxy     string
	token     string
	mediaPath string
	bot       *tele.Bot
}

func NewTelegramChannel(proxy, token, mediaPath string) *TelegramChannel {
	return &TelegramChannel{
		proxy:     proxy,
		token:     token,
		mediaPath: mediaPath,
	}
}

func (cc *TelegramChannel) Start() error {
	var (
		err    error        = nil
		client *http.Client = nil
	)

	client = base.NewHttpClient(cc.proxy, 20*time.Second)
	pref := tele.Settings{
		Token:       cc.token,
		Client:      client,
		Synchronous: true,
		Poller:      &tele.LongPoller{Timeout: 60 * time.Second},
	}
	bot, err := tele.NewBot(pref)
	if err != nil {
		logger.Errorf("[Telegram] Channel Start Fail, err: %v", err)
		return err
	}
	bot.Handle(tele.OnText, cc.handleText)
	bot.Handle(tele.OnPhoto, cc.handlePhoto)
	bot.Handle(tele.OnVideo, cc.handleVideo)
	bot.Handle(tele.OnDocument, cc.handleDocument)
	bot.Handle(tele.OnAnimation, cc.handleAnimation)
	bot.Handle(tele.OnEdited, cc.handleEdited)
	cc.bot = bot

	loop := func() {
		logger.Infof("[Telegram] Start...")
		cc.bot.Start()
	}
	go loop()

	return nil
}

func (cc *TelegramChannel) Stop() {
	logger.Infof("[Telegram] Stop")
	if cc.bot != nil {
		cc.bot.Stop()
	}
}
func (cc *TelegramChannel) Name() string {
	return ch.ChannelTelegram
}
func (cc TelegramChannel) SendMessage(msg *model.OutboundMessage) error {
	var (
		err error = nil
	)
	id, err := strconv.ParseInt(msg.ChatID, 10, 0)
	if err != nil {
		return err
	}
	to := &tele.User{ID: id}
	if msg.Content != "" {
		_, err = cc.bot.Send(to, msg.Content)
	}
	if err != nil {
		logger.Errorf("[WeChat] Send message to user fail, err: %v", err)
	}
	return nil
}
