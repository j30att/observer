package getlist

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

func (h *Handler) Execute() []model.Metrics {
	return h.repo.List()
}
