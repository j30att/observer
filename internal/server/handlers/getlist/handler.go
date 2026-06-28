package getlist

import (
	"context"

	"j30att/observer/internal/server/model"
)

type metricsLister interface {
	List(ctx context.Context) []model.Metrics
}

// Handler lists all stored metrics.
type Handler struct {
	repo metricsLister
}

// New creates a metrics list handler backed by repo.
func New(repo metricsLister) *Handler {
	return &Handler{repo: repo}
}

// Execute returns all stored metrics.
func (h *Handler) Execute(ctx context.Context) []model.Metrics {
	return h.repo.List(ctx)
}
