package main

import (
	"context"
	"log"

	"j30att/observer/internal/agent"
	"j30att/observer/internal/agent/collectors"
	"j30att/observer/internal/agent/repository"
	"j30att/observer/internal/agent/senders"
	"j30att/observer/internal/config"
)

func main() {
	cfg := config.NewAgentConfig()
	store := repository.NewMetricsRepository()
	collector := collectors.NewRuntimeCollector()
	sender := senders.NewHTTPSender(cfg.ServerAddress)
	app := agent.New(cfg, store, collector, sender)

	if err := app.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
