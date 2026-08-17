package main

import (
	"context"
	"crypto/rsa"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
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

const gracefulShutdownTimeout = 30 * time.Second

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	buildinfo.Print(os.Stdout, buildVersion, buildDate, buildCommit)
	logger := zerolog.New(os.Stdout).Level(zerolog.InfoLevel).With().Timestamp().Logger()

	if err := runServer(os.Args[1:], logger); err != nil {
		logger.Error().Err(err).Msg("server stopped with an error")
		os.Exit(1)
	}

	logger.Info().Msg("server stopped gracefully")
}

func runServer(args []string, logger zerolog.Logger) (runErr error) {
	cfg, err := config.ParseServerConfig(args)
	if err != nil {
		return fmt.Errorf("parse server config: %w", err)
	}

	var privateKey *rsa.PrivateKey
	if cfg.CryptoKey != "" {
		privateKey, err = encryption.LoadPrivateKey(cfg.CryptoKey)
		if err != nil {
			return fmt.Errorf("load private encryption key: %w", err)
		}
	}

	var db *sql.DB
	if cfg.DatabaseDSN != "" {
		db, err = sql.Open("postgres", cfg.DatabaseDSN)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer func() {
			if err := db.Close(); err != nil {
				runErr = errors.Join(runErr, fmt.Errorf("close database: %w", err))
			}
		}()

		pingCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		err = db.PingContext(pingCtx)
		cancel()
		if err != nil {
			logger.Error().Err(err).Msg("database is unavailable")
		} else if err := migrations.Up(db); err != nil {
			return fmt.Errorf("apply database migrations: %w", err)
		}
	}

	var periodicSaver *storage.PeriodicSaver
	var periodicSaverCancel context.CancelFunc
	var periodicSaverWG sync.WaitGroup

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
				return fmt.Errorf("restore metrics from file: %w", err)
			}

			if err := repo.Restore(metrics); err != nil {
				return fmt.Errorf("restore metrics repository: %w", err)
			}

			logger.Info().Int("metrics_count", len(metrics)).Msg("restored metrics")
		} else {
			fileStorage = storage.NewFileStorage(cfg.FileStoragePath, logger)
		}

		metricsRepo = repo
		if cfg.StoreInterval > 0 {
			var saverCtx context.Context
			saverCtx, periodicSaverCancel = context.WithCancel(context.Background())
			periodicSaver = storage.NewPeriodicSaver(cfg.StoreInterval, repo, fileStorage, logger)
			periodicSaverWG.Add(1)
			go func() {
				defer periodicSaverWG.Done()
				periodicSaver.Run(saverCtx)
			}()
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
	defer func() {
		if auditor == nil {
			return
		}

		if err := auditor.Close(); err != nil {
			runErr = errors.Join(runErr, fmt.Errorf("close audit subject: %w", err))
		}
	}()

	metricController := controller.NewMetricController(updateMetricCommand, getMetricQuery, listMetricsQuery, auditor)
	r := router.NewRouterWithOptions(metricController, logger, db, router.Options{
		SignatureKey:  cfg.Key,
		PrivateKey:    privateKey,
		TrustedSubnet: cfg.TrustedSubnet,
	})

	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	shutdownCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	serveErr := serveUntilShutdown(shutdownCtx, server, server.ListenAndServe, gracefulShutdownTimeout)
	shutdownErr := serveErr

	if err := stopPeriodicSaver(periodicSaverCancel, &periodicSaverWG, periodicSaver); err != nil {
		shutdownErr = errors.Join(shutdownErr, fmt.Errorf("save metrics during shutdown: %w", err))
	}

	if shutdownErr != nil {
		return shutdownErr
	}

	return nil
}

func serveUntilShutdown(ctx context.Context, server *http.Server, serve func() error, shutdownTimeout time.Duration) error {
	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- serve()
	}()

	select {
	case err := <-serveErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}

	gracefulCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(gracefulCtx); err != nil {
		if closeErr := server.Close(); closeErr != nil {
			return errors.Join(fmt.Errorf("shutdown HTTP server: %w", err), fmt.Errorf("close HTTP server: %w", closeErr))
		}
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}

	if err := <-serveErrors; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func stopPeriodicSaver(cancel context.CancelFunc, wg *sync.WaitGroup, saver *storage.PeriodicSaver) error {
	if cancel == nil || saver == nil {
		return nil
	}

	cancel()
	wg.Wait()

	return saver.Save(context.Background())
}
