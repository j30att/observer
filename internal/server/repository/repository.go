package repository

import (
	"errors"
	"j30att/observer/internal/server/model"
	"sync"
)

var ErrMetricNotFound = errors.New("metric not found")

type MetricsRepository interface {
	SaveGauge(name string, value float64) error
	SaveCounter(name string, delta int64) error
	Load(metricType, name string) (model.Metrics, error)
}

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
