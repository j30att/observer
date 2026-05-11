package update

import (
	"context"
	"errors"
	"strconv"

	"j30att/observer/internal/server/model"
)

var ErrUnsupportedMetricType = errors.New("unsupported metric type")

type metricsUpdater interface {
	SaveGauge(ctx context.Context, name string, value float64) error
	SaveCounter(ctx context.Context, name string, delta int64) error
	SaveBatch(ctx context.Context, metrics []model.Metrics) error
}

type Handler struct {
	repo metricsUpdater
}

func New(repo metricsUpdater) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) Execute(ctx context.Context, metricType, name, rawValue string) error {
	switch metricType {
	case model.Gauge:
		value, err := strconv.ParseFloat(rawValue, 64)
		if err != nil {
			return err
		}

		return h.repo.SaveGauge(ctx, name, value)
	case model.Counter:
		delta, err := strconv.ParseInt(rawValue, 10, 64)
		if err != nil {
			return err
		}

		return h.repo.SaveCounter(ctx, name, delta)
	default:
		return ErrUnsupportedMetricType
	}
}

func (h *Handler) ExecuteBatch(ctx context.Context, metrics []model.Metrics) error {
	for _, metric := range metrics {
		if err := validateMetric(metric); err != nil {
			return err
		}
	}

	return h.repo.SaveBatch(ctx, metrics)
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
