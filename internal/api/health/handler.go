package health

import (
	"github.com/gin-gonic/gin"

	"oncall-agent/internal/infra/config"
	"oncall-agent/internal/model/response"
)

func Register(router *gin.RouterGroup, cfg *config.Config) {
	router.GET("/health", func(c *gin.Context) {
		response.OK(c, gin.H{
			"status": "ok",
			"env":    cfg.App.Env,
			"mock":   cfg.Mock.Enabled,
		})
	})
}
