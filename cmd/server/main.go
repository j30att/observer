package main

import (
	"context"
	"crypto/rsa"
	"database/sql"
	stdlog "log"
	"net/http"
	"os"
	"time"

	"j30att/observer/internal/buildinfo"
	"j30att/observer/internal/config"
	"j30att/observer/internal/encryption"
	"j30att/observer/internal/server/audit"
	"j30att/observer/internal/server/controller"
	"j30att/observer/internal/server/handlers/get"
	"j30att/observer/internal/server/handlers/getlist"
	"j30att/observer/internal/server/handlers/update"
	"j30att/observer/internal/server/model"
	"j30att/observer/internal/server/repository"
	"j30att/observer/internal/server/router"
	"j30att/observer/internal/server/storage"
	"j30att/observer/migrations"

	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	buildinfo.Print(os.Stdout, buildVersion, buildDate, buildCommit)

	cfg, err := config.ParseServerConfig(os.Args[1:])
	if err != nil {
		stdlog.Fatal(err)
	}

	logger := zerolog.New(os.Stdout).Level(zerolog.InfoLevel).With().Timestamp().Logger()
	var privateKey *rsa.PrivateKey
	if cfg.CryptoKey != "" {
		privateKey, err = encryption.LoadPrivateKey(cfg.CryptoKey)
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to load private encryption key")
		}
	}

	var db *sql.DB
	if cfg.DatabaseDSN != "" {
		db, err = sql.Open("postgres", cfg.DatabaseDSN)
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to open database")
		}
		defer db.Close()

		pingCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		err = db.PingContext(pingCtx)
		cancel()
		if err != nil {
			logger.Error().Err(err).Msg("database is unavailable")
		} else if err := migrations.Up(db); err != nil {
			logger.Fatal().Err(err).Msg("failed to apply database migrations")
		}
	}

	var metricsRepo repository.MetricsRepository
	if db != nil {
		metricsRepo = repository.NewPostgresMetricsRepository(db)
	} else if cfg.FileStoragePath != "" {
		repo := repository.NewMetricsRepository()
		var fileStorage *storage.FileStorage
		if cfg.Restore {
			var metrics []model.Metrics
			fileStorage, metrics, err = storage.NewRestoredFileStorage(cfg.FileStoragePath, logger)
			if err != nil {
				logger.Fatal().Err(err).Msg("failed to restore metrics from file")
			}

			if err := repo.Restore(metrics); err != nil {
				logger.Fatal().Err(err).Msg("failed to restore metrics repository")
			}

			logger.Info().Int("metrics_count", len(metrics)).Msg("restored metrics")
		} else {
			fileStorage = storage.NewFileStorage(cfg.FileStoragePath, logger)
		}

		metricsRepo = repo
		if cfg.StoreInterval > 0 {
			periodicSaver := storage.NewPeriodicSaver(cfg.StoreInterval, repo, fileStorage, logger)
			go periodicSaver.Run(context.Background())
		} else {
			metricsRepo = repository.NewSyncPersistentRepository(repo, fileStorage)
		}
	} else {
		metricsRepo = repository.NewMetricsRepository()
	}

	updateMetricCommand := update.New(metricsRepo)
	getMetricQuery := get.New(metricsRepo)
	listMetricsQuery := getlist.New(metricsRepo)

	var auditor *audit.Subject
	if cfg.AuditFile != "" || cfg.AuditURL != "" {
		observers := make([]audit.Observer, 0, 2)
		if cfg.AuditFile != "" {
			observers = append(observers, audit.NewFileObserver(cfg.AuditFile))
		}
		if cfg.AuditURL != "" {
			observers = append(observers, audit.NewURLObserver(cfg.AuditURL))
		}
		auditor = audit.NewSubject(logger, observers...)
	}
	closeAuditor := func() {
		if auditor == nil {
			return
		}

		if err := auditor.Close(); err != nil {
			logger.Error().Err(err).Msg("failed to close audit subject")
		}
	}
	defer closeAuditor()

	metricController := controller.NewMetricController(updateMetricCommand, getMetricQuery, listMetricsQuery, auditor)
	r := router.NewRouterWithOptions(metricController, logger, db, router.Options{
		SignatureKey: cfg.Key,
		PrivateKey:   privateKey,
	})

	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		closeAuditor()
		logger.Fatal().Err(err).Msg("server stopped")
	}
}
