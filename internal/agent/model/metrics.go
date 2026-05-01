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

type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
}

func NewMetricsSnapshot() MetricsSnapshot {
	return MetricsSnapshot{
		Gauges:   make(map[string]float64),
		Counters: make(map[string]int64),
	}
}
