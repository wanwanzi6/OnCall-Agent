package api

import (
	"github.com/gin-gonic/gin"

	"oncall-agent/internal/api/aiops"
	"oncall-agent/internal/api/chat"
	"oncall-agent/internal/api/health"
	"oncall-agent/internal/api/knowledge"
	"oncall-agent/internal/infra/config"
	"oncall-agent/internal/service"
)

type Services struct {
	Chat      *service.ChatService
	Knowledge *service.KnowledgeService
	AIOps     *service.AIOpsService
}

func NewRouter(cfg *config.Config, services Services) *gin.Engine {
	if cfg.App.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), TraceID())

	group := router.Group("/api")
	health.Register(group, cfg)
	chat.Register(group, services.Chat)
	knowledge.Register(group, services.Knowledge)
	aiops.Register(group, services.AIOps)

	return router
}
