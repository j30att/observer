package collectors

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"j30att/observer/internal/agent/model"
	"j30att/observer/internal/agent/repository"
)

func TestRuntimeCollectorCollectsRuntimeMetrics(t *testing.T) {
	store := repository.NewMetricsRepository()
	collector := &RuntimeCollector{
		readMemStats: func(stats *runtime.MemStats) {
			*stats = runtime.MemStats{
				Alloc:         1,
				BuckHashSys:   2,
				Frees:         3,
				GCCPUFraction: 0.25,
				GCSys:         4,
				HeapAlloc:     5,
				HeapIdle:      6,
				HeapInuse:     7,
				HeapObjects:   8,
				HeapReleased:  9,
				HeapSys:       10,
				LastGC:        11,
				Lookups:       12,
				MCacheInuse:   13,
				MCacheSys:     14,
				MSpanInuse:    15,
				MSpanSys:      16,
				Mallocs:       17,
				NextGC:        18,
				NumForcedGC:   19,
				NumGC:         20,
				OtherSys:      21,
				PauseTotalNs:  22,
				StackInuse:    23,
				StackSys:      24,
				Sys:           25,
				TotalAlloc:    26,
			}
		},
		randomValue: func() float64 { return 0.75 },
	}

	err := collector.Collect(store)
	require.NoError(t, err)

	snapshot := store.Snapshot()
	require.Len(t, snapshot.Gauges, 28)
	require.Equal(t, float64(1), snapshot.Gauges["Alloc"])
	require.Equal(t, float64(2), snapshot.Gauges["BuckHashSys"])
	require.Equal(t, float64(3), snapshot.Gauges["Frees"])
	require.Equal(t, 0.25, snapshot.Gauges["GCCPUFraction"])
	require.Equal(t, float64(4), snapshot.Gauges["GCSys"])
	require.Equal(t, float64(5), snapshot.Gauges["HeapAlloc"])
	require.Equal(t, float64(6), snapshot.Gauges["HeapIdle"])
	require.Equal(t, float64(7), snapshot.Gauges["HeapInuse"])
	require.Equal(t, float64(8), snapshot.Gauges["HeapObjects"])
	require.Equal(t, float64(9), snapshot.Gauges["HeapReleased"])
	require.Equal(t, float64(10), snapshot.Gauges["HeapSys"])
	require.Equal(t, float64(11), snapshot.Gauges["LastGC"])
	require.Equal(t, float64(12), snapshot.Gauges["Lookups"])
	require.Equal(t, float64(13), snapshot.Gauges["MCacheInuse"])
	require.Equal(t, float64(14), snapshot.Gauges["MCacheSys"])
	require.Equal(t, float64(15), snapshot.Gauges["MSpanInuse"])
	require.Equal(t, float64(16), snapshot.Gauges["MSpanSys"])
	require.Equal(t, float64(17), snapshot.Gauges["Mallocs"])
	require.Equal(t, float64(18), snapshot.Gauges["NextGC"])
	require.Equal(t, float64(19), snapshot.Gauges["NumForcedGC"])
	require.Equal(t, float64(20), snapshot.Gauges["NumGC"])
	require.Equal(t, float64(21), snapshot.Gauges["OtherSys"])
	require.Equal(t, float64(22), snapshot.Gauges["PauseTotalNs"])
	require.Equal(t, float64(23), snapshot.Gauges["StackInuse"])
	require.Equal(t, float64(24), snapshot.Gauges["StackSys"])
	require.Equal(t, float64(25), snapshot.Gauges["Sys"])
	require.Equal(t, float64(26), snapshot.Gauges["TotalAlloc"])
	require.Equal(t, 0.75, snapshot.Gauges[model.RandomValueMetric])
	require.EqualValues(t, 1, snapshot.Counters[model.PollCountMetric])
}

func TestRuntimeCollectorIncrementsPollCountOnEveryCollect(t *testing.T) {
	store := repository.NewMetricsRepository()
	collector := &RuntimeCollector{
		readMemStats: func(_ *runtime.MemStats) {},
		randomValue:  func() float64 { return 0.1 },
	}

	err := collector.Collect(store)
	require.NoError(t, err)

	err = collector.Collect(store)
	require.NoError(t, err)

	snapshot := store.Snapshot()
	require.EqualValues(t, 2, snapshot.Counters[model.PollCountMetric])
}

func TestRuntimeCollectorUpdatesRandomValueOnCollect(t *testing.T) {
	store := repository.NewMetricsRepository()
	collector := &RuntimeCollector{
		readMemStats: func(_ *runtime.MemStats) {},
		randomValue:  func() float64 { return 0.33 },
	}

	err := collector.Collect(store)
	require.NoError(t, err)

	snapshot := store.Snapshot()
	require.Equal(t, 0.33, snapshot.Gauges[model.RandomValueMetric])
}
