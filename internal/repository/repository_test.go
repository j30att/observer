package repository_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"j30att/observer/internal/model"
	"j30att/observer/internal/repository"
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
	require.True(t, errors.Is(err, repository.ErrMetricNotFound))
}

func TestLoadReturnsErrMetricNotFoundForUnsupportedType(t *testing.T) {
	repo := repository.NewMetricsRepository()

	_, err := repo.Load("summary", "Alloc")
	require.Error(t, err)
	require.True(t, errors.Is(err, repository.ErrMetricNotFound))
}
