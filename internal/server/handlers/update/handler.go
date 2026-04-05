package update

import (
	"errors"
	"strconv"

	"j30att/observer/internal/server/model"
)

var ErrUnsupportedMetricType = errors.New("unsupported metric type")

type metricsUpdater interface {
	SaveGauge(name string, value float64) error
	SaveCounter(name string, delta int64) error
}

type Handler struct {
	repo metricsUpdater
}

func New(repo metricsUpdater) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) Execute(metricType, name, rawValue string) error {
	switch metricType {
	case model.Gauge:
		value, err := strconv.ParseFloat(rawValue, 64)
		if err != nil {
			return err
		}

		return h.repo.SaveGauge(name, value)
	case model.Counter:
		delta, err := strconv.ParseInt(rawValue, 10, 64)
		if err != nil {
			return err
		}

		return h.repo.SaveCounter(name, delta)
	default:
		return ErrUnsupportedMetricType
	}
}
