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

// NewRouter wires the HTTP routes, middlewares, controller, and health check.
func NewRouter(metricController *controller.MetricController, logger zerolog.Logger, db ping.Pinger, key ...string) http.Handler {
	signatureKey := ""
	if len(key) > 0 {
		signatureKey = key[0]
	}

	r := chi.NewRouter()
	r.Use(chimiddleware.StripSlashes)
	r.Use(middlewares.Logger(logger))
	if signatureKey != "" {
		r.Use(middlewares.Signature(signatureKey))
	}
	r.Use(middlewares.Gzip)
	pingHandler := ping.New(db)
	r.Get("/ping", pingHandler.Ping)
	r.Post("/updates", metricController.UpdateMetricsJSON)
	r.Post("/update", metricController.UpdateMetricJSON)
	r.Post("/value", metricController.GetMetricJSON)
	r.Post("/update/{type}/{name}/{value}", metricController.UpdateMetric)
	r.Get("/value/{type}/{name}", metricController.GetMetric)
	r.Get("/", metricController.ListMetrics)
	return r
}
