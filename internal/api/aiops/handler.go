package aiops

import (
	"github.com/gin-gonic/gin"

	"oncall-agent/internal/model/request"
	"oncall-agent/internal/model/response"
	"oncall-agent/internal/service"
)

func Register(router *gin.RouterGroup, aiopsService *service.AIOpsService) {
	router.POST("/aiops/analyze", func(c *gin.Context) {
		var req request.AnalyzeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "invalid analyze request")
			return
		}

		response.OK(c, aiopsService.Analyze(req.AlertName, req.Service))
	})
}
