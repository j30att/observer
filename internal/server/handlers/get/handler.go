package get

import (
	"errors"

	"j30att/observer/internal/server/model"
	"j30att/observer/internal/server/repository"
)

var ErrMetricNotFound = errors.New("metric not found")

type metricsLoader interface {
	Load(metricType, name string) (model.Metrics, error)
}

type Handler struct {
	repo metricsLoader
}

func New(repo metricsLoader) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) Execute(metricType, name string) (model.Metrics, error) {
	metric, err := h.repo.Load(metricType, name)
	if err != nil {
		if errors.Is(err, repository.ErrMetricNotFound) {
			return model.Metrics{}, ErrMetricNotFound
		}

		return model.Metrics{}, err
	}

	return metric, nil
}
