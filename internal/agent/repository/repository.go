package repository

import (
	"sync"

	"j30att/observer/internal/agent/model"
)

type MetricsRepository struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

func NewMetricsRepository() *MetricsRepository {
	return &MetricsRepository{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (r *MetricsRepository) SaveGauge(name string, value float64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.gauges[name] = value
}

func (r *MetricsRepository) SaveCounter(name string, delta int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.counters[name] += delta
}

func (r *MetricsRepository) Snapshot() model.MetricsSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	snapshot := model.MetricsSnapshot{
		Gauges:   make(map[string]float64, len(r.gauges)),
		Counters: make(map[string]int64, len(r.counters)),
	}
	for name, value := range r.gauges {
		snapshot.Gauges[name] = value
	}

	for name, value := range r.counters {
		snapshot.Counters[name] = value
	}

	return snapshot
}
