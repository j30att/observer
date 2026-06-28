package repository

import (
	"context"
	"strconv"
	"testing"
)

func BenchmarkInMemoryMetricsRepositoryList(b *testing.B) {
	ctx := context.Background()
	repo := NewMetricsRepository()
	for i := range 100 {
		if err := repo.SaveGauge(ctx, "gauge_"+strconv.Itoa(i), float64(i)); err != nil {
			b.Fatal(err)
		}
		if err := repo.SaveCounter(ctx, "counter_"+strconv.Itoa(i), int64(i)); err != nil {
			b.Fatal(err)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		_ = repo.List(ctx)
	}
}
