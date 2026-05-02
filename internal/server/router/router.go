package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"j30att/observer/internal/server/controller"
	"j30att/observer/internal/server/handlers/ping"
	"j30att/observer/internal/server/middlewares"
)

func NewRouter(metricController *controller.MetricController, logger zerolog.Logger, db ...ping.Pinger) http.Handler {
	r := chi.NewRouter()
	r.Use(chimiddleware.StripSlashes)
	r.Use(middlewares.Logger(logger))
	r.Use(middlewares.Gzip)
	var dbPinger ping.Pinger
	if len(db) > 0 {
		dbPinger = db[0]
	}
	pingHandler := ping.New(dbPinger)
	r.Get("/ping", pingHandler.Ping)
	r.Post("/update", metricController.UpdateMetricJSON)
	r.Post("/value", metricController.GetMetricJSON)
	r.Post("/update/{type}/{name}/{value}", metricController.UpdateMetric)
	r.Get("/value/{type}/{name}", metricController.GetMetric)
	r.Get("/", metricController.ListMetrics)
	return r
}
