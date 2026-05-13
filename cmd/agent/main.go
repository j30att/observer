package main

import (
	"context"
	"log"
	"os"

	"j30att/observer/internal/agent"
	"j30att/observer/internal/agent/collectors"
	"j30att/observer/internal/agent/repository"
	"j30att/observer/internal/agent/senders"
	"j30att/observer/internal/config"
)

func main() {
	cfg, err := config.ParseAgentConfig(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}
	store := repository.NewMetricsRepository()
	collector := collectors.NewRuntimeCollector()
	sender := senders.NewHTTPSender(cfg.ServerAddress, cfg.Key)
	app := agent.New(cfg, store, collector, sender)

	if err := app.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
