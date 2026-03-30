package handler

import (
	"errors"
	"j30att/observer/internal/server/model"
	"strconv"

	"j30att/observer/internal/server/repository"
)

var ErrUnsupportedMetricType = errors.New("unsupported metric type")

type UpdateMetricHandler struct {
	repo repository.MetricsRepository
}

func NewUpdateMetricHandler(repo repository.MetricsRepository) *UpdateMetricHandler {
	return &UpdateMetricHandler{repo: repo}
}

func (c *UpdateMetricHandler) Execute(metricType, name, rawValue string) error {
	switch metricType {
	case model.Gauge:
		value, err := strconv.ParseFloat(rawValue, 64)
		if err != nil {
			return err
		}

		return c.repo.SaveGauge(name, value)
	case model.Counter:
		delta, err := strconv.ParseInt(rawValue, 10, 64)
		if err != nil {
			return err
		}

		return c.repo.SaveCounter(name, delta)
	default:
		return ErrUnsupportedMetricType
	}
}
