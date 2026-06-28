package repository

import (
	"strconv"
	"testing"
)

func BenchmarkMetricsRepositorySnapshot(b *testing.B) {
	repo := NewMetricsRepository()
	for i := range 100 {
		repo.SaveGauge("gauge_"+strconv.Itoa(i), float64(i))
		repo.SaveCounter("counter_"+strconv.Itoa(i), int64(i))
	}

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		_ = repo.Snapshot()
	}
}
