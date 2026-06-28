package repository

import (
	"context"
	"errors"
	"sort"
	"sync"

	"j30att/observer/internal/server/model"
)

var (
	// ErrMetricNotFound is returned when a metric is not present in storage.
	ErrMetricNotFound = errors.New("metric not found")
	// ErrInvalidMetric is returned when a metric payload cannot be stored.
	ErrInvalidMetric = errors.New("invalid metric")
)

// InMemoryMetricsRepository stores metrics in process memory.
type InMemoryMetricsRepository struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

// NewMetricsRepository creates an empty in-memory metrics repository.
func NewMetricsRepository() *InMemoryMetricsRepository {
	return &InMemoryMetricsRepository{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

// SaveGauge stores the latest value for a gauge metric.
func (r *InMemoryMetricsRepository) SaveGauge(_ context.Context, name string, value float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.gauges[name] = value
	return nil
}

// SaveCounter increments a counter metric by delta.
func (r *InMemoryMetricsRepository) SaveCounter(_ context.Context, name string, delta int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.counters[name] += delta
	return nil
}

// SaveBatch stores a batch of gauge and counter updates atomically.
func (r *InMemoryMetricsRepository) SaveBatch(_ context.Context, metrics []model.Metrics) error {
	for _, metric := range metrics {
		if err := validateMetric(metric); err != nil {
			return err
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, metric := range metrics {
		switch metric.MType {
		case model.Gauge:
			r.gauges[metric.ID] = *metric.Value
		case model.Counter:
			r.counters[metric.ID] += *metric.Delta
		}
	}

	return nil
}

func validateMetric(metric model.Metrics) error {
	switch metric.MType {
	case model.Gauge:
		if metric.ID == "" || metric.Value == nil {
			return ErrInvalidMetric
		}
	case model.Counter:
		if metric.ID == "" || metric.Delta == nil {
			return ErrInvalidMetric
		}
	default:
		return ErrInvalidMetric
	}

	return nil
}

// Load returns one metric by type and name.
func (r *InMemoryMetricsRepository) Load(_ context.Context, metricType, name string) (model.Metrics, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	switch metricType {
	case model.Gauge:
		value, ok := r.gauges[name]
		if !ok {
			return model.Metrics{}, ErrMetricNotFound
		}

		return model.Metrics{
			ID:    name,
			MType: model.Gauge,
			Value: &value,
		}, nil
	case model.Counter:
		delta, ok := r.counters[name]
		if !ok {
			return model.Metrics{}, ErrMetricNotFound
		}

		return model.Metrics{
			ID:    name,
			MType: model.Counter,
			Delta: &delta,
		}, nil
	default:
		return model.Metrics{}, ErrMetricNotFound
	}
}

// Restore replaces all in-memory metrics with a previously saved snapshot.
func (r *InMemoryMetricsRepository) Restore(metrics []model.Metrics) error {
	gauges := make(map[string]float64, len(metrics))
	counters := make(map[string]int64, len(metrics))

	for _, metric := range metrics {
		switch metric.MType {
		case model.Gauge:
			if metric.ID == "" || metric.Value == nil {
				return ErrInvalidMetric
			}

			gauges[metric.ID] = *metric.Value
		case model.Counter:
			if metric.ID == "" || metric.Delta == nil {
				return ErrInvalidMetric
			}

			counters[metric.ID] = *metric.Delta
		default:
			return ErrInvalidMetric
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.gauges = gauges
	r.counters = counters

	return nil
}

// List returns all stored metrics ordered by metric name.
func (r *InMemoryMetricsRepository) List(_ context.Context) []model.Metrics {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metrics := make([]model.Metrics, 0, len(r.gauges)+len(r.counters))
	gaugeValues := make([]float64, len(r.gauges))
	counterDeltas := make([]int64, len(r.counters))

	i := 0
	for name, value := range r.gauges {
		gaugeValues[i] = value
		metrics = append(metrics, model.Metrics{
			ID:    name,
			MType: model.Gauge,
			Value: &gaugeValues[i],
		})
		i++
	}

	i = 0
	for name, delta := range r.counters {
		counterDeltas[i] = delta
		metrics = append(metrics, model.Metrics{
			ID:    name,
			MType: model.Counter,
			Delta: &counterDeltas[i],
		})
		i++
	}

	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].ID < metrics[j].ID
	})

	return metrics
}
