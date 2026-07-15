package get

import (
	"context"
	"errors"

	"j30att/observer/internal/server/model"
	"j30att/observer/internal/server/repository"
)

// ErrMetricNotFound is returned when a requested metric is absent.
var ErrMetricNotFound = errors.New("metric not found")

type metricsLoader interface {
	Load(ctx context.Context, metricType, name string) (model.Metrics, error)
}

// Handler loads a single metric from storage.
type Handler struct {
	repo metricsLoader
}

// New creates a get metric handler backed by repo.
func New(repo metricsLoader) *Handler {
	return &Handler{repo: repo}
}

// Execute returns one metric by type and name.
func (h *Handler) Execute(ctx context.Context, metricType, name string) (model.Metrics, error) {
	metric, err := h.repo.Load(ctx, metricType, name)
	if err != nil {
		if errors.Is(err, repository.ErrMetricNotFound) {
			return model.Metrics{}, ErrMetricNotFound
		}

		return model.Metrics{}, err
	}

	return metric, nil
}
