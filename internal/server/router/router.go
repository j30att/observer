package router

import (
	"crypto/rsa"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"j30att/observer/internal/server/controller"
	"j30att/observer/internal/server/handlers/ping"
	"j30att/observer/internal/server/middlewares"
)

// Options contains optional HTTP transport security settings.
type Options struct {
	SignatureKey  string
	PrivateKey    *rsa.PrivateKey
	TrustedSubnet string
}

// NewRouter wires the HTTP routes, middlewares, controller, and health check.
func NewRouter(metricController *controller.MetricController, logger zerolog.Logger, db ping.Pinger, key ...string) http.Handler {
	var signatureKey string
	if len(key) > 0 {
		signatureKey = key[0]
	}

	return NewRouterWithOptions(metricController, logger, db, Options{SignatureKey: signatureKey})
}

// NewRouterWithOptions wires the HTTP routes with signing and encryption settings.
func NewRouterWithOptions(metricController *controller.MetricController, logger zerolog.Logger, db ping.Pinger, opts Options) http.Handler {
	r := chi.NewRouter()
	r.Use(chimiddleware.StripSlashes)
	r.Use(middlewares.Logger(logger))
	r.Use(middlewares.TrustedSubnetForMetricUpdates(opts.TrustedSubnet))
	if opts.SignatureKey != "" {
		r.Use(middlewares.Signature(opts.SignatureKey))
	}
	if opts.PrivateKey != nil {
		r.Use(middlewares.Decryption(opts.PrivateKey))
	}
	r.Use(middlewares.Gzip)
	pingHandler := ping.New(db)
	r.Get("/ping", pingHandler.Ping)
	r.Post("/value", metricController.GetMetricJSON)
	r.Get("/value/{type}/{name}", metricController.GetMetric)
	r.Get("/", metricController.ListMetrics)
	r.Post("/updates", metricController.UpdateMetricsJSON)
	r.Post("/update", metricController.UpdateMetricJSON)
	r.Post("/update/{type}/{name}/{value}", metricController.UpdateMetric)
	return r
}
