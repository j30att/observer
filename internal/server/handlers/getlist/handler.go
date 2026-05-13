package getlist

import (
	"context"

	"j30att/observer/internal/server/model"
)

type metricsLister interface {
	List(ctx context.Context) []model.Metrics
}

type Handler struct {
	repo metricsLister
}

func New(repo metricsLister) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) Execute(ctx context.Context) []model.Metrics {
	return h.repo.List(ctx)
}
