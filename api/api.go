package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/er1cw00/claw.go/api/admin"
	"github.com/er1cw00/claw.go/base/logger"
	"github.com/gin-gonic/gin"
)

var server *http.Server = nil

func Start(host string, port int) error {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	g := r.Group("/admin")
	admin.Setup(g)

	addr := fmt.Sprintf("%s:%d", host, port)
	logger.Debugf("[API] listen: %s", addr)

	server = &http.Server{
		Addr:    addr,
		Handler: r,
	}

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func Stop() {
	logger.Debugf("[API] Stop HTTP Server ...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // 设置优雅退出的超时时间（例如 5 秒），防止未处理完的请求无限阻塞
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Errorf("[API] Stop HTTP Server fail, Err: %v", err)
		return
	}
	logger.Debugf("[API] HTTP Server stopped")
}
