package main

import (
	"log"
	"net/http"

	"j30att/observer/internal/commands"
	"j30att/observer/internal/config"
	"j30att/observer/internal/controller"
	"j30att/observer/internal/repository"
	"j30att/observer/internal/router"
)

func main() {
	cfg := config.NewServerConfig()
	repo := repository.NewMetricsRepository()
	updateMetricCommand := commands.NewUpdateMetricCommand(repo)
	metricController := controller.NewMetricController(updateMetricCommand)
	r := router.NewRouter(metricController)

	if err := http.ListenAndServe(cfg.Address, r); err != nil {
		log.Fatal(err)
	}
}
