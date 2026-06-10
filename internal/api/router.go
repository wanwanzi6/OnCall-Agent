package api

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"oncall-agent/internal/api/aiops"
	"oncall-agent/internal/api/chat"
	"oncall-agent/internal/api/health"
	"oncall-agent/internal/api/knowledge"
	"oncall-agent/internal/infra/config"
	"oncall-agent/internal/model/response"
	"oncall-agent/internal/service"
)

type Services struct {
	Chat      *service.ChatService
	Knowledge *service.KnowledgeService
	AIOps     *service.AIOpsService
}

func NewRouter(cfg *config.Config, services Services, log *slog.Logger) *gin.Engine {
	if log == nil {
		log = slog.Default()
	}
	if cfg.App.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(TraceID(), CORS(), AccessLog(log), gin.CustomRecovery(func(c *gin.Context, recovered any) {
		log.ErrorContext(c.Request.Context(), "panic recovered",
			"trace_id", c.GetString("trace_id"),
			"error", recovered,
		)
		response.InternalError(c, "internal server error")
	}))

	group := router.Group("/api")
	health.Register(group, cfg)
	chat.Register(group, services.Chat)
	knowledge.Register(group, services.Knowledge)
	aiops.Register(group, services.AIOps)

	return router
}
