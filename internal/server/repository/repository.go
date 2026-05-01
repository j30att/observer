package repository

import (
	"errors"
	"sort"
	"sync"

	"j30att/observer/internal/server/model"
)

var (
	ErrMetricNotFound = errors.New("metric not found")
	ErrInvalidMetric  = errors.New("invalid metric")
)

type InMemoryMetricsRepository struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

func NewMetricsRepository() *InMemoryMetricsRepository {
	return &InMemoryMetricsRepository{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (r *InMemoryMetricsRepository) SaveGauge(name string, value float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.gauges[name] = value
	return nil
}

func (r *InMemoryMetricsRepository) SaveCounter(name string, delta int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.counters[name] += delta
	return nil
}

func (r *InMemoryMetricsRepository) Load(metricType, name string) (model.Metrics, error) {
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

func (r *InMemoryMetricsRepository) Restore(metrics []model.Metrics) error {
	gauges := make(map[string]float64)
	counters := make(map[string]int64)

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

func (r *InMemoryMetricsRepository) List() []model.Metrics {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metrics := make([]model.Metrics, 0, len(r.gauges)+len(r.counters))

	for name, value := range r.gauges {
		value := value
		metrics = append(metrics, model.Metrics{
			ID:    name,
			MType: model.Gauge,
			Value: &value,
		})
	}

	for name, delta := range r.counters {
		delta := delta
		metrics = append(metrics, model.Metrics{
			ID:    name,
			MType: model.Counter,
			Delta: &delta,
		})
	}

	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].ID < metrics[j].ID
	})

	return metrics
}
