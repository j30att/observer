package get

import (
	"j30att/observer/internal/server/model"
	"j30att/observer/internal/server/repository"
)

type Handler struct {
	repo repository.MetricsRepository
}

func New(repo repository.MetricsRepository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) Execute(metricType, name string) (model.Metrics, error) {
	return h.repo.Load(metricType, name)
}
