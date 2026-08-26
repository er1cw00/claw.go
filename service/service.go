package service

import (
	"github.com/er1cw00/claw.go/base/logger"
	"github.com/er1cw00/claw.go/service/agent"
	"github.com/er1cw00/claw.go/service/bus"
	cc "github.com/er1cw00/claw.go/service/cache"
	ch "github.com/er1cw00/claw.go/service/channel"
	ss "github.com/er1cw00/claw.go/service/session"
)

func Start() error {
	var err error = nil

	if err = cc.GetService().Start(); err != nil {
		logger.Errorf("❌ Cache Service Start fail, err: %v", err)
		return err
	}
	if err = bus.GetService().Start(); err != nil {
		logger.Errorf("❌ Bus Service Start fail, err: %v", err)
		return err
	}
	if err = ch.GetService().Start(); err != nil {
		logger.Errorf("❌ Channels Service Start fail, err: %v", err)
		return err
	}
	if err = ss.GetService().Start(); err != nil {
		logger.Errorf("❌ Session Service Start fail, err: %v", err)
		return err
	}
	if err = agent.GetService().Start(); err != nil {
		logger.Errorf("❌ Agent Service Start fail, err: %v", err)
		return err
	}
	return err
}

func Stop() {

	ch.GetService().Stop()
	bus.GetService().Stop()
	cc.GetService().Stop()
	agent.GetService().Stop()
}
