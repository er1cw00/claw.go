package channel

import (
	"context"
	"path/filepath"
	"sync"

	"github.com/er1cw00/claw.go/base"
	"github.com/er1cw00/claw.go/base/logger"
	//"github.com/er1cw00/claw.go/model"
	"github.com/er1cw00/claw.go/service/bus"
	ch "github.com/er1cw00/claw.go/service/channel/base"
	"github.com/er1cw00/claw.go/service/channel/telegram"
	"github.com/er1cw00/claw.go/service/channel/wechat"
)

type Service struct {
	mu       sync.RWMutex
	wg       sync.WaitGroup
	context  context.Context
	stop     context.CancelFunc
	channels map[string]ch.Channel
}

var chService *Service = &Service{
	channels: make(map[string]ch.Channel),
}

func GetService() *Service {
	return chService
}

func (s *Service) Start() error {
	var (
		err error      = nil
		cc  ch.Channel = nil
	)
	channels := base.GetSettings().Channels
	s.context, s.stop = context.WithCancel(context.Background())

	s.mu.Lock()
	defer s.mu.Unlock()
	workspace := base.GetSettings().Workspace
	mediaPath := base.GetSettings().MediaPath

	if channels.Telegram.Enable {
		cc = telegram.NewTelegramChannel(channels.Telegram.Proxy, channels.Telegram.Token, channels.Telegram.Allow, mediaPath)
		if err = cc.Start(); err != nil {
			return err
		}
		s.channels[cc.Name()] = cc
	}
	if channels.WeChat.Enable {
		storage := filepath.Join(workspace, "wechat")
		cc = wechat.NewWeChatChannel(storage, mediaPath)
		if err = cc.Start(); err != nil {
			return err
		}
		s.channels[cc.Name()] = cc
	}
	s.wg.Add(1)
	go s.dispatch(s.context, &s.wg)
	return err
}

func (s *Service) Stop() {

	s.stop()
	for _, c := range s.channels {
		c.Stop()
	}
	s.wg.Wait()
}

func (s *Service) Name() string {
	return "Channel"
}

func (s *Service) dispatch(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	msgBus := bus.GetService().GetMessageBus()
	for {
		select {
		case <-ctx.Done():
			logger.Infof("[Channel] Outbound dispatcher stopped")
			return

		case msg, ok := <-msgBus.ConsumeOutbound():
			if !ok {
				logger.Infof("[Channel] Outbound dispatcher stopped")
				return
			}
			if msg.Channel == "all" {
				s.mu.RLock()
				for _, channel := range s.channels {
					channel.ProcessOutbountMessage(&msg)
				}
				s.mu.RUnlock()
			} else {
				s.mu.RLock()
				channel, found := s.channels[msg.Channel]
				s.mu.RUnlock()
				if found {
					channel.ProcessOutbountMessage(&msg)
				} else {
					logger.Errorf("channel(%s) not found", msg.Channel)
				}
			}
		}
	}
}
