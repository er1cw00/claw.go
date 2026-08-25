package api

import (
	"fmt"

	"github.com/er1cw00/claw.go/base/logger"
	"github.com/gin-gonic/gin"
)

func Start(host string, port int) error {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	addr := fmt.Sprintf("%s:%d", host, port)
	logger.Debugf("listen: %s", addr)
	return r.Run(addr)
}
