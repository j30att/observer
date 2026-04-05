package update_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"j30att/observer/internal/server/handlers/update"
	"j30att/observer/internal/server/model"
	"j30att/observer/internal/server/repository"
)

func TestExecuteSavesGauge(t *testing.T) {
	repo := repository.NewMetricsRepository()
	command := update.New(repo)

	err := command.Execute(model.Gauge, "Alloc", "12.5")
	require.NoError(t, err)

	metric, err := repo.Load(model.Gauge, "Alloc")
	require.NoError(t, err)

	require.NotNil(t, metric.Value)
	require.Equal(t, 12.5, *metric.Value)
}

func TestExecuteAccumulatesCounter(t *testing.T) {
	repo := repository.NewMetricsRepository()
	command := update.New(repo)

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
	command := update.New(repo)

	err := command.Execute("summary", "Alloc", "12.5")
	require.Error(t, err)
	require.ErrorIs(t, err, update.ErrUnsupportedMetricType)
}

func TestExecuteReturnsErrorForInvalidCounterValue(t *testing.T) {
	repo := repository.NewMetricsRepository()
	command := update.New(repo)

	err := command.Execute(model.Counter, "PollCount", "abc")
	require.Error(t, err)
}
