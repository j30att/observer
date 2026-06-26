package main

import (
	"context"
	"os"

	"github.com/rs/zerolog"
	"j30att/observer/internal/agent"
	"j30att/observer/internal/agent/collectors"
	"j30att/observer/internal/agent/repository"
	"j30att/observer/internal/agent/senders"
	"j30att/observer/internal/config"
)

func main() {
	logger := zerolog.New(os.Stdout).Level(zerolog.InfoLevel).With().Timestamp().Logger()

	cfg, err := config.ParseAgentConfig(os.Args[1:])
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to parse agent config")
	}
	store := repository.NewMetricsRepository()
	metricCollectors := []agent.Collector{
		collectors.NewRuntimeCollector(),
		collectors.NewGopsutilCollector(),
	}
	sender := senders.NewHTTPSender(cfg.ServerAddress, cfg.Key)
	app := agent.New(cfg, store, metricCollectors, sender, logger)

	if err := app.Run(context.Background()); err != nil {
		logger.Fatal().Err(err).Msg("agent stopped")
	}
}
