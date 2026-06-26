package collectors

import (
	"errors"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v3/mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"j30att/observer/internal/agent/repository"
)

func TestGopsutilCollector(t *testing.T) {
	t.Run("Должен собрать память и utilization для каждого CPU", func(t *testing.T) {
		store := repository.NewMetricsRepository()
		collector := &GopsutilCollector{
			virtualMemory: func() (*mem.VirtualMemoryStat, error) {
				return &mem.VirtualMemoryStat{
					Total: 1000,
					Free:  250,
				}, nil
			},
			cpuPercent: func(interval time.Duration, percpu bool) ([]float64, error) {
				assert.Zero(t, interval)
				assert.True(t, percpu)
				return []float64{10.5, 20.5, 30.5}, nil
			},
		}

		err := collector.Collect(store)

		require.NoError(t, err)
		snapshot := store.Snapshot()
		assert.Equal(t, float64(1000), snapshot.Gauges["TotalMemory"])
		assert.Equal(t, float64(250), snapshot.Gauges["FreeMemory"])
		assert.Equal(t, 10.5, snapshot.Gauges["CPUutilization1"])
		assert.Equal(t, 20.5, snapshot.Gauges["CPUutilization2"])
		assert.Equal(t, 30.5, snapshot.Gauges["CPUutilization3"])
		assert.NotContains(t, snapshot.Gauges, "CPUutilization4")
	})

	t.Run("Должен вернуть ошибку сбора памяти", func(t *testing.T) {
		expectedErr := errors.New("memory unavailable")
		collector := &GopsutilCollector{
			virtualMemory: func() (*mem.VirtualMemoryStat, error) {
				return nil, expectedErr
			},
			cpuPercent: func(time.Duration, bool) ([]float64, error) {
				return []float64{10.5}, nil
			},
		}

		err := collector.Collect(repository.NewMetricsRepository())

		require.ErrorIs(t, err, expectedErr)
		require.ErrorContains(t, err, "collect virtual memory metrics")
	})

	t.Run("Должен вернуть ошибку сбора CPU", func(t *testing.T) {
		expectedErr := errors.New("cpu unavailable")
		collector := &GopsutilCollector{
			virtualMemory: func() (*mem.VirtualMemoryStat, error) {
				return &mem.VirtualMemoryStat{}, nil
			},
			cpuPercent: func(time.Duration, bool) ([]float64, error) {
				return nil, expectedErr
			},
		}

		err := collector.Collect(repository.NewMetricsRepository())

		require.ErrorIs(t, err, expectedErr)
		require.ErrorContains(t, err, "collect cpu utilization metrics")
	})
}
