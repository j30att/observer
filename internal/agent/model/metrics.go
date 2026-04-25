package model

const (
	CounterMetricType = "counter"
	GaugeMetricType   = "gauge"
	RandomValueMetric = "RandomValue"
	PollCountMetric   = "PollCount"
)

type MetricsSnapshot struct {
	Gauges   map[string]float64
	Counters map[string]int64
}

func NewMetricsSnapshot() MetricsSnapshot {
	return MetricsSnapshot{
		Gauges:   make(map[string]float64),
		Counters: make(map[string]int64),
	}
}
