package agent

import (
	"context"
	"sync"

	"github.com/er1cw00/claw.go/base/logger"
)

// Service 缓存服务
type Service struct {
	agent *Agent
	wg    sync.WaitGroup
	ctx   context.Context
	stop  context.CancelFunc
}

var agentService *Service = &Service{}

// GetService 获取缓存服务实例
func GetService() *Service {
	return agentService
}

func (s *Service) Start() error {
	logger.Info("Agent Service Start !")
	s.ctx, s.stop = context.WithCancel(context.Background())
	s.agent = NewAgent()
	if err := s.agent.Start(); err != nil {
		return err
	}
	s.wg.Add(1)
	go s.agent.Run(s.ctx, &s.wg)

	return nil
}

// Stop 停止缓存服务
func (s *Service) Stop() {
	logger.Info("Agent Service Stop")
	s.stop()
	s.wg.Wait()
}
