package cache

import (
	"time"

	"github.com/er1cw00/claw.go/base/logger"
	"github.com/patrickmn/go-cache"
)

const NoExpiration = cache.NoExpiration

// Service 缓存服务
type Service struct {
	client *cache.Cache
}

var cacheService *Service = &Service{}

// GetService 获取缓存服务实例
func GetService() *Service {
	return cacheService
}

// Start 启动缓存服务
// defaultExpiration: 默认过期时间，cleanupInterval: 清理间隔
func (s *Service) Start() error {
	s.client = cache.New(10*time.Minute, 15*time.Minute)
	logger.Info("[Cache] Service Start !")
	return nil
}

// Stop 停止缓存服务
func (s *Service) Stop() {
	if s.client != nil {
		s.client.Flush()
	}
	logger.Info("[Cache] Service Stop")
}

func (s *Service) Name() string {
	return "Cache"
}

// Set 设置缓存，key为string类型
func (s *Service) Set(key string, val interface{}, expires time.Duration) {
	if s.client == nil {
		logger.Warn("[Cache] client is not initialized")
		return
	}
	s.client.Set(key, val, expires)
}

// Get 获取缓存，key为string类型，返回value和是否存在
func (s *Service) Get(key string) (interface{}, bool) {
	if s.client == nil {
		logger.Warn("[Cache] client is not initialized")
		return nil, false
	}
	return s.client.Get(key)
}

// Delete 删除缓存，key为string类型
func (s *Service) Delete(key string) {
	if s.client == nil {
		logger.Warn("[Cache] client is not initialized")
		return
	}
	s.client.Delete(key)
}

// Flush 清空所有缓存
func (s *Service) Flush() {
	if s.client == nil {
		logger.Warn("[Cache] client is not initialized")
		return
	}
	s.client.Flush()
}
