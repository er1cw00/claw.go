package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/er1cw00/claw.go/api"
	"github.com/er1cw00/claw.go/base"
	"github.com/er1cw00/claw.go/base/logger"
	"github.com/er1cw00/claw.go/service"
)

func main() {
	var configPath string
	var err error = nil
	flag.StringVar(&configPath, "c", "goclaw.yaml", "config yaml file")
	flag.Parse()

	if err := base.Start(configPath); err != nil {
		logger.Error(err)
		os.Exit(1)
	}
	logger.Infof("version:     %s", _version)
	logger.Infof("git branch:  %s", _gitBranch)
	logger.Infof("build time:  %s", _buildTime)
	logger.Info("🔌 claw.go start...")

	if err = service.Start(); err != nil {
		logger.Errorf("❌ Service 启动失败, err: %v", err)
		return
	}
	go func() {
		webConfig := base.GetSettings().Web
		if err := api.Start(webConfig.Host, webConfig.Port); err != nil {
			logger.Errorf("❌ API启动失败, err: %v", err)
		}
	}()

	logger.Info("✅ System started successfully, waiting for trading commands...")
	logger.Info("📌 Tip: Use Ctrl+C to stop the system")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan

	logger.Warnf("📴 收到退出信号: %v，开始清理并退出...\n", sig)
	api.Stop()
	service.Stop()

	logger.Info("监控退出")
}
