package collectors

import (
	"math/rand"
	"runtime"

	"j30att/observer/internal/agent/model"
	"j30att/observer/internal/agent/repository"
)

type runtimeGaugeMetric struct {
	name  string
	value func(stats runtime.MemStats) float64
}

var runtimeGaugeMetrics = []runtimeGaugeMetric{
	{name: "Alloc", value: func(stats runtime.MemStats) float64 { return float64(stats.Alloc) }},
	{name: "BuckHashSys", value: func(stats runtime.MemStats) float64 { return float64(stats.BuckHashSys) }},
	{name: "Frees", value: func(stats runtime.MemStats) float64 { return float64(stats.Frees) }},
	{name: "GCCPUFraction", value: func(stats runtime.MemStats) float64 { return stats.GCCPUFraction }},
	{name: "GCSys", value: func(stats runtime.MemStats) float64 { return float64(stats.GCSys) }},
	{name: "HeapAlloc", value: func(stats runtime.MemStats) float64 { return float64(stats.HeapAlloc) }},
	{name: "HeapIdle", value: func(stats runtime.MemStats) float64 { return float64(stats.HeapIdle) }},
	{name: "HeapInuse", value: func(stats runtime.MemStats) float64 { return float64(stats.HeapInuse) }},
	{name: "HeapObjects", value: func(stats runtime.MemStats) float64 { return float64(stats.HeapObjects) }},
	{name: "HeapReleased", value: func(stats runtime.MemStats) float64 { return float64(stats.HeapReleased) }},
	{name: "HeapSys", value: func(stats runtime.MemStats) float64 { return float64(stats.HeapSys) }},
	{name: "LastGC", value: func(stats runtime.MemStats) float64 { return float64(stats.LastGC) }},
	{name: "Lookups", value: func(stats runtime.MemStats) float64 { return float64(stats.Lookups) }},
	{name: "MCacheInuse", value: func(stats runtime.MemStats) float64 { return float64(stats.MCacheInuse) }},
	{name: "MCacheSys", value: func(stats runtime.MemStats) float64 { return float64(stats.MCacheSys) }},
	{name: "MSpanInuse", value: func(stats runtime.MemStats) float64 { return float64(stats.MSpanInuse) }},
	{name: "MSpanSys", value: func(stats runtime.MemStats) float64 { return float64(stats.MSpanSys) }},
	{name: "Mallocs", value: func(stats runtime.MemStats) float64 { return float64(stats.Mallocs) }},
	{name: "NextGC", value: func(stats runtime.MemStats) float64 { return float64(stats.NextGC) }},
	{name: "NumForcedGC", value: func(stats runtime.MemStats) float64 { return float64(stats.NumForcedGC) }},
	{name: "NumGC", value: func(stats runtime.MemStats) float64 { return float64(stats.NumGC) }},
	{name: "OtherSys", value: func(stats runtime.MemStats) float64 { return float64(stats.OtherSys) }},
	{name: "PauseTotalNs", value: func(stats runtime.MemStats) float64 { return float64(stats.PauseTotalNs) }},
	{name: "StackInuse", value: func(stats runtime.MemStats) float64 { return float64(stats.StackInuse) }},
	{name: "StackSys", value: func(stats runtime.MemStats) float64 { return float64(stats.StackSys) }},
	{name: "Sys", value: func(stats runtime.MemStats) float64 { return float64(stats.Sys) }},
	{name: "TotalAlloc", value: func(stats runtime.MemStats) float64 { return float64(stats.TotalAlloc) }},
}

// RuntimeCollector collects metrics from runtime.MemStats.
type RuntimeCollector struct {
	readMemStats func(stats *runtime.MemStats)
	randomValue  func() float64
}

// NewRuntimeCollector creates a collector for Go runtime memory metrics.
func NewRuntimeCollector() *RuntimeCollector {
	return &RuntimeCollector{
		readMemStats: runtime.ReadMemStats,
		randomValue:  rand.New(rand.NewSource(rand.Int63())).Float64,
	}
}

// Collect reads runtime statistics and stores them as gauge and counter metrics.
func (c *RuntimeCollector) Collect(store *repository.MetricsRepository) error {
	var stats runtime.MemStats
	c.readMemStats(&stats)

	for _, metric := range runtimeGaugeMetrics {
		store.SaveGauge(metric.name, metric.value(stats))
	}

	store.SaveGauge(model.RandomValueMetric, c.randomValue())
	store.SaveCounter(model.PollCountMetric, 1)

	return nil
}
