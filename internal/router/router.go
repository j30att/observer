package router

import (
	"net/http"

	"j30att/observer/internal/controller"
)

func NewRouter(metricController *controller.MetricController) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /update/{type}/{name}/{value}", metricController.UpdateMetric)
	return mux
}
