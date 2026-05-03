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
	SaveBatch(metrics []model.Metrics) error
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

func (h *Handler) ExecuteBatch(metrics []model.Metrics) error {
	for _, metric := range metrics {
		if err := validateMetric(metric); err != nil {
			return err
		}
	}

	return h.repo.SaveBatch(metrics)
}

func validateMetric(metric model.Metrics) error {
	if metric.ID == "" {
		return ErrUnsupportedMetricType
	}

	switch metric.MType {
	case model.Gauge:
		if metric.Value == nil {
			return ErrUnsupportedMetricType
		}
	case model.Counter:
		if metric.Delta == nil {
			return ErrUnsupportedMetricType
		}
	default:
		return ErrUnsupportedMetricType
	}

	return nil
}
