package service

import (
	"github.com/er1cw00/claw.go/base/logger"
	"github.com/er1cw00/claw.go/service/agent"
	"github.com/er1cw00/claw.go/service/bus"
	cc "github.com/er1cw00/claw.go/service/cache"
	ch "github.com/er1cw00/claw.go/service/channel"
	ss "github.com/er1cw00/claw.go/service/session"
	to "github.com/er1cw00/claw.go/service/tools"
)

type Service interface {
	Start() error
	Stop()
	Name() string
}

var services []Service = []Service{
	cc.GetService(),
	bus.GetService(),
	ch.GetService(),
	to.GetService(),
	ss.GetService(),
	agent.GetService(),
}

func Start() error {
	var err error = nil

	for _, s := range services {
		if err = s.Start(); err != nil {
			logger.Errorf("❌ %s Service Start fail, err: %v", s.Name(), err)
			break
		}
	}
	return err
}

func Stop() {
	for _, s := range services {
		s.Stop()
	}
}
