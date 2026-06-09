package main

import (
	"fmt"
	"log"

	"oncall-agent/internal/api"
	"oncall-agent/internal/infra/config"
	"oncall-agent/internal/service"
)

func main() {
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	services := api.Services{
		Chat:      service.NewChatService(cfg.Mock.Enabled),
		Knowledge: service.NewKnowledgeService(cfg.Mock.Enabled),
		AIOps:     service.NewAIOpsService(),
	}

	router := api.NewRouter(cfg, services)
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	if err := router.Run(addr); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
