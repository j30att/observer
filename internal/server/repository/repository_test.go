package repository_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"j30att/observer/internal/server/model"
	"j30att/observer/internal/server/repository"
)

func TestSaveGaugeAndLoadReturnsStoredValue(t *testing.T) {
	repo := repository.NewMetricsRepository()

	err := repo.SaveGauge("Alloc", 12.5)
	require.NoError(t, err)

	metric, err := repo.Load(model.Gauge, "Alloc")
	require.NoError(t, err)

	require.Equal(t, "Alloc", metric.ID)
	require.Equal(t, model.Gauge, metric.MType)
	require.NotNil(t, metric.Value)
	require.Equal(t, 12.5, *metric.Value)
}

func TestSaveCounterAccumulatesValue(t *testing.T) {
	repo := repository.NewMetricsRepository()

	err := repo.SaveCounter("PollCount", 2)
	require.NoError(t, err)

	err = repo.SaveCounter("PollCount", 3)
	require.NoError(t, err)

	metric, err := repo.Load(model.Counter, "PollCount")
	require.NoError(t, err)

	require.NotNil(t, metric.Delta)
	require.EqualValues(t, 5, *metric.Delta)
}

func TestLoadReturnsErrMetricNotFoundForMissingGauge(t *testing.T) {
	repo := repository.NewMetricsRepository()

	_, err := repo.Load(model.Gauge, "UnknownMetric")
	require.Error(t, err)
	require.ErrorIs(t, err, repository.ErrMetricNotFound)
}

func TestLoadReturnsErrMetricNotFoundForUnsupportedType(t *testing.T) {
	repo := repository.NewMetricsRepository()

	_, err := repo.Load("summary", "Alloc")
	require.Error(t, err)
	require.ErrorIs(t, err, repository.ErrMetricNotFound)
}

func TestListReturnsAllSavedMetrics(t *testing.T) {
	repo := repository.NewMetricsRepository()

	err := repo.SaveGauge("Alloc", 12.5)
	require.NoError(t, err)

	err = repo.SaveCounter("PollCount", 3)
	require.NoError(t, err)

	metrics := repo.List()
	require.Len(t, metrics, 2)
}

func TestRestoreLoadsMetricsSnapshot(t *testing.T) {
	repo := repository.NewMetricsRepository()

	value := 12.5
	delta := int64(42)
	err := repo.Restore([]model.Metrics{
		{
			ID:    "Alloc",
			MType: model.Gauge,
			Value: &value,
		},
		{
			ID:    "PollCount",
			MType: model.Counter,
			Delta: &delta,
		},
	})
	require.NoError(t, err)

	gauge, err := repo.Load(model.Gauge, "Alloc")
	require.NoError(t, err)
	require.NotNil(t, gauge.Value)
	require.Equal(t, value, *gauge.Value)

	counter, err := repo.Load(model.Counter, "PollCount")
	require.NoError(t, err)
	require.NotNil(t, counter.Delta)
	require.Equal(t, delta, *counter.Delta)
}

func TestRestoreReplacesCurrentMetrics(t *testing.T) {
	repo := repository.NewMetricsRepository()

	require.NoError(t, repo.SaveGauge("Alloc", 12.5))
	require.NoError(t, repo.SaveCounter("PollCount", 42))

	value := 7.5
	err := repo.Restore([]model.Metrics{
		{
			ID:    "HeapAlloc",
			MType: model.Gauge,
			Value: &value,
		},
	})
	require.NoError(t, err)

	_, err = repo.Load(model.Gauge, "Alloc")
	require.ErrorIs(t, err, repository.ErrMetricNotFound)

	_, err = repo.Load(model.Counter, "PollCount")
	require.ErrorIs(t, err, repository.ErrMetricNotFound)

	metric, err := repo.Load(model.Gauge, "HeapAlloc")
	require.NoError(t, err)
	require.NotNil(t, metric.Value)
	require.Equal(t, value, *metric.Value)
}

func TestRestoreReturnsErrorForInvalidMetric(t *testing.T) {
	repo := repository.NewMetricsRepository()

	err := repo.Restore([]model.Metrics{
		{
			ID:    "Alloc",
			MType: model.Gauge,
		},
	})

	require.ErrorIs(t, err, repository.ErrInvalidMetric)
}

func TestRestoreKeepsCurrentMetricsWhenSnapshotIsInvalid(t *testing.T) {
	repo := repository.NewMetricsRepository()
	require.NoError(t, repo.SaveGauge("Alloc", 12.5))

	err := repo.Restore([]model.Metrics{
		{
			ID:    "Alloc",
			MType: "summary",
		},
	})
	require.ErrorIs(t, err, repository.ErrInvalidMetric)

	metric, err := repo.Load(model.Gauge, "Alloc")
	require.NoError(t, err)
	require.NotNil(t, metric.Value)
	require.Equal(t, 12.5, *metric.Value)
}
