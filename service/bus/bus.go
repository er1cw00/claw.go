package bus

import (
	"github.com/er1cw00/claw.go/base/logger"
)

// Service 缓存服务
type Service struct {
	evBus  *EventBus
	msgBus *MessageBus
}

var busService *Service = &Service{}

// GetService 获取缓存服务实例
func GetService() *Service {
	return busService
}

func (s *Service) Start() error {
	s.evBus = NewEventBus()
	s.msgBus = NewMessageBus()
	logger.Info("[Bus] Service Start !")
	return nil
}

func (s *Service) Stop() {
	logger.Info("[Bus] Service Stop")
}

func (s *Service) Name() string {
	return "Bus"
}

func (s *Service) GetEventBus() *EventBus {
	return s.evBus
}

func (s *Service) GetMessageBus() *MessageBus {
	return s.msgBus
}
