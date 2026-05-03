package repository

import "j30att/observer/internal/server/model"

type MetricsRepository interface {
	SaveGauge(name string, value float64) error
	SaveCounter(name string, delta int64) error
	SaveBatch(metrics []model.Metrics) error
	Load(metricType, name string) (model.Metrics, error)
	List() []model.Metrics
}

type MetricsSaver interface {
	Save(metrics []model.Metrics) error
}

type SyncPersistentRepository struct {
	repo  MetricsRepository
	saver MetricsSaver
}

func NewSyncPersistentRepository(repo MetricsRepository, saver MetricsSaver) *SyncPersistentRepository {
	return &SyncPersistentRepository{
		repo:  repo,
		saver: saver,
	}
}

func (r *SyncPersistentRepository) SaveGauge(name string, value float64) error {
	if err := r.repo.SaveGauge(name, value); err != nil {
		return err
	}

	return r.saver.Save(r.repo.List())
}

func (r *SyncPersistentRepository) SaveCounter(name string, delta int64) error {
	if err := r.repo.SaveCounter(name, delta); err != nil {
		return err
	}

	return r.saver.Save(r.repo.List())
}

func (r *SyncPersistentRepository) SaveBatch(metrics []model.Metrics) error {
	if err := r.repo.SaveBatch(metrics); err != nil {
		return err
	}

	return r.saver.Save(r.repo.List())
}

func (r *SyncPersistentRepository) Load(metricType, name string) (model.Metrics, error) {
	return r.repo.Load(metricType, name)
}

func (r *SyncPersistentRepository) List() []model.Metrics {
	return r.repo.List()
}
