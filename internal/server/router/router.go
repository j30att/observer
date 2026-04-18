package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"j30att/observer/internal/server/controller"
	"j30att/observer/internal/server/middlewares"
)

func NewRouter(metricController *controller.MetricController) http.Handler {
	r := chi.NewRouter()
	r.Use(middlewares.Logger)
	r.Post("/update/{type}/{name}/{value}", metricController.UpdateMetric)
	r.Get("/value/{type}/{name}", metricController.GetMetric)
	r.Get("/", metricController.ListMetrics)
	return r
}
