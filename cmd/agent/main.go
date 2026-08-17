package main

import (
	"context"
	"crypto/rsa"
	"errors"
	"os"
	"os/signal"
	"syscall"

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
	var sender agent.Sender
	if cfg.GRPCAddress != "" {
		grpcSender, err := senders.NewGRPCSender(cfg.GRPCAddress)
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to create grpc sender")
		}
		defer func() {
			if err := grpcSender.Close(); err != nil {
				logger.Error().Err(err).Msg("failed to close grpc sender")
			}
		}()
		sender = grpcSender
	} else {
		sender = senders.NewHTTPSenderWithOptions(cfg.ServerAddress, senders.HTTPSenderOptions{
			SignatureKey: cfg.Key,
			PublicKey:    publicKey,
		})
	}
	app := agent.New(cfg, store, metricCollectors, sender, logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	if err := app.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Fatal().Err(err).Msg("agent stopped")
	}

	logger.Info().Msg("agent stopped gracefully")
}
