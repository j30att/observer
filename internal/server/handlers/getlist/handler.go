package getlist

import (
	"j30att/observer/internal/server/model"
)

type metricsLister interface {
	List() []model.Metrics
}

type Handler struct {
	repo metricsLister
}

func New(repo metricsLister) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) Execute() []model.Metrics {
	return h.repo.List()
}
