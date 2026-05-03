package collectors

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"j30att/observer/internal/agent/model"
	"j30att/observer/internal/agent/repository"
)

func TestRuntimeCollector(t *testing.T) {
	var (
		store     *repository.MetricsRepository
		collector *RuntimeCollector
	)

	setup := func(t *testing.T) {
		t.Helper()

		store = repository.NewMetricsRepository()
		collector = &RuntimeCollector{
			readMemStats: func(_ *runtime.MemStats) {},
			randomValue:  func() float64 { return 0.1 },
		}
	}

	t.Run("Тест метода Collect", func(t *testing.T) {
		t.Run("Должен собрать runtime метрики", func(t *testing.T) {
			setup(t)
			collector.readMemStats = func(stats *runtime.MemStats) {
				*stats = runtime.MemStats{
					Alloc: 1, BuckHashSys: 2, Frees: 3, GCCPUFraction: 0.25, GCSys: 4,
					HeapAlloc: 5, HeapIdle: 6, HeapInuse: 7, HeapObjects: 8, HeapReleased: 9,
					HeapSys: 10, LastGC: 11, Lookups: 12, MCacheInuse: 13, MCacheSys: 14,
					MSpanInuse: 15, MSpanSys: 16, Mallocs: 17, NextGC: 18, NumForcedGC: 19,
					NumGC: 20, OtherSys: 21, PauseTotalNs: 22, StackInuse: 23, StackSys: 24,
					Sys: 25, TotalAlloc: 26,
				}
			}
			collector.randomValue = func() float64 { return 0.75 }

			err := collector.Collect(store)

			require.NoError(t, err)
			snapshot := store.Snapshot()
			assert.Len(t, snapshot.Gauges, 28)
			assert.Equal(t, float64(1), snapshot.Gauges["Alloc"])
			assert.Equal(t, float64(2), snapshot.Gauges["BuckHashSys"])
			assert.Equal(t, float64(3), snapshot.Gauges["Frees"])
			assert.Equal(t, 0.25, snapshot.Gauges["GCCPUFraction"])
			assert.Equal(t, float64(4), snapshot.Gauges["GCSys"])
			assert.Equal(t, float64(5), snapshot.Gauges["HeapAlloc"])
			assert.Equal(t, float64(6), snapshot.Gauges["HeapIdle"])
			assert.Equal(t, float64(7), snapshot.Gauges["HeapInuse"])
			assert.Equal(t, float64(8), snapshot.Gauges["HeapObjects"])
			assert.Equal(t, float64(9), snapshot.Gauges["HeapReleased"])
			assert.Equal(t, float64(10), snapshot.Gauges["HeapSys"])
			assert.Equal(t, float64(11), snapshot.Gauges["LastGC"])
			assert.Equal(t, float64(12), snapshot.Gauges["Lookups"])
			assert.Equal(t, float64(13), snapshot.Gauges["MCacheInuse"])
			assert.Equal(t, float64(14), snapshot.Gauges["MCacheSys"])
			assert.Equal(t, float64(15), snapshot.Gauges["MSpanInuse"])
			assert.Equal(t, float64(16), snapshot.Gauges["MSpanSys"])
			assert.Equal(t, float64(17), snapshot.Gauges["Mallocs"])
			assert.Equal(t, float64(18), snapshot.Gauges["NextGC"])
			assert.Equal(t, float64(19), snapshot.Gauges["NumForcedGC"])
			assert.Equal(t, float64(20), snapshot.Gauges["NumGC"])
			assert.Equal(t, float64(21), snapshot.Gauges["OtherSys"])
			assert.Equal(t, float64(22), snapshot.Gauges["PauseTotalNs"])
			assert.Equal(t, float64(23), snapshot.Gauges["StackInuse"])
			assert.Equal(t, float64(24), snapshot.Gauges["StackSys"])
			assert.Equal(t, float64(25), snapshot.Gauges["Sys"])
			assert.Equal(t, float64(26), snapshot.Gauges["TotalAlloc"])
			assert.Equal(t, 0.75, snapshot.Gauges[model.RandomValueMetric])
			assert.EqualValues(t, 1, snapshot.Counters[model.PollCountMetric])
		})

		t.Run("Должен увеличивать PollCount на каждый collect", func(t *testing.T) {
			setup(t)

			require.NoError(t, collector.Collect(store))
			require.NoError(t, collector.Collect(store))

			snapshot := store.Snapshot()

			assert.EqualValues(t, 2, snapshot.Counters[model.PollCountMetric])
		})

		t.Run("Должен обновить RandomValue", func(t *testing.T) {
			setup(t)
			collector.randomValue = func() float64 { return 0.33 }

			err := collector.Collect(store)

			require.NoError(t, err)
			snapshot := store.Snapshot()
			assert.Equal(t, 0.33, snapshot.Gauges[model.RandomValueMetric])
		})
	})
}
