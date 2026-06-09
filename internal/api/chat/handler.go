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

		result, err := chatService.Chat(c.Request.Context(), req.Message)
		if err != nil {
			response.InternalError(c, err.Error())
			return
		}
		response.OK(c, result)
	})

	router.POST("/chat/stream", func(c *gin.Context) {
		var req request.ChatRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "message is required")
			return
		}

		chunks, err := chatService.StreamChat(c.Request.Context(), req.Message)
		if err != nil {
			response.InternalError(c, err.Error())
			return
		}

		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Status(http.StatusOK)
		for _, chunk := range chunks {
			c.SSEvent("message", chunk)
			c.Writer.Flush()
			time.Sleep(50 * time.Millisecond)
		}
	})
}
