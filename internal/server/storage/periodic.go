package storage

import (
	"context"
	"time"

	"github.com/rs/zerolog"
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
	logger   zerolog.Logger
}

func NewPeriodicSaver(interval time.Duration, source MetricsLister, target MetricsSaver, logger zerolog.Logger) *PeriodicSaver {
	return &PeriodicSaver{
		interval: interval,
		source:   source,
		target:   target,
		logger:   logger,
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
			if err := s.Save(); err != nil {
				s.logger.Error().Err(err).Msg("failed to save metrics")
			}
		}
	}
}
