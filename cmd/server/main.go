package main

import (
	"context"
	stdlog "log"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"j30att/observer/internal/config"
	"j30att/observer/internal/server/controller"
	"j30att/observer/internal/server/handlers/get"
	"j30att/observer/internal/server/handlers/getlist"
	"j30att/observer/internal/server/handlers/update"
	"j30att/observer/internal/server/repository"
	"j30att/observer/internal/server/router"
	"j30att/observer/internal/server/storage"
)

func main() {
	cfg, err := config.ParseServerConfig(os.Args[1:])
	if err != nil {
		stdlog.Fatal(err)
	}

	log.Logger = zerolog.New(os.Stdout).Level(zerolog.InfoLevel).With().Timestamp().Logger()

	repo := repository.NewMetricsRepository()
	fileStorage := storage.NewFileStorage(cfg.FileStoragePath)
	if cfg.Restore {
		metrics, err := fileStorage.Load()
		if err != nil {
			stdlog.Fatal(err)
		}

		if err := repo.Restore(metrics); err != nil {
			stdlog.Fatal(err)
		}

		log.Info().Int("metrics_count", len(metrics)).Msg("restored metrics")
	}

	var metricsRepo repository.MetricsRepository = repo
	if cfg.StoreInterval > 0 {
		periodicSaver := storage.NewPeriodicSaver(cfg.StoreInterval, repo, fileStorage, func(err error) {
			log.Error().Err(err).Msg("failed to save metrics")
		})
		go periodicSaver.Run(context.Background())
	} else {
		metricsRepo = repository.NewSyncPersistentRepository(repo, fileStorage)
	}

	updateMetricCommand := update.New(metricsRepo)
	getMetricQuery := get.New(metricsRepo)
	listMetricsQuery := getlist.New(metricsRepo)
	metricController := controller.NewMetricController(updateMetricCommand, getMetricQuery, listMetricsQuery)
	r := router.NewRouter(metricController)

	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		stdlog.Fatal(err)
	}
}
