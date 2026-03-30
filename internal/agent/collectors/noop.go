package collectors

import "j30att/observer/internal/agent/repository"

type NoopCollector struct{}

func (NoopCollector) Collect(_ *repository.MetricsRepository) error {
	return nil
}
