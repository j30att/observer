package main

import (
	"context"
	"crypto/rsa"
	"os"

	"github.com/rs/zerolog"
	"j30att/observer/internal/agent"
	"j30att/observer/internal/agent/collectors"
	"j30att/observer/internal/agent/repository"
	"j30att/observer/internal/agent/senders"
	"j30att/observer/internal/buildinfo"
	"j30att/observer/internal/config"
	"j30att/observer/internal/encryption"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	buildinfo.Print(os.Stdout, buildVersion, buildDate, buildCommit)

	logger := zerolog.New(os.Stdout).Level(zerolog.InfoLevel).With().Timestamp().Logger()

	cfg, err := config.ParseAgentConfig(os.Args[1:])
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to parse agent config")
	}

	var publicKey *rsa.PublicKey
	if cfg.CryptoKey != "" {
		publicKey, err = encryption.LoadPublicKey(cfg.CryptoKey)
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to load public encryption key")
		}
	}
	store := repository.NewMetricsRepository()
	metricCollectors := []agent.Collector{
		collectors.NewRuntimeCollector(),
		collectors.NewGopsutilCollector(),
	}
	sender := senders.NewHTTPSenderWithOptions(cfg.ServerAddress, senders.HTTPSenderOptions{
		SignatureKey: cfg.Key,
		PublicKey:    publicKey,
	})
	app := agent.New(cfg, store, metricCollectors, sender, logger)

	if err := app.Run(context.Background()); err != nil {
		logger.Fatal().Err(err).Msg("agent stopped")
	}
}
