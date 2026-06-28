package collectors

import "j30att/observer/internal/agent/repository"

// NoopCollector implements Collector without adding any metrics.
type NoopCollector struct{}

// Collect leaves the repository unchanged.
func (NoopCollector) Collect(_ *repository.MetricsRepository) error {
	return nil
}
