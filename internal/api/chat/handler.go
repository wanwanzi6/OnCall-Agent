package chat

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"oncall-agent/internal/model/request"
	"oncall-agent/internal/model/response"
	"oncall-agent/internal/service"
)

func Register(router *gin.RouterGroup, chatService *service.ChatService) {
	router.POST("/chat", func(c *gin.Context) {
		var req request.ChatRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "message is required")
			return
		}

		response.OK(c, chatService.Chat(req.Message))
	})

	router.POST("/chat/stream", func(c *gin.Context) {
		var req request.ChatRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "message is required")
			return
		}

		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Status(http.StatusOK)
		for _, chunk := range chatService.StreamChat(req.Message) {
			c.SSEvent("message", chunk)
			c.Writer.Flush()
			time.Sleep(50 * time.Millisecond)
		}
	})
}
