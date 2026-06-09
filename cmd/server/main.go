package main

import (
	"fmt"
	"log/slog"

	"oncall-agent/internal/api"
	"oncall-agent/internal/infra/config"
	"oncall-agent/internal/infra/logger"
	"oncall-agent/internal/service"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
	}
}

func run() error {
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	log := logger.New(cfg.App.Env)

	services := api.Services{
		Chat:      service.NewChatService(cfg.Mock.Enabled, log),
		Knowledge: service.NewKnowledgeService(cfg.Mock.Enabled, cfg.Knowledge, log),
		AIOps:     service.NewAIOpsService(log),
	}

	router := api.NewRouter(cfg, services, log)
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	if err := router.Run(addr); err != nil {
		return fmt.Errorf("run server: %w", err)
	}
	return nil
}
