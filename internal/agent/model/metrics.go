package model

const (
	// CounterMetricType identifies monotonically increasing counter metrics.
	CounterMetricType = "counter"
	// GaugeMetricType identifies point-in-time floating-point metrics.
	GaugeMetricType = "gauge"
	// RandomValueMetric is the runtime collector's synthetic random gauge.
	RandomValueMetric = "RandomValue"
	// PollCountMetric is the runtime collector's counter of poll cycles.
	PollCountMetric = "PollCount"
)

// MetricsSnapshot is the agent-side in-memory representation of collected metrics.
type MetricsSnapshot struct {
	Gauges   map[string]float64
	Counters map[string]int64
}

// Metrics is the JSON payload shape sent by the agent to the server.
type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
}

// NewMetricsSnapshot creates an empty metrics snapshot.
func NewMetricsSnapshot() MetricsSnapshot {
	return MetricsSnapshot{
		Gauges:   make(map[string]float64),
		Counters: make(map[string]int64),
	}
}
