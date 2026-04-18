package main

import (
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
)

func main() {
	cfg, err := config.ParseServerConfig(os.Args[1:])
	if err != nil {
		stdlog.Fatal(err)
	}

	log.Logger = zerolog.New(os.Stdout).Level(zerolog.InfoLevel).With().Timestamp().Logger()

	repo := repository.NewMetricsRepository()
	updateMetricCommand := update.New(repo)
	getMetricQuery := get.New(repo)
	listMetricsQuery := getlist.New(repo)
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
