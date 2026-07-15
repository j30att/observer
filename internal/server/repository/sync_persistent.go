package repository

import (
	"context"

	"j30att/observer/internal/server/model"
)

// MetricsRepository describes the metric storage operations used by decorators.
type MetricsRepository interface {
	SaveGauge(ctx context.Context, name string, value float64) error
	SaveCounter(ctx context.Context, name string, delta int64) error
	SaveBatch(ctx context.Context, metrics []model.Metrics) error
	Load(ctx context.Context, metricType, name string) (model.Metrics, error)
	List(ctx context.Context) []model.Metrics
}

// MetricsSaver persists a full metrics snapshot.
type MetricsSaver interface {
	Save(metrics []model.Metrics) error
}

// SyncPersistentRepository saves the full metrics snapshot after each mutation.
type SyncPersistentRepository struct {
	repo  MetricsRepository
	saver MetricsSaver
}

// NewSyncPersistentRepository wraps repo with synchronous persistence through saver.
func NewSyncPersistentRepository(repo MetricsRepository, saver MetricsSaver) *SyncPersistentRepository {
	return &SyncPersistentRepository{
		repo:  repo,
		saver: saver,
	}
}

// SaveGauge stores a gauge and persists the resulting metrics snapshot.
func (r *SyncPersistentRepository) SaveGauge(ctx context.Context, name string, value float64) error {
	if err := r.repo.SaveGauge(ctx, name, value); err != nil {
		return err
	}

	return r.saver.Save(r.repo.List(ctx))
}

// SaveCounter increments a counter and persists the resulting metrics snapshot.
func (r *SyncPersistentRepository) SaveCounter(ctx context.Context, name string, delta int64) error {
	if err := r.repo.SaveCounter(ctx, name, delta); err != nil {
		return err
	}

	return r.saver.Save(r.repo.List(ctx))
}

// SaveBatch stores a batch and persists the resulting metrics snapshot.
func (r *SyncPersistentRepository) SaveBatch(ctx context.Context, metrics []model.Metrics) error {
	if err := r.repo.SaveBatch(ctx, metrics); err != nil {
		return err
	}

	return r.saver.Save(r.repo.List(ctx))
}

// Load delegates metric lookup to the wrapped repository.
func (r *SyncPersistentRepository) Load(ctx context.Context, metricType, name string) (model.Metrics, error) {
	return r.repo.Load(ctx, metricType, name)
}

// List delegates metric listing to the wrapped repository.
func (r *SyncPersistentRepository) List(ctx context.Context) []model.Metrics {
	return r.repo.List(ctx)
}
