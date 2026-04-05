package get_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"j30att/observer/internal/server/handlers/get"
	"j30att/observer/internal/server/model"
	"j30att/observer/internal/server/repository"
)

func TestGetMetricReturnsStoredGauge(t *testing.T) {
	repo := repository.NewMetricsRepository()
	err := repo.SaveGauge("Alloc", 12.5)
	require.NoError(t, err)

	query := get.New(repo)
	metric, err := query.Execute(model.Gauge, "Alloc")
	require.NoError(t, err)
	require.NotNil(t, metric.Value)
	require.Equal(t, 12.5, *metric.Value)
}

func TestGetMetricReturnsServiceNotFoundError(t *testing.T) {
	repo := repository.NewMetricsRepository()

	query := get.New(repo)
	_, err := query.Execute(model.Gauge, "Alloc")

	require.Error(t, err)
	require.True(t, errors.Is(err, get.ErrMetricNotFound))
}
