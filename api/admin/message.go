package admin

import (
	"net/http"

	"github.com/er1cw00/claw.go/base/logger"
	"github.com/er1cw00/claw.go/model"
	bus "github.com/er1cw00/claw.go/service/bus"
	"github.com/gin-gonic/gin"
)

/*
	curl -X POST http://127.0.0.1:8080/admin/message \
		-H "Content-Type: application/json" \
	    -d '{"channel": "telegram", "chat_id": "6412449819", "content": "hello from api"}'
*/
type MessageController struct {
	bus *bus.Service
}

type postMessageRequest struct {
	Channel string `json:"channel" binding:"required"`
	ChatID  string `json:"chat_id" binding:"required"`
	Content string `json:"content" binding:"required"`
}

type postMessageResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func newMessageController() *MessageController {
	return &MessageController{
		bus: bus.GetService(),
	}
}

func Setup(g *gin.RouterGroup) {
	c := newMessageController()
	g.POST("/message", c.PostMessage)
}

func (c *MessageController) PostMessage(ctx *gin.Context) {
	var req postMessageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, postMessageResponse{
			Code:    1,
			Message: err.Error(),
		})
		return
	}
	logger.Debugf("[API] send message to channel(%s), chat(%s)", req.Channel, req.ChatID)
	c.bus.GetMessageBus().PublishOutbound(model.OutboundMessage{
		Channel: req.Channel,
		ChatID:  req.ChatID,
		Content: req.Content,
	})

	ctx.JSON(http.StatusOK, postMessageResponse{
		Code:    0,
		Message: "success",
	})
}
