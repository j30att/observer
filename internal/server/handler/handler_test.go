package handler_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"j30att/observer/internal/model"
	"j30att/observer/internal/server/handler"
	"j30att/observer/internal/server/repository"
)

func TestExecuteSavesGauge(t *testing.T) {
	repo := repository.NewMetricsRepository()
	command := handler.NewUpdateMetricHandler(repo)

	err := command.Execute(model.Gauge, "Alloc", "12.5")
	require.NoError(t, err)

	metric, err := repo.Load(model.Gauge, "Alloc")
	require.NoError(t, err)

	require.NotNil(t, metric.Value)
	require.Equal(t, 12.5, *metric.Value)
}

func TestExecuteAccumulatesCounter(t *testing.T) {
	repo := repository.NewMetricsRepository()
	command := handler.NewUpdateMetricHandler(repo)

	err := command.Execute(model.Counter, "PollCount", "2")
	require.NoError(t, err)

	err = command.Execute(model.Counter, "PollCount", "3")
	require.NoError(t, err)

	metric, err := repo.Load(model.Counter, "PollCount")
	require.NoError(t, err)

	require.NotNil(t, metric.Delta)
	require.EqualValues(t, 5, *metric.Delta)
}

func TestExecuteReturnsErrorForUnsupportedMetricType(t *testing.T) {
	repo := repository.NewMetricsRepository()
	command := handler.NewUpdateMetricHandler(repo)

	err := command.Execute("summary", "Alloc", "12.5")
	require.Error(t, err)
	require.True(t, errors.Is(err, handler.ErrUnsupportedMetricType))
}

func TestExecuteReturnsErrorForInvalidCounterValue(t *testing.T) {
	repo := repository.NewMetricsRepository()
	command := handler.NewUpdateMetricHandler(repo)

	err := command.Execute(model.Counter, "PollCount", "abc")
	require.Error(t, err)
}
