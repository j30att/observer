package storage

import (
	"context"
	"time"

	"j30att/observer/internal/server/model"
)

type MetricsLister interface {
	List() []model.Metrics
}

type MetricsSaver interface {
	Save(metrics []model.Metrics) error
}

type PeriodicSaver struct {
	interval time.Duration
	source   MetricsLister
	target   MetricsSaver
	onError  func(error)
}

func NewPeriodicSaver(interval time.Duration, source MetricsLister, target MetricsSaver, onError func(error)) *PeriodicSaver {
	return &PeriodicSaver{
		interval: interval,
		source:   source,
		target:   target,
		onError:  onError,
	}
}

func (s *PeriodicSaver) Save() error {
	return s.target.Save(s.source.List())
}

func (s *PeriodicSaver) Run(ctx context.Context) {
	if s.interval <= 0 {
		return
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.Save(); err != nil && s.onError != nil {
				s.onError(err)
			}
		}
	}
}
