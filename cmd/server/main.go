package main

import (
	"log"
	"net/http"

	"j30att/observer/internal/config"
	"j30att/observer/internal/server/controller"
	"j30att/observer/internal/server/handler"
	"j30att/observer/internal/server/repository"
	"j30att/observer/internal/server/router"
)

func main() {
	cfg := config.NewServerConfig()
	repo := repository.NewMetricsRepository()
	updateMetricCommand := handler.NewUpdateMetricHandler(repo)
	metricController := controller.NewMetricController(updateMetricCommand)
	r := router.NewRouter(metricController)

	if err := http.ListenAndServe(cfg.Address, r); err != nil {
		log.Fatal(err)
	}
}
