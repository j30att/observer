package storage

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"j30att/observer/internal/server/model"
)

// MetricsLister lists the current metrics snapshot.
type MetricsLister interface {
	List(ctx context.Context) []model.Metrics
}

// MetricsSaver persists a metrics snapshot.
type MetricsSaver interface {
	Save(metrics []model.Metrics) error
}

// PeriodicSaver periodically persists metrics from a source to a target.
type PeriodicSaver struct {
	interval time.Duration
	source   MetricsLister
	target   MetricsSaver
	logger   zerolog.Logger
}

// NewPeriodicSaver creates a periodic metrics snapshot saver.
func NewPeriodicSaver(interval time.Duration, source MetricsLister, target MetricsSaver, logger zerolog.Logger) *PeriodicSaver {
	return &PeriodicSaver{
		interval: interval,
		source:   source,
		target:   target,
		logger:   logger,
	}
}

// Save immediately persists the current source snapshot.
func (s *PeriodicSaver) Save(ctx context.Context) error {
	return s.target.Save(s.source.List(ctx))
}

// Run saves metrics on each interval tick until the context is cancelled.
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
			if err := s.Save(ctx); err != nil {
				s.logger.Error().Err(err).Msg("failed to save metrics")
			}
		}
	}
}
